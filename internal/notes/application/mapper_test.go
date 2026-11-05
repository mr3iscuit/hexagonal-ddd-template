package application

import (
	"testing"
	"time"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/domain"
)

func TestViewMappingKeepsEveryField(t *testing.T) {
	createdAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	editedAt := createdAt.Add(time.Hour)
	id, err := domain.NewNoteID("groceries")
	if err != nil {
		t.Fatal(err)
	}
	title, err := domain.NewTitle("Groceries")
	if err != nil {
		t.Fatal(err)
	}
	body, err := domain.NewBody("Milk")
	if err != nil {
		t.Fatal(err)
	}
	note, err := domain.ReconstituteNote(id, title, body, true, createdAt, editedAt, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := NoteViewDTO{ID: "groceries", Title: "Groceries", Body: "Milk", Archived: true, CreatedAt: createdAt, EditedAt: editedAt}
	if got := toNoteViewDTO(note); got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if views := toNoteViewDTOs([]domain.Note{note}); len(views) != 1 || views[0] != want {
		t.Errorf("views = %+v", views)
	}
}

func TestAnEmptyListIsNeverNil(t *testing.T) {
	if views := toNoteViewDTOs(nil); views == nil {
		t.Error("views is nil")
	}
}
