package domain

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Body is the text of a Note. It may be empty, and its whitespace is kept.
type Body struct{ value string }

func NewBody(value string) (Body, error) {
	if utf8.RuneCountInString(value) > maximumBodyLength {
		return Body{}, ErrBodyTooLong
	}
	if strings.ContainsFunc(value, isForbiddenInBody) {
		return Body{}, ErrBodyHasControlCharacter
	}
	return Body{value: value}, nil
}

func isForbiddenInBody(character rune) bool {
	return unicode.Is(unicode.Cc, character) && !strings.ContainsRune(allowedBodyControlCharacters, character)
}

func (body Body) String() string { return body.value }
