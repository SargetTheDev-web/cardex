// internal/service/validation.go

package service

import (
	"strings"
	"unicode"
)

// validatePersonOrCategoryName accepts Unicode letters separated by spaces.
// Numbers, symbols, punctuation, and other characters are rejected.
func validateAlphabeticName(name string) bool {
	name = strings.TrimSpace(name)

	if name == "" {
		return false
	}

	hasLetter := false

	for _, r := range name {
		if unicode.IsLetter(r) {
			hasLetter = true
			continue
		}

		if unicode.IsSpace(r) {
			continue
		}

		return false
	}

	return hasLetter
}
