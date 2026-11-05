package domain

import "strings"

// NoteID identifies a Note.
type NoteID struct{ value string }

func NewNoteID(value string) (NoteID, error) {
	if strings.TrimSpace(value) == "" {
		return NoteID{}, ErrNoteIDRequired
	}
	return NoteID{value: value}, nil
}

func (id NoteID) String() string { return id.value }

func (id NoteID) IsZero() bool { return id.value == "" }
