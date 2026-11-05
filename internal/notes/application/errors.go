package application

import "errors"

var (
	ErrNoteNotFound             = errors.New("note not found")
	ErrNoteModifiedConcurrently = errors.New("note was changed by another request")
	ErrNoteLimitReached         = errors.New("no more notes can be stored")
)
