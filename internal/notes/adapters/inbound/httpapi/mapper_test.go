package httpapi

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/application"
)

func TestRequiredFieldListsMatchTheJSONTags(t *testing.T) {
	testCases := []struct {
		dto    any
		fields []string
	}{
		{CreateNoteRequestDTO{}, createNoteRequestFields},
		{EditNoteRequestDTO{}, editNoteRequestFields},
		{ScheduleReminderRequestDTO{}, scheduleReminderRequestFields},
	}
	for _, testCase := range testCases {
		var tags []string
		for field := range reflect.TypeOf(testCase.dto).Fields() {
			tags = append(tags, field.Tag.Get("json"))
		}
		if !slices.Equal(tags, testCase.fields) {
			t.Errorf("%T: tags = %v, fields = %v", testCase.dto, tags, testCase.fields)
		}
	}
}

func TestRequestMappersKeepEveryField(t *testing.T) {
	if got, want := toCreateNoteCommand(CreateNoteRequestDTO{Title: "a", Body: "b"}), (application.CreateNoteCommand{Title: "a", Body: "b"}); got != want {
		t.Errorf("got %+v", got)
	}
	if got, want := toEditNoteCommand("id", EditNoteRequestDTO{Title: "a", Body: "b"}), (application.EditNoteCommand{NoteID: "id", Title: "a", Body: "b"}); got != want {
		t.Errorf("got %+v", got)
	}
	remindAt := time.Date(2026, time.March, 4, 9, 30, 0, 0, time.UTC)
	if got, want := toScheduleReminderCommand("id", remindAt), (application.ScheduleReminderCommand{NoteID: "id", RemindAt: remindAt}); got != want {
		t.Errorf("got %+v", got)
	}
	if got, want := toListNotesQuery(true), (application.ListNotesQuery{Archived: true}); got != want {
		t.Errorf("got %+v", got)
	}
	if toArchiveNoteCommand("id").NoteID != "id" ||
		toRestoreNoteCommand("id").NoteID != "id" || toDeleteNoteCommand("id").NoteID != "id" || toGetNoteQuery("id").NoteID != "id" {
		t.Error("a mapper lost the note ID")
	}
}

func TestResponseMappingKeepsEveryFieldAndConvertsToUTC(t *testing.T) {
	zone := time.FixedZone("Baku", 4*60*60)
	createdAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, zone)
	editedAt := createdAt.Add(time.Hour)
	view := application.NoteViewDTO{ID: "id", Title: "a", Body: "b", Archived: true, RemindAt: editedAt, CreatedAt: createdAt, EditedAt: editedAt}
	want := NoteResponseDTO{ID: "id", Title: "a", Body: "b", Archived: true, RemindAt: editedAt.UTC(), CreatedAt: createdAt.UTC(), EditedAt: editedAt.UTC()}
	if got := toNoteResponseDTO(view); got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
	list := toNoteListResponseDTO([]application.NoteViewDTO{view})
	if len(list.Notes) != 1 || list.Notes[0] != want {
		t.Errorf("list = %+v", list)
	}
}

func TestAnEmptyListIsAnEmptyJSONArray(t *testing.T) {
	content, err := json.Marshal(toNoteListResponseDTO(nil))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != `{"notes":[]}` {
		t.Errorf("content = %s", content)
	}
}
