package domain

import (
	"slices"
	"time"
)

// Note is the aggregate root. Its behaviour returns a new Note, so a rejected
// change can't leave a half-changed one behind.
type Note struct {
	id            NoteID
	title         Title
	body          Body
	archived      bool
	remindAt      time.Time
	createdAt     time.Time
	editedAt      time.Time
	version       int
	pendingEvents []Event
}

// NewNote creates a Note that has not been stored yet, so it is at version 0.
func NewNote(id NoteID, title Title, body Body, createdAt time.Time) (Note, error) {
	if id.IsZero() {
		return Note{}, ErrNoteIDRequired
	}
	if title.IsZero() {
		return Note{}, ErrTitleRequired
	}
	if createdAt.IsZero() {
		return Note{}, errNoteTimeMissing
	}
	return Note{
		id: id, title: title, body: body, createdAt: createdAt, editedAt: createdAt,
		pendingEvents: []Event{NoteCreated{noteEvent{id, createdAt}}},
	}, nil
}

// ReconstituteNote rebuilds a stored Note. It checks structure only, so a later
// change to a business rule can't make saved data look damaged.
func ReconstituteNote(id NoteID, title Title, body Body, archived bool, createdAt, editedAt time.Time, version int) (Note, error) {
	if id.IsZero() {
		return Note{}, ErrNoteIDRequired
	}
	if title.IsZero() {
		return Note{}, ErrTitleRequired
	}
	if createdAt.IsZero() || editedAt.IsZero() {
		return Note{}, errNoteTimeMissing
	}
	if version < 1 {
		return Note{}, errNoteVersionInvalid
	}
	return Note{id: id, title: title, body: body, archived: archived, createdAt: createdAt, editedAt: editedAt, version: version}, nil
}

// Edit replaces the title and body.
func (note Note) Edit(title Title, body Body, editedAt time.Time) (Note, error) {
	if err := note.requireChange(editedAt); err != nil {
		return Note{}, err
	}
	if note.archived {
		return Note{}, ErrNoteArchived
	}
	if title.IsZero() {
		return Note{}, ErrTitleRequired
	}
	note.title = title
	note.body = body
	note.editedAt = editedAt
	return note.recording(NoteEdited{noteEvent{note.id, editedAt}}), nil
}

// ScheduleReminder asks to be reminded about the note at remindAt.
func (note Note) ScheduleReminder(remindAt, scheduledAt time.Time) (Note, error) {
	if note.archived {
		return Note{}, ErrNoteArchived
	}
	if !remindAt.After(scheduledAt) {
		return Note{}, ErrReminderNotInFuture
	}
	note.remindAt = remindAt
	return note.recording(ReminderScheduled{noteEvent{note.id, scheduledAt}}), nil
}

// CancelReminder removes the reminder.
func (note Note) CancelReminder(cancelledAt time.Time) (Note, error) {
	if note.remindAt.IsZero() {
		return Note{}, ErrNoReminderScheduled
	}
	note.remindAt = time.Time{}
	return note.recording(ReminderCancelled{noteEvent{note.id, cancelledAt}}), nil
}

// Archive puts the note away.
func (note Note) Archive(archivedAt time.Time) (Note, error) {
	if err := note.requireChange(archivedAt); err != nil {
		return Note{}, err
	}
	if note.archived {
		return Note{}, ErrNoteAlreadyArchived
	}
	note.archived = true
	return note.recording(NoteArchived{noteEvent{note.id, archivedAt}}), nil
}

// Restore brings an archived note back.
func (note Note) Restore(restoredAt time.Time) (Note, error) {
	if err := note.requireChange(restoredAt); err != nil {
		return Note{}, err
	}
	if !note.archived {
		return Note{}, ErrNoteNotArchived
	}
	note.archived = false
	return note.recording(NoteRestored{noteEvent{note.id, restoredAt}}), nil
}

// Delete marks the end of the note's life.
func (note Note) Delete(deletedAt time.Time) (Note, error) {
	if err := note.requireChange(deletedAt); err != nil {
		return Note{}, err
	}
	return note.recording(NoteDeleted{noteEvent{note.id, deletedAt}}), nil
}

// requireChange refuses a zero Note, which Go always allows, and a change with
// no time, which would save a Note that can never be reconstituted.
func (note Note) requireChange(changedAt time.Time) error {
	if note.id.IsZero() {
		return errNoteNotCreated
	}
	if changedAt.IsZero() {
		return errNoteTimeMissing
	}
	return nil
}

func (note Note) recording(event Event) Note {
	note.pendingEvents = append(slices.Clone(note.pendingEvents), event)
	return note
}

func (note Note) ID() NoteID { return note.id }

func (note Note) Title() Title { return note.title }

func (note Note) Body() Body { return note.body }

func (note Note) IsArchived() bool { return note.archived }

func (note Note) RemindAt() time.Time { return note.remindAt }

func (note Note) CreatedAt() time.Time { return note.createdAt }

func (note Note) EditedAt() time.Time { return note.editedAt }

func (note Note) Version() int { return note.version }

func (note Note) PendingEvents() []Event { return slices.Clone(note.pendingEvents) }
