package application

import "time"

type NoteViewDTO struct {
	ID        string
	Title     string
	Body      string
	Archived  bool
	RemindAt  time.Time
	CreatedAt time.Time
	EditedAt  time.Time
}
