package psy

// Code represents the results of a content coding decision.
type Code struct {
	Topic      string `json:"topic,omitzero"`      // Topic name
	Subtopic   string `json:"subtopic,omitzero"`   // Subtopic name (optional)
	Code       string `json:"code,omitzero"`       // Code: either a label or a number
	Confidence int    `json:"confidence,omitzero"` // Confidence level, ranging from 0-100 (optional)
	Detail     string `json:"detail,omitzero"`     // Reasoning details, if appropriate (optional)
}

// CodeSchema is a JSON Schema expressing a structured output format
var CodeSchema = map[string]any{
	"type": "array",
	"items": map[string]any{
		"type": "object",
		"properties": map[string]any{
			"topic":      map[string]any{"type": "string"},
			"subtopic":   map[string]any{"type": "string"},
			"code":       map[string]any{"type": "string"},
			"confidence": map[string]any{"type": "number"},
			"detail":     map[string]any{"type": "string"},
		},
		"additionalProperties": false,
		"required":             []string{"topic", "code"},
	},
}
