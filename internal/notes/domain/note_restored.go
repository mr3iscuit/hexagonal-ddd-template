package domain

// NoteRestored records that a note was restored.
type NoteRestored struct{ noteEvent }

func (NoteRestored) EventName() string { return noteRestoredEventName }
