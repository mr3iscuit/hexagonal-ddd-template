package application

import "github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/domain"

func toNoteViewDTO(note domain.Note) NoteViewDTO {
	return NoteViewDTO{
		ID:        note.ID().String(),
		Title:     note.Title().String(),
		Body:      note.Body().String(),
		Archived:  note.IsArchived(),
		CreatedAt: note.CreatedAt(),
		EditedAt:  note.EditedAt(),
	}
}

func toNoteViewDTOs(notes []domain.Note) []NoteViewDTO {
	views := make([]NoteViewDTO, 0, len(notes))
	for _, note := range notes {
		views = append(views, toNoteViewDTO(note))
	}
	return views
}
