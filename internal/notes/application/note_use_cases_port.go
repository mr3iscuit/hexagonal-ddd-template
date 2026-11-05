package application

import "context"

// NoteUseCasesPort is the inbound port.
type NoteUseCasesPort interface {
	CreateNote(ctx context.Context, command CreateNoteCommand) (NoteViewDTO, error)
	EditNote(ctx context.Context, command EditNoteCommand) (NoteViewDTO, error)
	ArchiveNote(ctx context.Context, command ArchiveNoteCommand) (NoteViewDTO, error)
	RestoreNote(ctx context.Context, command RestoreNoteCommand) (NoteViewDTO, error)
	DeleteNote(ctx context.Context, command DeleteNoteCommand) error
	ScheduleReminder(ctx context.Context, command ScheduleReminderCommand) (NoteViewDTO, error)
	GetNote(ctx context.Context, query GetNoteQuery) (NoteViewDTO, error)
	ListNotes(ctx context.Context, query ListNotesQuery) ([]NoteViewDTO, error)
}
