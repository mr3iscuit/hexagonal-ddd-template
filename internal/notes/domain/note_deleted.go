package domain

// NoteDeleted records that a note was deleted.
type NoteDeleted struct{ noteEvent }

func (NoteDeleted) EventName() string { return noteDeletedEventName }
