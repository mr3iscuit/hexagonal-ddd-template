package domain

// NoteArchived records that a note was archived.
type NoteArchived struct{ noteEvent }

func (NoteArchived) EventName() string { return noteArchivedEventName }
