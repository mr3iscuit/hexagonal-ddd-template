package domain

// NoteEdited records that a note was edited.
type NoteEdited struct{ noteEvent }

func (NoteEdited) EventName() string { return noteEditedEventName }
