package domain

// NoteCreated records that a note was created.
type NoteCreated struct{ noteEvent }

func (NoteCreated) EventName() string { return noteCreatedEventName }
