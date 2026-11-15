package psy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectCategory(t *testing.T) {
	terms := []string{"red", "green", "blue"}
	tests := []struct {
		name   string
		field  string
		terms  []string
		expect string
		ok     bool
	}{
		{"exact match", "red", terms, "red", true},
		{"last term", "blue", terms, "blue", true},
		{"no match", "yellow", terms, "", false},
		{"empty field", "", terms, "", false},
		{"nil terms", "red", nil, "", false},
		{"empty terms", "red", []string{}, "", false},
		{"case-sensitive", "Red", terms, "", false},
		{"leading punctuation", "(red", terms, "red", true},
		{"trailing punctuation", "red.", terms, "red", true},
		{"surrounding punctuation", "\"green\",", terms, "green", true},
		{"only punctuation", "...", terms, "", false},
		{"internal punctuation is kept", "re-d", terms, "", false},
		{"substring is not a match", "reddish", terms, "", false},
		{"term with internal punctuation", "e-mail,", []string{"e-mail"}, "e-mail", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := SelectCategory(tc.field, tc.terms)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.expect, got)
		})
	}
}

func TestSelectCategories(t *testing.T) {
	terms := []string{"red", "green", "blue"}
	text := "The sky is blue, the grass is green. A rose is red! Blue again: blue."
	tests := []struct {
		name   string
		text   string
		terms  []string
		sel    Selection
		expect []string
	}{
		{"first", text, terms, First, []string{"blue"}},
		{"last", text, terms, Last, []string{"blue"}},
		{"first of several", "red green blue", terms, First, []string{"red"}},
		{"last of several", "red green blue", terms, Last, []string{"blue"}},
		{"all", "red green blue", terms, All, []string{"red", "green", "blue"}},
		{"all in text order", "blue, red; green", terms, All, []string{"blue", "red", "green"}},
		{"all de-duplicated", "red red green red blue green", terms, All, []string{"red", "green", "blue"}},
		{"none selection", "red green blue", terms, None, nil},
		{"invalid selection", "red green blue", terms, Selection("bogus"), nil},
		{"empty selection", "red green blue", terms, Selection(""), nil},
		{"empty text", "", terms, First, nil},
		{"whitespace-only text", " \t\n ", terms, All, nil},
		{"nil terms", "red green", nil, All, nil},
		{"empty terms", "red green", []string{}, First, nil},
		{"no matches first", "nothing to see here", terms, First, nil},
		{"no matches last", "nothing to see here", terms, Last, nil},
		{"no matches all", "nothing to see here", terms, All, nil},
		{"multiline text", "first line\nred\n\ngreen\tblue", terms, All, []string{"red", "green", "blue"}},
		{"punctuation is trimmed", "(red), \"green\".", terms, All, []string{"red", "green"}},
		{"case-sensitive", "Red GREEN Blue", terms, All, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, SelectCategories(tc.text, tc.terms, tc.sel))
		})
	}
}

func TestSelectCategoriesDoesNotModifyInputs(t *testing.T) {
	terms := []string{"red", "green", "blue"}
	SelectCategories("red green blue", terms, Last)
	assert.Equal(t, []string{"red", "green", "blue"}, terms)
}
