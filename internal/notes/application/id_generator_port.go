package application

import "github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/domain"

// IDGeneratorPort gives each new note its identity.
type IDGeneratorPort interface {
	NextNoteID() (domain.NoteID, error)
}
