// Package memory is the in-memory storage adapter.
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/application"
	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/domain"
)

// InMemoryNoteRepository keeps notes in a map, so they are lost on restart. A
// full outbox becomes a ring, and oldestOutboxRecord is where it starts.
type InMemoryNoteRepository struct {
	mutex              sync.Mutex
	records            map[string]noteRecord
	outbox             []outboxRecord
	oldestOutboxRecord int
}

func NewInMemoryNoteRepository() *InMemoryNoteRepository {
	return &InMemoryNoteRepository{records: map[string]noteRecord{}}
}

// Save stores the note and its pending events together, or stores nothing.
func (repository *InMemoryNoteRepository) Save(ctx context.Context, note domain.Note) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	if err := repository.checkVersion(note); err != nil {
		return err
	}
	if note.Version() == 0 && len(repository.records) >= maximumNotes {
		return application.ErrNoteLimitReached
	}
	record := toRecord(note)
	record.version = note.Version() + 1
	repository.records[record.id] = record
	repository.storeEvents(note)
	return nil
}

// Remove deletes the note and stores its pending events together.
func (repository *InMemoryNoteRepository) Remove(ctx context.Context, note domain.Note) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	if note.Version() == 0 {
		return application.ErrNoteModifiedConcurrently
	}
	if err := repository.checkVersion(note); err != nil {
		return err
	}
	delete(repository.records, note.ID().String())
	repository.storeEvents(note)
	return nil
}

func (repository *InMemoryNoteRepository) FindByID(ctx context.Context, id domain.NoteID) (domain.Note, error) {
	if err := ctx.Err(); err != nil {
		return domain.Note{}, err
	}
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	record, found := repository.records[id.String()]
	if !found {
		return domain.Note{}, application.ErrNoteNotFound
	}
	return toDomain(record)
}

func (repository *InMemoryNoteRepository) List(ctx context.Context, archived bool) ([]domain.Note, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return toDomains(repository.recordsWhere(func(record noteRecord) bool {
		return record.archived == archived
	}))
}

// ListDueReminders finds the notes whose reminder has come due.
func (repository *InMemoryNoteRepository) ListDueReminders(ctx context.Context, now time.Time) ([]domain.Note, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return toDomains(repository.recordsWhere(func(record noteRecord) bool {
		return !record.remindAt.IsZero() && !record.remindAt.After(now)
	}))
}

// recordsWhere copies the matching records, so that rebuilding them for a long
// list doesn't hold the lock that every other request needs.
func (repository *InMemoryNoteRepository) recordsWhere(matches func(noteRecord) bool) []noteRecord {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	var records []noteRecord
	for _, record := range repository.records {
		if matches(record) {
			records = append(records, record)
		}
	}
	return records
}

// checkVersion refuses a note that isn't at the version it was loaded from. A
// missing record has version 0, so a note deleted meanwhile can't come back.
func (repository *InMemoryNoteRepository) checkVersion(note domain.Note) error {
	if repository.records[note.ID().String()].version != note.Version() {
		return application.ErrNoteModifiedConcurrently
	}
	return nil
}

func (repository *InMemoryNoteRepository) storeEvents(note domain.Note) {
	for _, record := range toOutboxRecords(note) {
		if len(repository.outbox) < maximumOutboxRecords {
			repository.outbox = append(repository.outbox, record)
			continue
		}
		repository.outbox[repository.oldestOutboxRecord] = record
		repository.oldestOutboxRecord = (repository.oldestOutboxRecord + 1) % maximumOutboxRecords
	}
}
