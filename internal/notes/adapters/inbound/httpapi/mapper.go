package httpapi

import "github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/application"

func toCreateNoteCommand(request CreateNoteRequestDTO) application.CreateNoteCommand {
	return application.CreateNoteCommand{Title: request.Title, Body: request.Body}
}

func toEditNoteCommand(noteID string, request EditNoteRequestDTO) application.EditNoteCommand {
	return application.EditNoteCommand{NoteID: noteID, Title: request.Title, Body: request.Body}
}

func toArchiveNoteCommand(noteID string) application.ArchiveNoteCommand {
	return application.ArchiveNoteCommand{NoteID: noteID}
}

func toRestoreNoteCommand(noteID string) application.RestoreNoteCommand {
	return application.RestoreNoteCommand{NoteID: noteID}
}

func toDeleteNoteCommand(noteID string) application.DeleteNoteCommand {
	return application.DeleteNoteCommand{NoteID: noteID}
}

func toGetNoteQuery(noteID string) application.GetNoteQuery {
	return application.GetNoteQuery{NoteID: noteID}
}

func toListNotesQuery(archived bool) application.ListNotesQuery {
	return application.ListNotesQuery{Archived: archived}
}

func toNoteResponseDTO(view application.NoteViewDTO) NoteResponseDTO {
	return NoteResponseDTO{
		ID:        view.ID,
		Title:     view.Title,
		Body:      view.Body,
		Archived:  view.Archived,
		RemindAt:  view.RemindAt.UTC(),
		CreatedAt: view.CreatedAt.UTC(),
		EditedAt:  view.EditedAt.UTC(),
	}
}

func toNoteResponseDTOs(views []application.NoteViewDTO) []NoteResponseDTO {
	responses := make([]NoteResponseDTO, 0, len(views))
	for _, view := range views {
		responses = append(responses, toNoteResponseDTO(view))
	}
	return responses
}

func toNoteListResponseDTO(views []application.NoteViewDTO) NoteListResponseDTO {
	return NoteListResponseDTO{Notes: toNoteResponseDTOs(views)}
}
