package psy

import (
	"slices"
	"strings"
	"unicode"
)

// SelectCategories selects the specified category term(s) from a block of text.
// Valid selections are "first", "last", "all", or "none".
// An invalid selection defaults to "none".
func SelectCategories(s string, terms []string, sel Selection) []string {
	if len(terms) == 0 || s == "" || sel == None || !sel.IsValid() {
		return nil
	}
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return nil
	}
	if sel == Last {
		// Reverse the fields
		for i, j := 0, len(fields)-1; i < j; i, j = i+1, j-1 {
			fields[i], fields[j] = fields[j], fields[i]
		}
	}
	if sel == First || sel == Last {
		// Find the first category
		for _, field := range fields {
			if c, ok := SelectCategory(field, terms); ok {
				return []string{c}
			}
		}
		return nil
	}
	// Find all the categories (de-duplicated)
	var categories []string
	for _, field := range fields {
		if c, ok := SelectCategory(field, terms); ok {
			if !slices.Contains(categories, c) {
				categories = append(categories, c)
			}
		}
	}
	return categories
}

// SelectCategory returns one of the provided category terms if the provided
// field is a match. The comparison is case-sensitive.
func SelectCategory(field string, terms []string) (string, bool) {
	// Empty is not a category
	if len(field) == 0 || len(terms) == 0 {
		return "", false
	}
	// Remove leading and trailing punctuation
	field = strings.TrimFunc(field, unicode.IsPunct)
	// Find the category term
	for _, term := range terms {
		if field == term {
			return term, true
		}
	}
	return "", false
}
