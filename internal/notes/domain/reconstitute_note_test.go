package domain_test

import (
	"testing"
	"time"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/domain"
)

func TestReconstitutingKeepsEveryValueAndRaisesNoEvents(t *testing.T) {
	note, err := domain.ReconstituteNote(noteID(t, "groceries"), title(t, "Groceries"), body(t, "Milk"), true, createdAt, changedAt, 7)
	if err != nil {
		t.Fatal(err)
	}
	if note.ID().String() != "groceries" || note.Title().String() != "Groceries" || note.Body().String() != "Milk" {
		t.Errorf("note = %+v", note)
	}
	if !note.IsArchived() || !note.CreatedAt().Equal(createdAt) || !note.EditedAt().Equal(changedAt) || note.Version() != 7 {
		t.Errorf("note = %+v", note)
	}
	if len(note.PendingEvents()) != 0 {
		t.Errorf("events = %v", note.PendingEvents())
	}
}

func TestReconstitutingRefusesAnIncompleteNote(t *testing.T) {
	id, text := noteID(t, "groceries"), title(t, "Groceries")
	testCases := []struct {
		name      string
		id        domain.NoteID
		title     domain.Title
		createdAt time.Time
		editedAt  time.Time
		version   int
	}{
		{"no ID", domain.NoteID{}, text, createdAt, changedAt, 1},
		{"no title", id, domain.Title{}, createdAt, changedAt, 1},
		{"no creation time", id, text, time.Time{}, changedAt, 1},
		{"no edit time", id, text, createdAt, time.Time{}, 1},
		{"never stored", id, text, createdAt, changedAt, 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := domain.ReconstituteNote(testCase.id, testCase.title, domain.Body{}, false, testCase.createdAt, testCase.editedAt, testCase.version); err == nil {
				t.Error("the incomplete note was accepted")
			}
		})
	}
}

// A clock can step backwards, so an edit time before the creation time is valid
// saved data. A check on it would be a business rule, and those stay out of
// reconstitution, or changing one would make saved data look damaged.
func TestReconstitutingAppliesNoBusinessRules(t *testing.T) {
	if _, err := domain.ReconstituteNote(noteID(t, "groceries"), title(t, "Groceries"), domain.Body{}, true, changedAt, createdAt, 1); err != nil {
		t.Errorf("err = %v", err)
	}
}

func TestReconstitutedNoteCanStillChange(t *testing.T) {
	note, err := domain.ReconstituteNote(noteID(t, "groceries"), title(t, "Groceries"), domain.Body{}, false, createdAt, createdAt, 3)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := note.Archive(changedAt)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Version() != 3 || len(changed.PendingEvents()) != 1 {
		t.Errorf("version = %d, events = %v", changed.Version(), changed.PendingEvents())
	}
}
