package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/domain"
)

func TestNoteIDIsRequired(t *testing.T) {
	for _, value := range []string{"", "   ", "\t\n"} {
		if _, err := domain.NewNoteID(value); !errors.Is(err, domain.ErrNoteIDRequired) {
			t.Errorf("NewNoteID(%q) = %v", value, err)
		}
	}
}

func TestNoteIDKeepsItsValue(t *testing.T) {
	id, err := domain.NewNoteID("groceries")
	if err != nil || id.String() != "groceries" {
		t.Errorf("id = %q, err = %v", id, err)
	}
}

func TestNoteIDKeepsTheSpaceAroundIt(t *testing.T) {
	id, err := domain.NewNoteID(" groceries ")
	if err != nil || id.String() != " groceries " {
		t.Errorf("id = %q, err = %v", id, err)
	}
}

func TestTitleIsRequired(t *testing.T) {
	for _, value := range []string{"", "   ", "\t\n"} {
		if _, err := domain.NewTitle(value); !errors.Is(err, domain.ErrTitleRequired) {
			t.Errorf("NewTitle(%q) = %v", value, err)
		}
	}
}

func TestTitleNeedsAVisibleCharacter(t *testing.T) {
	for _, value := range []string{"\u200b", "\u2060", "\ufeff", "\u00ad", "\u200b \u200b", "\u0301", "\ufe0f"} {
		if _, err := domain.NewTitle(value); !errors.Is(err, domain.ErrTitleRequired) {
			t.Errorf("NewTitle(%q) = %v", value, err)
		}
	}
}

func TestOneVisibleCharacterIsEnoughForATitle(t *testing.T) {
	for _, value := range []string{"a", "7", "!", "+", "😀", "\u200ba", "a\u0301", "👩\u200d💻"} {
		if _, err := domain.NewTitle(value); err != nil {
			t.Errorf("NewTitle(%q) = %v", value, err)
		}
	}
}

func TestTitleIsTrimmed(t *testing.T) {
	title, err := domain.NewTitle("  Groceries\n")
	if err != nil || title.String() != "Groceries" {
		t.Errorf("title = %q, err = %v", title, err)
	}
}

func TestTitleLengthLimitCountsCharactersNotBytes(t *testing.T) {
	if _, err := domain.NewTitle(strings.Repeat("ə", 200)); err != nil {
		t.Errorf("200 characters were rejected: %v", err)
	}
	if _, err := domain.NewTitle(strings.Repeat("ə", 201)); !errors.Is(err, domain.ErrTitleTooLong) {
		t.Errorf("201 characters gave %v", err)
	}
}

func TestSurroundingSpaceDoesNotCountTowardsTheTitleLimit(t *testing.T) {
	if _, err := domain.NewTitle("  " + strings.Repeat("a", 200) + "  "); err != nil {
		t.Errorf("err = %v", err)
	}
}

func TestBodyMayBeEmptyAndKeepsItsWhitespace(t *testing.T) {
	for _, value := range []string{"", "  indented\n"} {
		body, err := domain.NewBody(value)
		if err != nil || body.String() != value {
			t.Errorf("body = %q, err = %v", body, err)
		}
	}
}

func TestBodyLengthLimitCountsCharactersNotBytes(t *testing.T) {
	if _, err := domain.NewBody(strings.Repeat("ə", 10000)); err != nil {
		t.Errorf("10000 characters were rejected: %v", err)
	}
	if _, err := domain.NewBody(strings.Repeat("ə", 10001)); !errors.Is(err, domain.ErrBodyTooLong) {
		t.Errorf("10001 characters gave %v", err)
	}
}

func TestTitleIsASingleLineWithNoControlCharacters(t *testing.T) {
	for _, value := range []string{"a\nb", "a\rb", "a\tb", "a\x00b", "a\x1bb", "a\x7fb", "a\u0085b", "a\u2028b", "a\u2029b"} {
		if _, err := domain.NewTitle(value); !errors.Is(err, domain.ErrTitleHasControlCharacter) {
			t.Errorf("NewTitle(%q) = %v", value, err)
		}
	}
}

func TestTitleKeepsOrdinaryPunctuationAndSpaces(t *testing.T) {
	const value = "Ideas: a, b & c \u00a0— ə 😀"
	title, err := domain.NewTitle(value)
	if err != nil || title.String() != value {
		t.Errorf("title = %q, err = %v", title, err)
	}
}

func TestBodyKeepsLineBreaksAndTabs(t *testing.T) {
	const value = "line one\r\n\tline two\n\u2028line three"
	body, err := domain.NewBody(value)
	if err != nil || body.String() != value {
		t.Errorf("body = %q, err = %v", body, err)
	}
}

func TestBodyRefusesOtherControlCharacters(t *testing.T) {
	for _, value := range []string{"a\x00b", "a\x0bb", "a\x0cb", "a\x1bb", "a\x7fb", "a\u0085b"} {
		if _, err := domain.NewBody(value); !errors.Is(err, domain.ErrBodyHasControlCharacter) {
			t.Errorf("NewBody(%q) = %v", value, err)
		}
	}
}

func TestLengthErrorsStateTheLimit(t *testing.T) {
	if !strings.Contains(domain.ErrTitleTooLong.Error(), "200") || !strings.Contains(domain.ErrBodyTooLong.Error(), "10000") {
		t.Errorf("messages = %q, %q", domain.ErrTitleTooLong, domain.ErrBodyTooLong)
	}
}
