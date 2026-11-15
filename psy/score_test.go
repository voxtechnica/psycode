package psy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseScore(t *testing.T) {
	valid := []struct {
		name   string
		input  string
		expect float32
	}{
		{"integer", "4", 4},
		{"zero", "0", 0},
		{"multiple digits", "100", 100},
		{"decimal", "3.5", 3.5},
		{"leading zero decimal", "0.25", 0.25},
		{"positive sign", "+5", 5},
		{"negative sign", "-2", -2},
		{"negative decimal", "-1.5", -1.5},
		{"exponent", "1e3", 1000},
		{"trailing period", "4.", 4},
		{"trailing comma", "3.5,", 3.5},
		{"trailing semicolon", "2;", 2},
		{"trailing closing paren", "5)", 5},
		{"trailing percent", "75%", 75},
		{"multiple trailing punctuation", "4.5!?", 4.5},
		{"trailing quote and period", "3\".", 3},
		{"trailing letters are trimmed", "4th", 4},
		{"underscore separator", "1_000", 1000},
	}
	for _, tc := range valid {
		t.Run("valid/"+tc.name, func(t *testing.T) {
			got, err := ParseScore(tc.input)
			if assert.NoError(t, err) {
				assert.Equal(t, tc.expect, got)
			}
		})
	}

	invalid := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"word", "four"},
		{"leading period", ".5"},
		{"leading punctuation", "(5"},
		{"leading quote", "\"5\""},
		{"sign only", "-"},
		{"plus only", "+"},
		{"sign and punctuation", "-."},

		{"fraction", "7/10"},
		{"two decimals", "1.2.3"},
		{"double sign", "--5"},
		{"hex", "0x10"},

		{"out of float32 range", "1e999"},
		{"infinity", "-Inf"},
		{"nan", "NaN"},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			_, err := ParseScore(tc.input)
			assert.Error(t, err)
		})
	}

	t.Run("invalid returns zero value for non-numeric input", func(t *testing.T) {
		got, err := ParseScore("abc")
		assert.Error(t, err)
		assert.Equal(t, float32(0), got)
	})
}

func TestSelectScores(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		sel    Selection
		expect []float32
	}{
		{"first", "Score: 4 out of 5", First, []float32{4}},
		{"last", "Score: 4 out of 5", Last, []float32{5}},
		{"all", "Score: 4 out of 5", All, []float32{4, 5}},
		{"single score first", "I give it 3.5.", First, []float32{3.5}},
		{"single score last", "I give it 3.5.", Last, []float32{3.5}},
		{"single score all", "I give it 3.5.", All, []float32{3.5}},
		{"decimals and punctuation", "I give 3.5, maybe 4.", All, []float32{3.5, 4}},
		{"negative and positive", "went from -2 to +3", All, []float32{-2, 3}},
		{"duplicates are kept", "1 1 2 1", All, []float32{1, 1, 2, 1}},
		{"all in text order", "3 1 2", All, []float32{3, 1, 2}},
		{"first in text order", "3 1 2", First, []float32{3}},
		{"last in text order", "3 1 2", Last, []float32{2}},
		{"non-numeric words skipped", "abc 7 4th 8 xyz", All, []float32{7, 4, 8}},
		{"first skips leading non-numbers", "none here, then 6", First, []float32{6}},
		{"last skips trailing non-numbers", "6 then none here", Last, []float32{6}},
		{"multiline and tabs", "line one\n9\n\n8\t7", All, []float32{9, 8, 7}},
		{"none selection", "1 2 3", None, nil},
		{"invalid selection", "1 2 3", Selection("bogus"), nil},
		{"empty selection", "1 2 3", Selection(""), nil},
		{"empty text", "", All, nil},
		{"whitespace-only text", " \t\n ", First, nil},
		{"no scores first", "no numbers at all", First, nil},
		{"no scores last", "no numbers at all", Last, nil},
		{"no scores all", "no numbers at all", All, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, SelectScores(tc.text, tc.sel))
		})
	}
}
