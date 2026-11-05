package domain

import "time"

// Event is a fact that happened to a Note.
type Event interface {
	EventName() string
	NoteID() NoteID
	OccurredAt() time.Time
}
