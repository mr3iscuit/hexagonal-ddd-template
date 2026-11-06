package memory

import (
	"testing"
	"time"
)

var (
	createdAt = time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	editedAt  = createdAt.Add(time.Hour)
)

func validRecord() noteRecord {
	return noteRecord{id: "groceries", title: "Groceries", body: "Milk", createdAt: createdAt, editedAt: editedAt, version: 3}
}

func TestRecordRoundTripKeepsEveryField(t *testing.T) {
	for _, record := range []noteRecord{
		validRecord(),
		{id: "old", title: "Old", archived: true, createdAt: createdAt, editedAt: createdAt, version: 1},
		{id: "soon", title: "Soon", remindAt: editedAt.Add(time.Hour), createdAt: createdAt, editedAt: createdAt, version: 2},
	} {
		note, err := toDomain(record)
		if err != nil {
			t.Fatal(err)
		}
		if got := toRecord(note); got != record {
			t.Errorf("got %+v, want %+v", got, record)
		}
	}
}

func TestDamagedRecordsAreNeverReportedAsAClientMistake(t *testing.T) {
	blankID, blankTitle, noCreationTime, longBody := validRecord(), validRecord(), validRecord(), validRecord()
	blankID.id = " "
	blankTitle.title = ""
	noCreationTime.createdAt = time.Time{}
	longBody.body = string(make([]byte, 10001))
	damaged := map[string]noteRecord{"blank ID": blankID, "blank title": blankTitle, "no creation time": noCreationTime, "long body": longBody}
	for name, record := range damaged {
		t.Run(name, func(t *testing.T) {
			_, err := toDomain(record)
			if err == nil {
				t.Fatal("the damaged record was accepted")
			}
			if unwrappable, ok := err.(interface{ Unwrap() error }); ok && unwrappable.Unwrap() != nil {
				t.Errorf("err = %v wraps a domain error", err)
			}
		})
	}
}

func TestOutboxRecordsKeepEveryEventField(t *testing.T) {
	note, err := toDomain(validRecord())
	if err != nil {
		t.Fatal(err)
	}
	note, err = note.Edit(note.Title(), note.Body(), editedAt)
	if err != nil {
		t.Fatal(err)
	}
	note, err = note.Archive(editedAt)
	if err != nil {
		t.Fatal(err)
	}
	records := toOutboxRecords(note)
	want := []outboxRecord{{"note.edited", "groceries", editedAt}, {"note.archived", "groceries", editedAt}}
	if len(records) != len(want) || records[0] != want[0] || records[1] != want[1] {
		t.Errorf("records = %+v", records)
	}
}
