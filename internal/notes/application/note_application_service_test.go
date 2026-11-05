package application_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/application"
	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/domain"
)

var startedAt = time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)

type fakeClock struct{ now time.Time }

func (clock *fakeClock) Now() time.Time { return clock.now }

type fakeIDGenerator struct {
	id  string
	err error
}

func (generator fakeIDGenerator) NextNoteID() (domain.NoteID, error) {
	if generator.err != nil {
		return domain.NoteID{}, generator.err
	}
	return domain.NewNoteID(generator.id)
}

// fakeRepository records calls, so tests can prove what was not stored.
type fakeRepository struct {
	notes       map[string]domain.Note
	calls       []string
	archived    bool
	failure     error
	listFailure error
}

func (repository *fakeRepository) Save(_ context.Context, note domain.Note) error {
	repository.calls = append(repository.calls, "save")
	if repository.failure != nil {
		return repository.failure
	}
	repository.notes[note.ID().String()] = note
	return nil
}

func (repository *fakeRepository) Remove(_ context.Context, note domain.Note) error {
	repository.calls = append(repository.calls, "remove")
	if repository.failure != nil {
		return repository.failure
	}
	delete(repository.notes, note.ID().String())
	return nil
}

func (repository *fakeRepository) FindByID(_ context.Context, id domain.NoteID) (domain.Note, error) {
	repository.calls = append(repository.calls, "find")
	note, found := repository.notes[id.String()]
	if !found {
		return domain.Note{}, application.ErrNoteNotFound
	}
	return note, nil
}

func (repository *fakeRepository) List(_ context.Context, archived bool) ([]domain.Note, error) {
	repository.calls = append(repository.calls, "list")
	repository.archived = archived
	if repository.listFailure != nil {
		return nil, repository.listFailure
	}
	var notes []domain.Note
	for _, note := range repository.notes {
		notes = append(notes, note)
	}
	// A fixed order, so that a test can't pass because the map happened to help.
	slices.SortFunc(notes, func(first, second domain.Note) int {
		return strings.Compare(first.ID().String(), second.ID().String())
	})
	return notes, nil
}

func (repository *fakeRepository) ListDueReminders(context.Context, time.Time) ([]domain.Note, error) {
	return nil, nil
}

type fixture struct {
	service    application.NoteApplicationService
	repository *fakeRepository
	clock      *fakeClock
}

func newFixture() fixture {
	repository := &fakeRepository{notes: map[string]domain.Note{}}
	clock := &fakeClock{now: startedAt}
	return fixture{
		service:    application.NewNoteApplicationService(repository, clock, fakeIDGenerator{id: "groceries"}),
		repository: repository,
		clock:      clock,
	}
}

func (test fixture) create(t *testing.T) application.NoteViewDTO {
	t.Helper()
	view, err := test.service.CreateNote(context.Background(), application.CreateNoteCommand{Title: "Groceries", Body: "Milk"})
	if err != nil {
		t.Fatal(err)
	}
	test.repository.calls = nil
	return view
}

func TestCreatingANoteStoresItAndReturnsItsView(t *testing.T) {
	test := newFixture()
	view := test.create(t)
	want := application.NoteViewDTO{ID: "groceries", Title: "Groceries", Body: "Milk", CreatedAt: startedAt, EditedAt: startedAt}
	if view != want {
		t.Errorf("view = %+v, want %+v", view, want)
	}
	stored := test.repository.notes["groceries"]
	if stored.Title().String() != "Groceries" || stored.PendingEvents()[0].EventName() != "note.created" {
		t.Errorf("stored = %+v", stored)
	}
}

func TestCreatingANoteWithInvalidContentStoresNothing(t *testing.T) {
	testCases := []struct {
		name    string
		command application.CreateNoteCommand
		want    error
	}{
		{"no title", application.CreateNoteCommand{Title: " "}, domain.ErrTitleRequired},
		{"long title", application.CreateNoteCommand{Title: strings.Repeat("a", 201)}, domain.ErrTitleTooLong},
		{"long body", application.CreateNoteCommand{Title: "Groceries", Body: strings.Repeat("a", 10001)}, domain.ErrBodyTooLong},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			test := newFixture()
			if _, err := test.service.CreateNote(context.Background(), testCase.command); !errors.Is(err, testCase.want) {
				t.Errorf("err = %v", err)
			}
			if len(test.repository.calls) != 0 {
				t.Errorf("calls = %v", test.repository.calls)
			}
		})
	}
}

func TestCreatingANoteStopsWhenNoIDCanBeGenerated(t *testing.T) {
	failure := errors.New("no randomness")
	repository := &fakeRepository{notes: map[string]domain.Note{}}
	service := application.NewNoteApplicationService(repository, &fakeClock{now: startedAt}, fakeIDGenerator{err: failure})
	if _, err := service.CreateNote(context.Background(), application.CreateNoteCommand{Title: "Groceries"}); !errors.Is(err, failure) {
		t.Errorf("err = %v", err)
	}
	if len(repository.calls) != 0 {
		t.Errorf("calls = %v", repository.calls)
	}
}

