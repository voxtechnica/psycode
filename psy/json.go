package psy

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// Json represents a complete structured output request/response transaction.
type Json struct {
	ID           string `json:"id,omitzero"`           // batch-unique ID
	MaxTokens    int64  `json:"maxTokens,omitzero"`    // max tokens to generate
	Model        string `json:"model,omitzero"`        // model ID
	System       string `json:"system,omitzero"`       // system prompt
	Prompt       string `json:"prompt,omitzero"`       // user message prompt
	Response     string `json:"response,omitzero"`     // assistant message response
	Codes        []Code `json:"codes,omitzero"`        // generated content codes
	ErrMsg       string `json:"error,omitzero"`        // generation error message
	StopReason   string `json:"stopReason,omitzero"`   // stop reason
	InputTokens  int64  `json:"inputTokens,omitzero"`  // input tokens used
	OutputTokens int64  `json:"outputTokens,omitzero"` // output tokens used
	Millis       int64  `json:"millis,omitzero"`       // duration in milliseconds
}

// String produces a simple text display of the Json intended for console output.
func (j *Json) String() string {
	s := "Json"
	if len(j.ID) > 0 {
		s += " ID: " + j.ID
	}
	if j.Millis > 0 {
		s += " (" + strconv.FormatInt(j.Millis, 10) + "ms)"
	}
	s += fmt.Sprintf("\nModel: %s (%d max, %d input, %d output tokens)\n", j.Model, j.MaxTokens, j.InputTokens, j.OutputTokens)
	if len(j.StopReason) > 0 {
		s += "Stop Reason: " + j.StopReason + "\n"
	}
	if len(j.ErrMsg) > 0 {
		s += "Error: " + j.ErrMsg + "\n"
	}
	if len(j.System) > 0 {
		s += "---------------------\nSystem:\n" + j.System + "\n"
	}
	if len(j.Prompt) > 0 {
		s += "---------------------\nPrompt:\n" + j.Prompt + "\n"
	}
	if len(j.Response) > 0 {
		s += "---------------------\nResponse:\n" + j.Response + "\n"
	}
	return s
}

// Generate a JSON structured output.
func (j *Json) Generate(ctx context.Context, apiClient *anthropic.Client) error {
	startTime := time.Now()
	params := anthropic.MessageNewParams{
		MaxTokens: j.MaxTokens,
		Model:     anthropic.Model(j.Model),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(j.Prompt)),
		},
		OutputConfig: anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffortHigh,
			Format: anthropic.JSONOutputFormatParam{
				Schema: CodeSchema,
			},
		},
	}
	if len(j.System) > 0 {
		params.System = []anthropic.TextBlockParam{{
			Text:         j.System,
			CacheControl: anthropic.NewCacheControlEphemeralParam(),
		}}
	}
	message, err := apiClient.Messages.New(ctx, params)
	j.Response = ""
	for _, block := range message.Content {
		if textBlock, ok := block.AsAny().(anthropic.TextBlock); ok {
			j.Response += textBlock.Text
		}
	}
	if len(j.Response) > 0 {
		e := json.Unmarshal([]byte(j.Response), &j.Codes)
		if e != nil {
			j.Response += "\nError parsing structured output: " + e.Error()
		}
	}
	j.StopReason = string(message.StopReason)
	if len(message.StopDetails.Explanation) > 0 {
		j.StopReason += " " + string(message.StopDetails.Category) + " " + message.StopDetails.Explanation
	}
	j.InputTokens = message.Usage.InputTokens
	j.OutputTokens = message.Usage.OutputTokens
	if err != nil {
		j.ErrMsg = err.Error()
	} else {
		err = json.Unmarshal([]byte(j.Response), &j.Codes)
		if err != nil {
			j.Response += "\nError parsing structured output: " + err.Error()
		}
	}
	j.Millis = time.Since(startTime).Milliseconds()
	return err
}

// Create a structured output batch request
func (j *Json) BatchRequest() anthropic.MessageBatchNewParamsRequest {
	params := anthropic.MessageBatchNewParamsRequestParams{
		MaxTokens: j.MaxTokens,
		Model:     anthropic.Model(j.Model),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(j.Prompt)),
		},
		OutputConfig: anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffortHigh,
			Format: anthropic.JSONOutputFormatParam{
				Schema: CodeSchema,
			},
		},
	}
	if len(j.System) > 0 {
		params.System = []anthropic.TextBlockParam{{
			Text:         j.System,
			CacheControl: anthropic.NewCacheControlEphemeralParam(),
		}}
	}
	return anthropic.MessageBatchNewParamsRequest{
		CustomID: j.ID,
		Params:   params,
	}
}
