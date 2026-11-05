package application

import (
	"context"
	"time"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/domain"
)

// NoteRepositoryPort stores notes. Save and Remove reject a note that isn't at
// the version it was loaded from, and store its pending events atomically. Save
// can refuse a new note when storage has a limit.
type NoteRepositoryPort interface {
	Save(ctx context.Context, note domain.Note) error
	Remove(ctx context.Context, note domain.Note) error
	FindByID(ctx context.Context, id domain.NoteID) (domain.Note, error)
	List(ctx context.Context, archived bool) ([]domain.Note, error)
	ListDueReminders(ctx context.Context, now time.Time) ([]domain.Note, error)
}
