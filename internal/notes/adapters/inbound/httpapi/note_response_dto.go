package httpapi

import "time"

type NoteResponseDTO struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Archived  bool      `json:"archived"`
	RemindAt  time.Time `json:"remindAt,omitzero"`
	CreatedAt time.Time `json:"createdAt"`
	EditedAt  time.Time `json:"editedAt"`
}
