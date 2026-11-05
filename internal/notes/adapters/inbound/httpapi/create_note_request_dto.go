package httpapi

type CreateNoteRequestDTO struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

var createNoteRequestFields = []string{titleField, bodyField}
