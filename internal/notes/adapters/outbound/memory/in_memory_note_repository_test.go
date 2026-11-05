package memory

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/application"
	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/domain"
)

func newNote(t *testing.T, id string) domain.Note {
	t.Helper()
	noteID, err := domain.NewNoteID(id)
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
	note, err := domain.NewNote(noteID, title, body, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	return note
}

func saved(t *testing.T, repository *InMemoryNoteRepository, id string) domain.Note {
	t.Helper()
	note := newNote(t, id)
	if err := repository.Save(context.Background(), note); err != nil {
		t.Fatal(err)
	}
	found, err := repository.FindByID(context.Background(), note.ID())
	if err != nil {
		t.Fatal(err)
	}
	return found
}

func edit(t *testing.T, note domain.Note) domain.Note {
	t.Helper()
	edited, err := note.Edit(note.Title(), note.Body(), editedAt)
	if err != nil {
		t.Fatal(err)
	}
	return edited
}

func archive(t *testing.T, note domain.Note) domain.Note {
	t.Helper()
	archived, err := note.Archive(editedAt)
	if err != nil {
		t.Fatal(err)
	}
	return archived
}

func TestASavedNoteCanBeFoundAgain(t *testing.T) {
	note := saved(t, NewInMemoryNoteRepository(), "groceries")
	if note.ID().String() != "groceries" || note.Title().String() != "Groceries" || note.Body().String() != "Milk" || note.Version() != 1 {
		t.Errorf("note = %+v", note)
	}
	if len(note.PendingEvents()) != 0 {
		t.Errorf("events = %v", note.PendingEvents())
	}
}

func TestFindingAnUnknownNoteFails(t *testing.T) {
	id, err := domain.NewNoteID("missing")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewInMemoryNoteRepository().FindByID(context.Background(), id); !errors.Is(err, application.ErrNoteNotFound) {
		t.Errorf("err = %v", err)
	}
}

func TestEverySaveMovesTheVersionForward(t *testing.T) {
	repository := NewInMemoryNoteRepository()
	note := saved(t, repository, "groceries")
	if err := repository.Save(context.Background(), archive(t, note)); err != nil {
		t.Fatal(err)
	}
	found, err := repository.FindByID(context.Background(), note.ID())
	if err != nil {
		t.Fatal(err)
	}
	if found.Version() != 2 || !found.IsArchived() {
		t.Errorf("version = %d, archived = %v", found.Version(), found.IsArchived())
	}
}

func TestSavingAStaleNoteIsRejectedAndStoresNothing(t *testing.T) {
	repository := NewInMemoryNoteRepository()
	note := saved(t, repository, "groceries")
	if err := repository.Save(context.Background(), archive(t, note)); err != nil {
		t.Fatal(err)
	}
	stored := len(repository.outbox)
	if err := repository.Save(context.Background(), edit(t, note)); !errors.Is(err, application.ErrNoteModifiedConcurrently) {
		t.Errorf("err = %v", err)
	}
	found, err := repository.FindByID(context.Background(), note.ID())
	if err != nil {
		t.Fatal(err)
	}
	if !found.EditedAt().Equal(createdAt) || len(repository.outbox) != stored {
		t.Errorf("the rejected save left editedAt = %v, outbox = %d", found.EditedAt(), len(repository.outbox))
	}
}

func TestANewNoteCannotOverwriteAStoredOne(t *testing.T) {
	repository := NewInMemoryNoteRepository()
	saved(t, repository, "groceries")
	if err := repository.Save(context.Background(), newNote(t, "groceries")); !errors.Is(err, application.ErrNoteModifiedConcurrently) {
		t.Errorf("err = %v", err)
	}
}

func TestAnOverlappingEditCannotBringADeletedNoteBack(t *testing.T) {
	repository := NewInMemoryNoteRepository()
	note := saved(t, repository, "groceries")
	if err := repository.Remove(context.Background(), note); err != nil {
		t.Fatal(err)
	}
	if err := repository.Save(context.Background(), edit(t, note)); !errors.Is(err, application.ErrNoteModifiedConcurrently) {
		t.Errorf("err = %v", err)
	}
	if _, err := repository.FindByID(context.Background(), note.ID()); !errors.Is(err, application.ErrNoteNotFound) {
		t.Errorf("err = %v", err)
	}
}

func TestRemovingANoteStoresItsDeletedEvent(t *testing.T) {
	repository := NewInMemoryNoteRepository()
	note := saved(t, repository, "groceries")
	deleted, err := note.Delete(editedAt)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Remove(context.Background(), deleted); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.FindByID(context.Background(), note.ID()); !errors.Is(err, application.ErrNoteNotFound) {
		t.Errorf("err = %v", err)
	}
	if last := repository.outbox[len(repository.outbox)-1]; last.eventName != "note.deleted" {
		t.Errorf("last event = %+v", last)
	}
}

func TestRemovingAStaleNoteIsRejectedAndKeepsIt(t *testing.T) {
	repository := NewInMemoryNoteRepository()
	note := saved(t, repository, "groceries")
	if err := repository.Save(context.Background(), archive(t, note)); err != nil {
		t.Fatal(err)
	}
	if err := repository.Remove(context.Background(), note); !errors.Is(err, application.ErrNoteModifiedConcurrently) {
		t.Errorf("err = %v", err)
	}
	if _, err := repository.FindByID(context.Background(), note.ID()); err != nil {
		t.Errorf("err = %v", err)
	}
}

func TestRemovingANoteThatIsAlreadyGoneIsAConflict(t *testing.T) {
	repository := NewInMemoryNoteRepository()
	note := saved(t, repository, "groceries")
	if err := repository.Remove(context.Background(), note); err != nil {
		t.Fatal(err)
	}
	if err := repository.Remove(context.Background(), note); !errors.Is(err, application.ErrNoteModifiedConcurrently) {
		t.Errorf("err = %v", err)
	}
}

func TestRemovingANoteThatWasNeverStoredIsRejected(t *testing.T) {
	repository := NewInMemoryNoteRepository()
	if err := repository.Remove(context.Background(), newNote(t, "groceries")); !errors.Is(err, application.ErrNoteModifiedConcurrently) {
		t.Errorf("err = %v", err)
	}
	if len(repository.outbox) != 0 {
		t.Errorf("outbox = %+v", repository.outbox)
	}
}

func TestSavingStoresThePendingEventsWithTheNote(t *testing.T) {
	repository := NewInMemoryNoteRepository()
	note := saved(t, repository, "groceries")
	if err := repository.Save(context.Background(), archive(t, note)); err != nil {
		t.Fatal(err)
	}
	if len(repository.outbox) != 2 || repository.outbox[0].eventName != "note.created" || repository.outbox[1].eventName != "note.archived" {
		t.Errorf("outbox = %+v", repository.outbox)
	}
}

func fullOutbox() *InMemoryNoteRepository {
	repository := NewInMemoryNoteRepository()
	repository.outbox = make([]outboxRecord, maximumOutboxRecords)
	for index := range repository.outbox {
		repository.outbox[index].noteID = strconv.Itoa(index)
	}
	return repository
}

// oldestFirst reads the ring in the order a relay would publish it.
func oldestFirst(repository *InMemoryNoteRepository) []outboxRecord {
	return slices.Concat(repository.outbox[repository.oldestOutboxRecord:], repository.outbox[:repository.oldestOutboxRecord])
}

func assertOutboxEnds(t *testing.T, repository *InMemoryNoteRepository, oldest string, newest ...string) {
	t.Helper()
	records := oldestFirst(repository)
	if len(records) != maximumOutboxRecords {
		t.Fatalf("length = %d", len(records))
	}
	if records[0].noteID != oldest {
		t.Errorf("oldest = %+v, want %s", records[0], oldest)
	}
	for position, id := range newest {
		if got := records[maximumOutboxRecords-len(newest)+position].noteID; got != id {
			t.Errorf("newest[%d] = %s, want %s", position, got, id)
		}
	}
}

func TestTheOutboxDropsItsOldestRecordsAtTheCap(t *testing.T) {
	repository := fullOutbox()
	saved(t, repository, "first")
	saved(t, repository, "second")
	assertOutboxEnds(t, repository, "2", "first", "second")
}

func TestTheOutboxWrapsAroundItsEnd(t *testing.T) {
	repository := fullOutbox()
	repository.oldestOutboxRecord = maximumOutboxRecords - 1
	saved(t, repository, "first")
	saved(t, repository, "second")
	assertOutboxEnds(t, repository, "1", "first", "second")
}

// Nothing reads the outbox yet, so it stays full, and copying it on every write
// would cost hundreds of kilobytes while every other request waits.
func TestAFullOutboxReusesItsMemory(t *testing.T) {
	repository := fullOutbox()
	firstRecord := &repository.outbox[0]
	saved(t, repository, "groceries")
	if &repository.outbox[0] != firstRecord {
		t.Error("the full outbox was copied")
	}
}

// withNotes stores that many notes, without going through Save.
func withNotes(count int) *InMemoryNoteRepository {
	repository := NewInMemoryNoteRepository()
	for index := range count {
		repository.records[strconv.Itoa(index)] = noteRecord{version: 1}
	}
	return repository
}

func TestANewNoteIsRefusedOnceTheLimitIsReached(t *testing.T) {
	repository := withNotes(maximumNotes - 1)
	saved(t, repository, "last")
	stored := len(repository.outbox)
	if err := repository.Save(context.Background(), newNote(t, "extra")); !errors.Is(err, application.ErrNoteLimitReached) {
		t.Errorf("err = %v", err)
	}
	if len(repository.records) != maximumNotes || len(repository.outbox) != stored {
		t.Errorf("the refused note left %d records and %d events", len(repository.records), len(repository.outbox))
	}
}

func TestAStoredNoteCanStillChangeAtTheLimit(t *testing.T) {
	repository := withNotes(maximumNotes - 1)
	note := saved(t, repository, "last")
	if err := repository.Save(context.Background(), archive(t, note)); err != nil {
		t.Errorf("err = %v", err)
	}
}

func TestRemovingANoteMakesRoomForAnother(t *testing.T) {
	repository := withNotes(maximumNotes - 1)
	note := saved(t, repository, "last")
	if err := repository.Remove(context.Background(), note); err != nil {
		t.Fatal(err)
	}
	if err := repository.Save(context.Background(), newNote(t, "next")); err != nil {
		t.Errorf("err = %v", err)
	}
}

func TestListingSeparatesActiveFromArchivedNotes(t *testing.T) {
	repository := NewInMemoryNoteRepository()
	saved(t, repository, "active")
	archived, err := saved(t, repository, "old").Archive(editedAt)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Save(context.Background(), archived); err != nil {
		t.Fatal(err)
	}
	for wantArchived, wantID := range map[bool]string{false: "active", true: "old"} {
		notes, err := repository.List(context.Background(), wantArchived)
		if err != nil {
			t.Fatal(err)
		}
		if len(notes) != 1 || notes[0].ID().String() != wantID {
			t.Errorf("archived = %v gave %+v", wantArchived, notes)
		}
	}
}

func withReminderDueAt(t *testing.T, repository *InMemoryNoteRepository, id string, dueAt time.Time) {
	t.Helper()
	note, err := saved(t, repository, id).ScheduleReminder(dueAt, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Save(context.Background(), note); err != nil {
		t.Fatal(err)
	}
}

func TestListingDueRemindersFindsOnlyDueNotes(t *testing.T) {
	repository := NewInMemoryNoteRepository()
	withReminderDueAt(t, repository, "overdue", editedAt.Add(-time.Minute))
	withReminderDueAt(t, repository, "exactly-due", editedAt)
	withReminderDueAt(t, repository, "not-due-yet", editedAt.Add(time.Nanosecond))
	saved(t, repository, "no-reminder")
	notes, err := repository.ListDueReminders(context.Background(), editedAt)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, note := range notes {
		ids = append(ids, note.ID().String())
	}
	slices.Sort(ids)
	if want := []string{"exactly-due", "overdue"}; !slices.Equal(ids, want) {
		t.Errorf("ids = %v, want %v", ids, want)
	}
}

func TestDamagedDataIsNotAClientMistake(t *testing.T) {
	repository := NewInMemoryNoteRepository()
	note := saved(t, repository, "groceries")
	record := repository.records["groceries"]
	record.title = ""
	record.remindAt = createdAt
	repository.records["groceries"] = record
	_, findError := repository.FindByID(context.Background(), note.ID())
	_, listError := repository.List(context.Background(), false)
	_, dueError := repository.ListDueReminders(context.Background(), editedAt)
	for _, err := range []error{findError, listError, dueError} {
		if err == nil || errors.Is(err, domain.ErrTitleRequired) || errors.Is(err, application.ErrNoteNotFound) {
			t.Errorf("err = %v", err)
		}
	}
}

func TestACancelledRequestReachesNoStorage(t *testing.T) {
	repository := NewInMemoryNoteRepository()
	note := saved(t, repository, "groceries")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, findError := repository.FindByID(ctx, note.ID())
	_, listError := repository.List(ctx, false)
	_, dueError := repository.ListDueReminders(ctx, editedAt)
	for name, err := range map[string]error{
		"save":   repository.Save(ctx, archive(t, note)),
		"remove": repository.Remove(ctx, note),
		"find":   findError,
		"list":   listError,
		"due":    dueError,
	} {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if found, err := repository.FindByID(context.Background(), note.ID()); err != nil || found.IsArchived() {
		t.Errorf("found = %+v, err = %v", found, err)
	}
}

func TestOverlappingSavesCannotBothWin(t *testing.T) {
	repository := NewInMemoryNoteRepository()
	note := saved(t, repository, "groceries")
	const attempts = 16
	results := make([]error, attempts)
	archived := archive(t, note)
	var group sync.WaitGroup
	for attempt := range attempts {
		group.Go(func() { results[attempt] = repository.Save(context.Background(), archived) })
	}
	group.Wait()
	successes := 0
	for _, err := range results {
		if err == nil {
			successes++
		} else if !errors.Is(err, application.ErrNoteModifiedConcurrently) {
			t.Errorf("err = %v", err)
		}
	}
	if successes != 1 {
		t.Errorf("%d saves won", successes)
	}
}
