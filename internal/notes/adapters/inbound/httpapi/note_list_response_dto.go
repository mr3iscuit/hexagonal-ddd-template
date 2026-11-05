package httpapi

type NoteListResponseDTO struct {
	Notes []NoteResponseDTO `json:"notes"`
}
