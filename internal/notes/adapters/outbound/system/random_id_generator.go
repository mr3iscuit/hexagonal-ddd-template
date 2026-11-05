package system

import (
	"crypto/rand"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/domain"
)

// RandomIDGenerator makes IDs that can't be guessed.
type RandomIDGenerator struct{}

func (RandomIDGenerator) NextNoteID() (domain.NoteID, error) {
	return domain.NewNoteID(rand.Text())
}
