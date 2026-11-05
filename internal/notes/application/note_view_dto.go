package application

import "time"

type NoteViewDTO struct {
	ID        string
	Title     string
	Body      string
	Archived  bool
	CreatedAt time.Time
	EditedAt  time.Time
}
