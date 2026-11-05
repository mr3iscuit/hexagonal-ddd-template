package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNoteIDRequired           = errors.New("note ID is required")
	ErrTitleRequired            = errors.New("note title is required")
	ErrTitleTooLong             = fmt.Errorf("note title must be at most %d characters", maximumTitleLength)
	ErrBodyTooLong              = fmt.Errorf("note body must be at most %d characters", maximumBodyLength)
	ErrTitleHasControlCharacter = errors.New("note title must be a single line with no control characters")
	ErrBodyHasControlCharacter  = errors.New("note body must have no control characters apart from line breaks and tabs")
	ErrNoteAlreadyArchived      = errors.New("note is already archived")
	ErrNoteNotArchived          = errors.New("note is not archived")
	ErrNoteArchived             = errors.New("an archived note cannot be edited")
	ErrReminderNotInFuture      = errors.New("a reminder must be in the future")

	// These faults can never come from a client, so they have no HTTP form.
	errNoteNotCreated     = errors.New("note was never created")
	errNoteTimeMissing    = errors.New("note has no time")
	errNoteVersionInvalid = errors.New("stored note has no valid version")
)
