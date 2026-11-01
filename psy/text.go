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
	OutputFile    string    `json:"outputFile,omitzero"`    // output file name
	SystemFile    string    `json:"systemFile,omitzero"`    // system message file
	PromptFile    string    `json:"promptFile,omitzero"`    // prompt template file
	QuestionFile  string    `json:"questionFile,omitzero"`  // question template file
	QuestionField string    `json:"questionField,omitzero"` // question field name
	QuestionID    string    `json:"questionID,omitzero"`    // question ID
	AnswerFile    string    `json:"answerFile,omitzero"`    // answer template file
	AnswerField   string    `json:"answerField,omitzero"`   // answer field name
	AnswerID      string    `json:"answerID,omitzero"`      // answer ID
	ScoreField    string    `json:"scoreField,omitzero"`    // score field name
	ScoreSelect   Selection `json:"scoreSelect,omitzero"`   // score selection
	Model         string    `json:"model,omitzero"`         // model ID
	MaxTokens     int64     `json:"maxTokens,omitzero"`     // max tokens to generate
}

// Metadata returns a map of key-value pairs for the TextParameters.
func (p TextParameters) Metadata() map[string]string {
	m := make(map[string]string)
	if len(p.OutputFile) > 0 {
		m["output_file"] = p.OutputFile
	}
	if len(p.SystemFile) > 0 {
		m["system_file"] = p.SystemFile
	}
	if len(p.PromptFile) > 0 {
		m["prompt_file"] = p.PromptFile
	}
	if len(p.QuestionFile) > 0 {
		m["question_file"] = p.QuestionFile
	}
	if len(p.QuestionField) > 0 {
		m["question_field"] = p.QuestionField
	}
	if len(p.QuestionID) > 0 {
		m["question_id"] = p.QuestionID
	}
	if len(p.AnswerFile) > 0 {
		m["answer_file"] = p.AnswerFile
	}
	if len(p.AnswerField) > 0 {
		m["answer_field"] = p.AnswerField
	}
	if len(p.AnswerID) > 0 {
		m["answer_id"] = p.AnswerID
	}
	if len(p.ScoreField) > 0 {
		m["score_field"] = p.ScoreField
	}
	if p.ScoreSelect.IsValid() {
		m["score_select"] = p.ScoreSelect.String()
	}
	if len(p.Model) > 0 {
		m["model"] = p.Model
	}
	if p.MaxTokens > 0 {
		m["max_tokens"] = strconv.FormatInt(p.MaxTokens, 10)
	}
	return m
}

// Text represents a complete request/response text generation exchange.
type Text struct {
	ID           string    `json:"id,omitzero"` // batch-unique ID
	MaxTokens    int64     `json:"maxTokens,omitzero"`
	Model        string    `json:"model,omitzero"`
	System       string    `json:"system,omitzero"`
	Prompt       string    `json:"prompt,omitzero"`
	Response     string    `json:"response,omitzero"`
	Scores       []float32 `json:"scores,omitzero"`
	ErrMsg       string    `json:"error,omitzero"`
	StopReason   string    `json:"stopReason,omitzero"`
	InputTokens  int64     `json:"inputTokens,omitzero"`
	OutputTokens int64     `json:"outputTokens,omitzero"`
	Millis       int64     `json:"millis,omitzero"`
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
func GenerateText(ctx context.Context, client *anthropic.Client, text Text, sel Selection) (Text, error) {
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
		text.Scores = SelectScores(text.Response, sel)
	}
	text.Millis = time.Since(startTime).Milliseconds()
	return text, err
}

// GenerateTextParallel concurrently processes a single batch of text generations.
func GenerateTextParallel(ctx context.Context, client *anthropic.Client, texts []Text, sel Selection) map[string]Text {
	results := make(chan Text, len(texts))
	var wg sync.WaitGroup
	wg.Add(len(texts))
	for _, text := range texts {
		go func(text Text) {
			defer wg.Done()
			t, _ := GenerateText(ctx, client, text, sel)
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
