package httpapi

type EditNoteRequestDTO struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

var editNoteRequestFields = []string{titleField, bodyField}
