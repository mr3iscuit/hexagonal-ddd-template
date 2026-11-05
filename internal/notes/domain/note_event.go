package domain

import "time"

// noteEvent holds what every event has in common.
type noteEvent struct {
	noteID     NoteID
	occurredAt time.Time
}

func (event noteEvent) NoteID() NoteID { return event.noteID }

func (event noteEvent) OccurredAt() time.Time { return event.occurredAt }
