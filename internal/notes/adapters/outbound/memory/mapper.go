package memory

import (
	"fmt"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/domain"
)

func toRecord(note domain.Note) noteRecord {
	return noteRecord{
		id:        note.ID().String(),
		title:     note.Title().String(),
		body:      note.Body().String(),
		archived:  note.IsArchived(),
		remindAt:  note.RemindAt(),
		createdAt: note.CreatedAt(),
		editedAt:  note.EditedAt(),
		version:   note.Version(),
	}
}

// toDomain wraps domain errors with %v, because a broken record is a storage
// fault, and %w would report it to the client as their mistake.
func toDomain(record noteRecord) (domain.Note, error) {
	id, err := domain.NewNoteID(record.id)
	if err != nil {
		return domain.Note{}, fmt.Errorf(damagedIDFormat, err)
	}
	title, err := domain.NewTitle(record.title)
	if err != nil {
		return domain.Note{}, fmt.Errorf(damagedTitleFormat, err)
	}
	body, err := domain.NewBody(record.body)
	if err != nil {
		return domain.Note{}, fmt.Errorf(damagedBodyFormat, err)
	}
	note, err := domain.ReconstituteNote(id, title, body, record.archived, record.remindAt, record.createdAt, record.editedAt, record.version)
	if err != nil {
		return domain.Note{}, fmt.Errorf(damagedNoteFormat, err)
	}
	return note, nil
}

func toDomains(records []noteRecord) ([]domain.Note, error) {
	notes := make([]domain.Note, 0, len(records))
	for _, record := range records {
		note, err := toDomain(record)
		if err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}
	return notes, nil
}

func toOutboxRecords(note domain.Note) []outboxRecord {
	events := note.PendingEvents()
	records := make([]outboxRecord, 0, len(events))
	for _, event := range events {
		records = append(records, outboxRecord{eventName: event.EventName(), noteID: event.NoteID().String(), occurredAt: event.OccurredAt()})
	}
	return records
}
