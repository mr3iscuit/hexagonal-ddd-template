package domain_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/domain"
)

var (
	createdAt = time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	changedAt = createdAt.Add(time.Hour)
)

func noteID(t *testing.T, value string) domain.NoteID {
	t.Helper()
	id, err := domain.NewNoteID(value)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func title(t *testing.T, value string) domain.Title {
	t.Helper()
	parsed, err := domain.NewTitle(value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func body(t *testing.T, value string) domain.Body {
	t.Helper()
	parsed, err := domain.NewBody(value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func newNote(t *testing.T) domain.Note {
	t.Helper()
	note, err := domain.NewNote(noteID(t, "groceries"), title(t, "Groceries"), body(t, "Milk"), createdAt)
	if err != nil {
		t.Fatal(err)
	}
	return note
}

func archived(t *testing.T) domain.Note {
	t.Helper()
	note, err := newNote(t).Archive(createdAt)
	if err != nil {
		t.Fatal(err)
	}
	return note
}

func eventNames(note domain.Note) []string {
	var names []string
	for _, event := range note.PendingEvents() {
		names = append(names, event.EventName())
	}
	return names
}

func TestANewNoteStartsActiveAndUnsaved(t *testing.T) {
	note := newNote(t)
	if note.ID().String() != "groceries" || note.Title().String() != "Groceries" || note.Body().String() != "Milk" {
		t.Errorf("note = %+v", note)
	}
	if note.IsArchived() || note.Version() != 0 {
		t.Errorf("note = %+v", note)
	}
	if !note.CreatedAt().Equal(createdAt) || !note.EditedAt().Equal(createdAt) {
		t.Errorf("createdAt = %v, editedAt = %v", note.CreatedAt(), note.EditedAt())
	}
	if !slices.Equal(eventNames(note), []string{"note.created"}) {
		t.Errorf("events = %v", eventNames(note))
	}
}

func TestANoteRechecksItsOwnInvariants(t *testing.T) {
	testCases := []struct {
		name string
		id   domain.NoteID
		text domain.Title
		time time.Time
		want error
	}{
		{"no ID", domain.NoteID{}, title(t, "Groceries"), createdAt, domain.ErrNoteIDRequired},
		{"no title", noteID(t, "groceries"), domain.Title{}, createdAt, domain.ErrTitleRequired},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := domain.NewNote(testCase.id, testCase.text, domain.Body{}, testCase.time); !errors.Is(err, testCase.want) {
				t.Errorf("err = %v", err)
			}
		})
	}
	if _, err := domain.NewNote(noteID(t, "groceries"), title(t, "Groceries"), domain.Body{}, time.Time{}); err == nil {
		t.Error("a note without a creation time was accepted")
	}
}

func TestEditingReplacesTheContentAndLeavesTheOriginalAlone(t *testing.T) {
	note := newNote(t)
	edited, err := note.Edit(title(t, "Shopping"), body(t, "Eggs"), changedAt)
	if err != nil {
		t.Fatal(err)
	}
	if edited.Title().String() != "Shopping" || edited.Body().String() != "Eggs" || !edited.EditedAt().Equal(changedAt) || !edited.CreatedAt().Equal(createdAt) {
		t.Errorf("edited = %+v", edited)
	}
	if note.Title().String() != "Groceries" || !note.EditedAt().Equal(createdAt) || len(note.PendingEvents()) != 1 {
		t.Errorf("the original changed: %+v", note)
	}
	if !slices.Equal(eventNames(edited), []string{"note.created", "note.edited"}) {
		t.Errorf("events = %v", eventNames(edited))
	}
}

func TestEditingRefusesAMissingTitle(t *testing.T) {
	if _, err := newNote(t).Edit(domain.Title{}, domain.Body{}, changedAt); !errors.Is(err, domain.ErrTitleRequired) {
		t.Errorf("err = %v", err)
	}
}

func TestRejectsEditingAnArchivedNote(t *testing.T) {
	if _, err := archived(t).Edit(title(t, "Shopping"), body(t, "Eggs"), changedAt); !errors.Is(err, domain.ErrNoteArchived) {
		t.Errorf("err = %v", err)
	}
}

func TestArchivingAndRestoringAreNotEdits(t *testing.T) {
	note, err := newNote(t).Archive(changedAt)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := note.Restore(changedAt.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if !note.EditedAt().Equal(createdAt) || !restored.EditedAt().Equal(createdAt) {
		t.Errorf("archived = %v, restored = %v", note.EditedAt(), restored.EditedAt())
	}
}

func TestArchivingAndRestoring(t *testing.T) {
	note := archived(t)
	if !note.IsArchived() {
		t.Error("the note is not archived")
	}
	restored, err := note.Restore(changedAt)
	if err != nil {
		t.Fatal(err)
	}
	if restored.IsArchived() || !note.IsArchived() {
		t.Errorf("restored = %v, original = %v", restored.IsArchived(), note.IsArchived())
	}
	if !slices.Equal(eventNames(restored), []string{"note.created", "note.archived", "note.restored"}) {
		t.Errorf("events = %v", eventNames(restored))
	}
}

func TestRejectsArchivingAnArchivedNote(t *testing.T) {
	if _, err := archived(t).Archive(changedAt); !errors.Is(err, domain.ErrNoteAlreadyArchived) {
		t.Errorf("err = %v", err)
	}
}

func TestRejectsRestoringANoteThatIsNotArchived(t *testing.T) {
	if _, err := newNote(t).Restore(changedAt); !errors.Is(err, domain.ErrNoteNotArchived) {
		t.Errorf("err = %v", err)
	}
}

func TestDeletingRecordsTheFact(t *testing.T) {
	deleted, err := archived(t).Delete(changedAt)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(eventNames(deleted), []string{"note.created", "note.archived", "note.deleted"}) {
		t.Errorf("events = %v", eventNames(deleted))
	}
}

func TestANoteThatWasNeverCreatedRefusesEveryBehaviour(t *testing.T) {
	var note domain.Note
	behaviours := map[string]func() (domain.Note, error){
		"edit":    func() (domain.Note, error) { return note.Edit(title(t, "Shopping"), domain.Body{}, changedAt) },
		"archive": func() (domain.Note, error) { return note.Archive(changedAt) },
		"restore": func() (domain.Note, error) { return note.Restore(changedAt) },
		"delete":  func() (domain.Note, error) { return note.Delete(changedAt) },
	}
	for name, behaviour := range behaviours {
		t.Run(name, func(t *testing.T) {
			_, err := behaviour()
			if err == nil {
				t.Fatal("the behaviour was allowed")
			}
			for _, published := range []error{domain.ErrNoteNotArchived, domain.ErrNoteAlreadyArchived, domain.ErrNoteArchived} {
				if errors.Is(err, published) {
					t.Errorf("err = %v must not look like a client's mistake", err)
				}
			}
		})
	}
}

// A stored note must have its times, so a change with no time would save a note
// that can never be reconstituted.
func TestEveryBehaviourRefusesAChangeWithNoTime(t *testing.T) {
	var noTime time.Time
	behaviours := map[string]func() (domain.Note, error){
		"edit":    func() (domain.Note, error) { return newNote(t).Edit(title(t, "Shopping"), domain.Body{}, noTime) },
		"archive": func() (domain.Note, error) { return newNote(t).Archive(noTime) },
		"restore": func() (domain.Note, error) { return archived(t).Restore(noTime) },
		"delete":  func() (domain.Note, error) { return newNote(t).Delete(noTime) },
	}
	for name, behaviour := range behaviours {
		t.Run(name, func(t *testing.T) {
			if _, err := behaviour(); err == nil {
				t.Fatal("the change was allowed")
			}
		})
	}
}

func TestPendingEventsCannotBeChangedFromOutside(t *testing.T) {
	note := newNote(t)
	note.PendingEvents()[0] = nil
	if note.PendingEvents()[0] == nil {
		t.Error("the caller changed the note's events")
	}
}

func TestBranchingChangesDoNotShareEvents(t *testing.T) {
	note := newNote(t)
	first, err := note.Edit(title(t, "Shopping"), domain.Body{}, changedAt)
	if err != nil {
		t.Fatal(err)
	}
	second, err := note.Archive(changedAt)
	if err != nil {
		t.Fatal(err)
	}
	if eventNames(first)[1] != "note.edited" || eventNames(second)[1] != "note.archived" {
		t.Errorf("first = %v, second = %v", eventNames(first), eventNames(second))
	}
}

func TestEventsCarryTheNoteAndTheTime(t *testing.T) {
	note, err := newNote(t).Archive(changedAt)
	if err != nil {
		t.Fatal(err)
	}
	event := note.PendingEvents()[1]
	if event.NoteID() != note.ID() || !event.OccurredAt().Equal(changedAt) {
		t.Errorf("event = %+v", event)
	}
}

func TestPublishedEventNamesNeverChange(t *testing.T) {
	want := map[string]domain.Event{
		"note.created":       domain.NoteCreated{},
		"note.edited":        domain.NoteEdited{},
		"note.archived":      domain.NoteArchived{},
		"note.restored":      domain.NoteRestored{},
		"note.deleted":       domain.NoteDeleted{},
		"reminder.scheduled": domain.ReminderScheduled{},
	}
	for name, event := range want {
		if event.EventName() != name {
			t.Errorf("%T has the name %q", event, event.EventName())
		}
	}
}
