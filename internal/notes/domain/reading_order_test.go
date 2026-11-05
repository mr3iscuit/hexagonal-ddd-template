package domain_test

import (
	"slices"
	"testing"
	"time"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/domain"
)

func storedNote(t *testing.T, id string, editedAt time.Time) domain.Note {
	t.Helper()
	note, err := domain.ReconstituteNote(noteID(t, id), title(t, id), domain.Body{}, false, createdAt, editedAt, 1)
	if err != nil {
		t.Fatal(err)
	}
	return note
}

func TestTheMostRecentlyEditedNotesComeFirst(t *testing.T) {
	notes := []domain.Note{
		storedNote(t, "old", createdAt),
		storedNote(t, "new", changedAt),
		storedNote(t, "middle", createdAt.Add(time.Minute)),
	}
	var ids []string
	for _, note := range domain.OrderForReading(notes) {
		ids = append(ids, note.ID().String())
	}
	if want := []string{"new", "middle", "old"}; !slices.Equal(ids, want) {
		t.Errorf("order = %v, want %v", ids, want)
	}
	if notes[0].ID().String() != "old" {
		t.Error("the caller's slice was reordered")
	}
}

func TestNotesEditedAtTheSameTimeHaveAStableOrder(t *testing.T) {
	notes := []domain.Note{storedNote(t, "b", createdAt), storedNote(t, "a", createdAt)}
	ordered := domain.OrderForReading(notes)
	if ordered[0].ID().String() != "a" {
		t.Errorf("first = %v", ordered[0].ID())
	}
}