func TestEditingUsesTheClockAndStoresTheChange(t *testing.T) {
	test := newFixture()
	test.create(t)
	test.clock.now = startedAt.Add(time.Hour)
	view, err := test.service.EditNote(context.Background(), application.EditNoteCommand{NoteID: "groceries", Title: "Shopping", Body: "Eggs"})
	if err != nil {
		t.Fatal(err)
	}
	if view.Title != "Shopping" || view.Body != "Eggs" || !view.EditedAt.Equal(test.clock.now) || !view.CreatedAt.Equal(startedAt) {
		t.Errorf("view = %+v", view)
	}
	if !slices.Equal(test.repository.calls, []string{"find", "save"}) {
		t.Errorf("calls = %v", test.repository.calls)
	}
}

func TestAnInvalidEditCostsNoStorageCall(t *testing.T) {
	for name, command := range map[string]application.EditNoteCommand{
		"no title":  {NoteID: "groceries", Title: " "},
		"long body": {NoteID: "groceries", Title: "Shopping", Body: strings.Repeat("a", 10001)},
	} {
		t.Run(name, func(t *testing.T) {
			test := newFixture()
			test.create(t)
			if _, err := test.service.EditNote(context.Background(), command); err == nil {
				t.Error("the edit was accepted")
			}
			if len(test.repository.calls) != 0 {
				t.Errorf("calls = %v", test.repository.calls)
			}
		})
	}
}

func TestArchivingAndRestoring(t *testing.T) {
	test := newFixture()
	test.create(t)
	ctx := context.Background()
	archived, err := test.service.ArchiveNote(ctx, application.ArchiveNoteCommand{NoteID: "groceries"})
	if err != nil || !archived.Archived {
		t.Fatalf("archived = %+v, err = %v", archived, err)
	}
	restored, err := test.service.RestoreNote(ctx, application.RestoreNoteCommand{NoteID: "groceries"})
	if err != nil || restored.Archived {
		t.Fatalf("restored = %+v, err = %v", restored, err)
	}
	if saves := len(slices.DeleteFunc(test.repository.calls, func(call string) bool { return call != "save" })); saves != 2 {
		t.Errorf("saves = %d", saves)
	}
}

func TestEveryChangeIsStoredWithItsTime(t *testing.T) {
	test := newFixture()
	test.create(t)
	test.clock.now = startedAt.Add(time.Hour)
	if _, err := test.service.ArchiveNote(context.Background(), application.ArchiveNoteCommand{NoteID: "groceries"}); err != nil {
		t.Fatal(err)
	}
	events := test.repository.notes["groceries"].PendingEvents()
	if last := events[len(events)-1]; !last.OccurredAt().Equal(test.clock.now) {
		t.Errorf("event at %v", last.OccurredAt())
	}
}

