package psy

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// TextParameters represents the parameters for text prompts and completions.
type TextParameters struct {
	OutputFile     string    `json:"outputFile,omitzero"`     // output file name
	SystemFile     string    `json:"systemFile,omitzero"`     // system message file
	PromptFile     string    `json:"promptFile,omitzero"`     // prompt template file
	QuestionFile   string    `json:"questionFile,omitzero"`   // question template file
	QuestionField  string    `json:"questionField,omitzero"`  // question field name
	QuestionID     string    `json:"questionID,omitzero"`     // question ID
	AnswerFile     string    `json:"answerFile,omitzero"`     // answer template file
	AnswerID       string    `json:"answerID,omitzero"`       // answer ID
	CategoryField  string    `json:"categoryField,omitzero"`  // category field name
	CategorySelect Selection `json:"categorySelect,omitzero"` // category selection
	CategoryTerms  []string  `json:"categoryTerms,omitzero"`  // category terms
	ScoreField     string    `json:"scoreField,omitzero"`     // score field name
	ScoreSelect    Selection `json:"scoreSelect,omitzero"`    // score selection
	Model          string    `json:"model,omitzero"`          // model ID
	MaxTokens      int64     `json:"maxTokens,omitzero"`      // max tokens to generate
}

// Text represents a complete request/response text generation exchange.
type Text struct {
	ID             string    `json:"id,omitzero"`             // batch-unique ID
	MaxTokens      int64     `json:"maxTokens,omitzero"`      // max tokens to generate
	Model          string    `json:"model,omitzero"`          // model ID
	System         string    `json:"system,omitzero"`         // system prompt
	Prompt         string    `json:"prompt,omitzero"`         // user message prompt
	Response       string    `json:"response,omitzero"`       // assistant message response
	CategorySelect Selection `json:"categorySelect,omitzero"` // category selection
	CategoryTerms  []string  `json:"categoryTerms,omitzero"`  // category terms
	Categories     []string  `json:"categories,omitzero"`     // category terms identified in the response
	ScoreSelect    Selection `json:"scoreSelect,omitzero"`    // score selection
	Scores         []float32 `json:"scores,omitzero"`         // scores identified in the response
	ErrMsg         string    `json:"error,omitzero"`          // text generation error message
	StopReason     string    `json:"stopReason,omitzero"`     // stop reason
	InputTokens    int64     `json:"inputTokens,omitzero"`    // input tokens used
	OutputTokens   int64     `json:"outputTokens,omitzero"`   // output tokens used
	Millis         int64     `json:"millis,omitzero"`         // duration in milliseconds
}

// String produces a simple text display of the Text intended for console output.
func (t *Text) String() string {
	s := "Text"
	if len(t.ID) > 0 {
		s += " ID: " + t.ID
	}
	if t.Millis > 0 {
		s += " (" + strconv.FormatInt(t.Millis, 10) + "ms)"
	}
	if len(t.Scores) > 0 {
		s += " scores: " + fmt.Sprint(t.Scores)
	}
	if len(t.Categories) > 0 {
		s += " categories: " + fmt.Sprint(t.Categories)
	}
	s += fmt.Sprintf("\nModel: %s (%d max, %d input, %d output tokens)\n", t.Model, t.MaxTokens, t.InputTokens, t.OutputTokens)
	if len(t.StopReason) > 0 {
		s += "Stop Reason: " + t.StopReason + "\n"
	}
	if len(t.ErrMsg) > 0 {
		s += "Error: " + t.ErrMsg + "\n"
	}
	if len(t.System) > 0 {
		s += "---------------------\nSystem:\n" + t.System + "\n"
	}
	if len(t.Prompt) > 0 {
		s += "---------------------\nPrompt:\n" + t.Prompt + "\n"
	}
	if len(t.Response) > 0 {
		s += "---------------------\nResponse:\n" + t.Response + "\n"
	}
	return s
}

// GenerateText generates a text response based on the provided Text object and selection.
func GenerateText(ctx context.Context, client *anthropic.Client, text Text) (Text, error) {
	startTime := time.Now()
	var err error
	params := anthropic.MessageNewParams{
		MaxTokens: text.MaxTokens,
		Model:     anthropic.Model(text.Model),
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(text.Prompt))},
	}
	if len(text.System) > 0 {
		params.System = []anthropic.TextBlockParam{{
			Text:         text.System,
			CacheControl: anthropic.NewCacheControlEphemeralParam(),
		}}
	}
	message, err := client.Messages.New(ctx, params)
	text.Response = ""
	for _, block := range message.Content {
		if textBlock, ok := block.AsAny().(anthropic.TextBlock); ok {
			text.Response += textBlock.Text
		}
	}
	text.StopReason = string(message.StopReason)
	if len(message.StopDetails.Explanation) > 0 {
		text.StopReason += " " + string(message.StopDetails.Category) + " " + message.StopDetails.Explanation
	}
	text.InputTokens = message.Usage.InputTokens
	text.OutputTokens = message.Usage.OutputTokens
	if err != nil {
		text.ErrMsg = err.Error()
	} else {
		text.Categories = SelectCategories(text.Response, text.CategoryTerms, text.CategorySelect)
		text.Scores = SelectScores(text.Response, text.ScoreSelect)
	}
	text.Millis = time.Since(startTime).Milliseconds()
	return text, err
}

// GenerateTextParallel concurrently processes a single batch of text generations.
func GenerateTextParallel(ctx context.Context, client *anthropic.Client, texts []Text) map[string]Text {
	results := make(chan Text, len(texts))
	var wg sync.WaitGroup
	wg.Add(len(texts))
	for _, text := range texts {
		go func(text Text) {
			defer wg.Done()
			t, _ := GenerateText(ctx, client, text)
			results <- t
		}(text)
	}
	wg.Wait()
	close(results)
	batch := make(map[string]Text, len(texts))
	for text := range results {
		batch[text.ID] = text
	}
	return batch
}

// Batch divides the provided slice of things into batches of the specified maximum size.
func Batch[T any](items []T, batchSize int) [][]T {
	batches := make([][]T, 0)
	batch := make([]T, 0)
	for _, item := range items {
		batch = append(batch, item)
		if len(batch) == batchSize {
			batches = append(batches, batch)
			batch = make([]T, 0)
		}
	}
	if len(batch) > 0 {
		batches = append(batches, batch)
	}
	return batches
}
