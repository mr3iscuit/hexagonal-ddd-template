// Package application holds the use cases and the ports they depend on.
package application

import (
	"context"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/domain"
)

// NoteApplicationService sequences each use case. Business rules stay in the
// domain.
type NoteApplicationService struct {
	noteRepository NoteRepositoryPort
	clock          ClockPort
	idGenerator    IDGeneratorPort
}

func NewNoteApplicationService(noteRepository NoteRepositoryPort, clock ClockPort, idGenerator IDGeneratorPort) NoteApplicationService {
	return NoteApplicationService{noteRepository: noteRepository, clock: clock, idGenerator: idGenerator}
}

func (service NoteApplicationService) CreateNote(ctx context.Context, command CreateNoteCommand) (NoteViewDTO, error) {
	title, err := domain.NewTitle(command.Title)
	if err != nil {
		return NoteViewDTO{}, err
	}
	body, err := domain.NewBody(command.Body)
	if err != nil {
		return NoteViewDTO{}, err
	}
	noteID, err := service.idGenerator.NextNoteID()
	if err != nil {
		return NoteViewDTO{}, err
	}
	note, err := domain.NewNote(noteID, title, body, service.clock.Now())
	if err != nil {
		return NoteViewDTO{}, err
	}
	if err := service.noteRepository.Save(ctx, note); err != nil {
		return NoteViewDTO{}, err
	}
	return toNoteViewDTO(note), nil
}

func (service NoteApplicationService) EditNote(ctx context.Context, command EditNoteCommand) (NoteViewDTO, error) {
	title, err := domain.NewTitle(command.Title)
	if err != nil {
		return NoteViewDTO{}, err
	}
	body, err := domain.NewBody(command.Body)
	if err != nil {
		return NoteViewDTO{}, err
	}
	note, err := service.load(ctx, command.NoteID)
	if err != nil {
		return NoteViewDTO{}, err
	}
	edited, err := note.Edit(title, body, service.clock.Now())
	if err != nil {
		return NoteViewDTO{}, err
	}
	if err := service.noteRepository.Save(ctx, edited); err != nil {
		return NoteViewDTO{}, err
	}
	return toNoteViewDTO(edited), nil
}

func (service NoteApplicationService) ArchiveNote(ctx context.Context, command ArchiveNoteCommand) (NoteViewDTO, error) {
	note, err := service.load(ctx, command.NoteID)
	if err != nil {
		return NoteViewDTO{}, err
	}
	archived, err := note.Archive(service.clock.Now())
	if err != nil {
		return NoteViewDTO{}, err
	}
	if err := service.noteRepository.Save(ctx, archived); err != nil {
		return NoteViewDTO{}, err
	}
	return toNoteViewDTO(archived), nil
}

func (service NoteApplicationService) RestoreNote(ctx context.Context, command RestoreNoteCommand) (NoteViewDTO, error) {
	note, err := service.load(ctx, command.NoteID)
	if err != nil {
		return NoteViewDTO{}, err
	}
	restored, err := note.Restore(service.clock.Now())
	if err != nil {
		return NoteViewDTO{}, err
	}
	if err := service.noteRepository.Save(ctx, restored); err != nil {
		return NoteViewDTO{}, err
	}
	return toNoteViewDTO(restored), nil
}

func (service NoteApplicationService) DeleteNote(ctx context.Context, command DeleteNoteCommand) error {
	note, err := service.load(ctx, command.NoteID)
	if err != nil {
		return err
	}
	deleted, err := note.Delete(service.clock.Now())
	if err != nil {
		return err
	}
	return service.noteRepository.Remove(ctx, deleted)
}

func (service NoteApplicationService) GetNote(ctx context.Context, query GetNoteQuery) (NoteViewDTO, error) {
	note, err := service.load(ctx, query.NoteID)
	if err != nil {
		return NoteViewDTO{}, err
	}
	return toNoteViewDTO(note), nil
}

func (service NoteApplicationService) ListNotes(ctx context.Context, query ListNotesQuery) ([]NoteViewDTO, error) {
	notes, err := service.noteRepository.List(ctx, query.Archived)
	if err != nil {
		return nil, err
	}
	return toNoteViewDTOs(domain.OrderForReading(notes)), nil
}

// load only parses an ID and finds the aggregate.
func (service NoteApplicationService) load(ctx context.Context, noteID string) (domain.Note, error) {
	id, err := domain.NewNoteID(noteID)
	if err != nil {
		return domain.Note{}, err
	}
	return service.noteRepository.FindByID(ctx, id)
}