func TestDeletingRemovesTheNoteWithItsDeletedEvent(t *testing.T) {
	test := newFixture()
	test.create(t)
	if err := test.service.DeleteNote(context.Background(), application.DeleteNoteCommand{NoteID: "groceries"}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(test.repository.calls, []string{"find", "remove"}) {
		t.Errorf("calls = %v", test.repository.calls)
	}
	if _, found := test.repository.notes["groceries"]; found {
		t.Error("the note is still stored")
	}
}

func TestReadingANote(t *testing.T) {
	test := newFixture()
	created := test.create(t)
	view, err := test.service.GetNote(context.Background(), application.GetNoteQuery{NoteID: "groceries"})
	if err != nil || view != created {
		t.Errorf("view = %+v, err = %v", view, err)
	}
}

func TestListingOrdersNotesForReadingAndPassesTheFilter(t *testing.T) {
	repository := &fakeRepository{notes: map[string]domain.Note{}}
	clock := &fakeClock{now: startedAt}
	// Storage returns IDs in order, which here is the opposite of reading order.
	for _, id := range []string{"a-older", "b-newer"} {
		service := application.NewNoteApplicationService(repository, clock, fakeIDGenerator{id: id})
		if _, err := service.CreateNote(context.Background(), application.CreateNoteCommand{Title: id}); err != nil {
			t.Fatal(err)
		}
		clock.now = clock.now.Add(time.Hour)
	}
	service := application.NewNoteApplicationService(repository, clock, fakeIDGenerator{})
	views, err := service.ListNotes(context.Background(), application.ListNotesQuery{Archived: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 2 || views[0].ID != "b-newer" || views[1].ID != "a-older" {
		t.Errorf("views = %+v", views)
	}
	if !repository.archived {
		t.Error("the archived filter was not passed on")
	}
}

// useCaseCalls reaches every use case that takes a note ID.
func useCaseCalls() map[string]func(application.NoteApplicationService, string) error {
	ctx := context.Background()
	return map[string]func(application.NoteApplicationService, string) error{
		"edit": func(service application.NoteApplicationService, id string) error {
			_, err := service.EditNote(ctx, application.EditNoteCommand{NoteID: id, Title: "Shopping"})
			return err
		},
		"archive": func(service application.NoteApplicationService, id string) error {
			_, err := service.ArchiveNote(ctx, application.ArchiveNoteCommand{NoteID: id})
			return err
		},
		"restore": func(service application.NoteApplicationService, id string) error {
			_, err := service.RestoreNote(ctx, application.RestoreNoteCommand{NoteID: id})
			return err
		},
		"delete": func(service application.NoteApplicationService, id string) error {
			return service.DeleteNote(ctx, application.DeleteNoteCommand{NoteID: id})
		},
		"get": func(service application.NoteApplicationService, id string) error {
			_, err := service.GetNote(ctx, application.GetNoteQuery{NoteID: id})
			return err
		},
	}
}

// changeable creates a note in the one state each use case accepts.
func (test fixture) changeable(t *testing.T, useCase string) {
	t.Helper()
	test.create(t)
	if useCase == "restore" {
		if _, err := test.service.ArchiveNote(context.Background(), application.ArchiveNoteCommand{NoteID: "groceries"}); err != nil {
			t.Fatal(err)
		}
	}
	test.repository.calls = nil
}

func assertNothingStored(t *testing.T, repository *fakeRepository) {
	t.Helper()
	if slices.Contains(repository.calls, "save") || slices.Contains(repository.calls, "remove") {
		t.Errorf("calls = %v", repository.calls)
	}
}

func TestUseCasesReportAnUnknownOrBlankNote(t *testing.T) {
	for name, useCase := range useCaseCalls() {
		t.Run(name, func(t *testing.T) {
			test := newFixture()
			if err := useCase(test.service, "missing"); !errors.Is(err, application.ErrNoteNotFound) {
				t.Errorf("unknown note: err = %v", err)
			}
			if err := useCase(test.service, " "); !errors.Is(err, domain.ErrNoteIDRequired) {
				t.Errorf("blank ID: err = %v", err)
			}
		})
	}
}

func TestRefusedChangesReachTheCallerAndAreNeverSaved(t *testing.T) {
	testCases := []struct {
		useCase  string
		archived bool
		want     error
	}{
		{useCase: "edit", archived: true, want: domain.ErrNoteArchived},
		{useCase: "archive", archived: true, want: domain.ErrNoteAlreadyArchived},
		{useCase: "restore", want: domain.ErrNoteNotArchived},
	}
	calls := useCaseCalls()
	for _, testCase := range testCases {
		t.Run(testCase.want.Error(), func(t *testing.T) {
			test := newFixture()
			test.create(t)
			if testCase.archived {
				if err := calls["archive"](test.service, "groceries"); err != nil {
					t.Fatal(err)
				}
			}
			test.repository.calls = nil
			if err := calls[testCase.useCase](test.service, "groceries"); !errors.Is(err, testCase.want) {
				t.Errorf("err = %v", err)
			}
			assertNothingStored(t, test.repository)
		})
	}
}

func TestAChangeWithNoTimeIsNeverStored(t *testing.T) {
	for name, useCase := range useCaseCalls() {
		if name == "get" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			test := newFixture()
			test.changeable(t, name)
			test.clock.now = time.Time{}
			if err := useCase(test.service, "groceries"); err == nil {
				t.Error("the change was allowed")
			}
			assertNothingStored(t, test.repository)
		})
	}
}

func TestCreatingANoteWithNoTimeStoresNothing(t *testing.T) {
	test := newFixture()
	test.clock.now = time.Time{}
	if _, err := test.service.CreateNote(context.Background(), application.CreateNoteCommand{Title: "Groceries"}); err == nil {
		t.Error("the note was created")
	}
	assertNothingStored(t, test.repository)
}

func TestStorageFailuresReachTheCaller(t *testing.T) {
	failure := errors.New("storage is down")
	for name, useCase := range useCaseCalls() {
		if name == "get" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			test := newFixture()
			test.changeable(t, name)
			test.repository.failure = failure
			if err := useCase(test.service, "groceries"); !errors.Is(err, failure) {
				t.Errorf("err = %v", err)
			}
		})
	}
	t.Run("create", func(t *testing.T) {
		test := newFixture()
		test.repository.failure = failure
		if _, err := test.service.CreateNote(context.Background(), application.CreateNoteCommand{Title: "Groceries"}); !errors.Is(err, failure) {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("list", func(t *testing.T) {
		test := newFixture()
		test.repository.listFailure = failure
		if _, err := test.service.ListNotes(context.Background(), application.ListNotesQuery{}); !errors.Is(err, failure) {
			t.Errorf("err = %v", err)
		}
	})
}
