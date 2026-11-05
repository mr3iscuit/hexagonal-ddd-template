package domain

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Title names a Note on a single line.
type Title struct{ value string }

func NewTitle(value string) (Title, error) {
	trimmed := strings.TrimSpace(value)
	if !strings.ContainsFunc(trimmed, isVisible) {
		return Title{}, ErrTitleRequired
	}
	if utf8.RuneCountInString(trimmed) > maximumTitleLength {
		return Title{}, ErrTitleTooLong
	}
	if strings.ContainsFunc(trimmed, isForbiddenInTitle) {
		return Title{}, ErrTitleHasControlCharacter
	}
	return Title{value: trimmed}, nil
}

// isVisible leaves out spaces, format characters and combining marks, which
// show nothing by themselves.
func isVisible(character rune) bool {
	return unicode.In(character, unicode.L, unicode.N, unicode.P, unicode.S)
}

func isForbiddenInTitle(character rune) bool {
	return unicode.In(character, unicode.Cc, unicode.Zl, unicode.Zp)
}

func (title Title) String() string { return title.value }

func (title Title) IsZero() bool { return title.value == "" }
