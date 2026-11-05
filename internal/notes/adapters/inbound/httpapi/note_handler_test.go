package httpapi_test

import (
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"
)

func listIDs(t *testing.T, url string) []string {
	t.Helper()
	response := call(t, http.MethodGet, url, jsonType, "")
	assertStatus(t, response, http.StatusOK)
	notes, ok := response.body["notes"].([]any)
	if !ok {
		t.Fatalf("notes = %v", response.body["notes"])
	}
	ids := []string{}
	for _, note := range notes {
		ids = append(ids, note.(map[string]any)["id"].(string))
	}
	return ids
}

func TestANoteGoesThroughItsWholeLifeOverHTTP(t *testing.T) {
	server := newServer(t)
	created := call(t, http.MethodPost, server.URL+"/notes", jsonType, groceries)
	assertStatus(t, created, http.StatusCreated)
	if created.header.Get("Content-Type") != jsonType {
		t.Errorf("content type = %q", created.header.Get("Content-Type"))
	}
	for field, value := range map[string]any{"title": "Groceries", "body": "Milk", "archived": false} {
		if created.body[field] != value {
			t.Errorf("%s = %v", field, created.body[field])
		}
	}
	note := "/notes/" + created.body["id"].(string)
	if got := call(t, http.MethodGet, server.URL+note, jsonType, ""); got.body["title"] != "Groceries" {
		t.Errorf("got = %v", got.body)
	}

	edited := call(t, http.MethodPut, server.URL+note, jsonType, `{"title":"Shopping","body":"Eggs"}`)
	assertStatus(t, edited, http.StatusOK)
	if edited.body["title"] != "Shopping" || edited.body["body"] != "Eggs" {
		t.Errorf("edited = %v", edited.body)
	}

	archived := call(t, http.MethodPost, server.URL+note+"/archive", jsonType, "")
	assertStatus(t, archived, http.StatusOK)
	if archived.body["archived"] != true {
		t.Errorf("archived = %v", archived.body)
	}
	restored := call(t, http.MethodPost, server.URL+note+"/restore", jsonType, "")
	assertStatus(t, restored, http.StatusOK)
	if restored.body["archived"] != false {
		t.Errorf("restored = %v", restored.body)
	}

	deleted := call(t, http.MethodDelete, server.URL+note, jsonType, "")
	assertStatus(t, deleted, http.StatusNoContent)
	gone := call(t, http.MethodGet, server.URL+note, jsonType, "")
	assertStandardError(t, gone, http.StatusNotFound, "business", "NOTE_NOT_FOUND")
}

// Clients depend on these names, so adding or dropping one must be a decision.
func TestANoteHasExactlyItsDocumentedFields(t *testing.T) {
	response := call(t, http.MethodPost, newServer(t).URL+"/notes", jsonType, groceries)
	assertStatus(t, response, http.StatusCreated)
	fields := slices.Sorted(maps.Keys(response.body))
	if want := []string{"archived", "body", "createdAt", "editedAt", "id", "title"}; !slices.Equal(fields, want) {
		t.Errorf("fields = %v, want %v", fields, want)
	}
}

func TestTimesLeaveTheAPIInUTC(t *testing.T) {
	server := newServer(t)
	created := call(t, http.MethodPost, server.URL+"/notes", jsonType, groceries)
	for _, field := range []string{"createdAt", "editedAt"} {
		text, ok := created.body[field].(string)
		if !ok || !strings.HasSuffix(text, "Z") {
			t.Errorf("%s = %v", field, created.body[field])
			continue
		}
		if _, err := time.Parse(time.RFC3339Nano, text); err != nil {
			t.Error(err)
		}
	}
}

func TestAnEmptyBodyIsAllowedButMustBeSent(t *testing.T) {
	server := newServer(t)
	assertStatus(t, call(t, http.MethodPost, server.URL+"/notes", jsonType, `{"title":"Groceries","body":""}`), http.StatusCreated)
}

func TestAnArchivedNoteMovesToTheArchivedList(t *testing.T) {
	server := newServer(t)
	note := createNote(t, server)
	if ids := listIDs(t, server.URL+"/notes"); len(ids) != 1 {
		t.Fatalf("ids = %v", ids)
	}
	if ids := listIDs(t, server.URL+"/notes?archived=true"); len(ids) != 0 {
		t.Fatalf("ids = %v", ids)
	}
	assertStatus(t, call(t, http.MethodPost, server.URL+note+"/archive", jsonType, ""), http.StatusOK)
	if ids := listIDs(t, server.URL+"/notes?archived=false"); len(ids) != 0 {
		t.Fatalf("ids = %v", ids)
	}
	if ids := listIDs(t, server.URL+"/notes?archived=true"); len(ids) != 1 {
		t.Fatalf("ids = %v", ids)
	}
}

func TestTheMostRecentlyEditedNoteIsListedFirst(t *testing.T) {
	server := newServer(t)
	older := createNote(t, server)
	newer := createNote(t, server)
	if ids := listIDs(t, server.URL+"/notes"); "/notes/"+ids[0] != newer {
		t.Errorf("ids = %v, newer = %s", ids, newer)
	}
	assertStatus(t, call(t, http.MethodPut, server.URL+older, jsonType, groceries), http.StatusOK)
	if ids := listIDs(t, server.URL+"/notes"); "/notes/"+ids[0] != older {
		t.Errorf("ids = %v, older = %s", ids, older)
	}
}

// Every character of the largest valid note can need twelve bytes of JSON.
func TestTheLargestValidNoteFitsTheBodyLimit(t *testing.T) {
	const escapedCharacter = `😀`
	server := newServer(t)
	content := `{"title":"` + strings.Repeat(escapedCharacter, 200) + `","body":"` + strings.Repeat(escapedCharacter, 10000) + `"}`
	assertStatus(t, call(t, http.MethodPost, server.URL+"/notes", jsonType, content), http.StatusCreated)
}

func TestEscapesThatOnlyLookLikeSurrogatesAreKept(t *testing.T) {
	server := newServer(t)
	testCases := map[string]string{
		`\\ud800`:         `\ud800`,
		`\\\ud83d\ude00`:  "\\😀",
		`\ud83d\ude00`:    "😀",
		`\u0041\ufffd`:    "A\ufffd",
		`line\nnext\ttab`: "line\nnext\ttab",
	}
	for escaped, want := range testCases {
		response := call(t, http.MethodPost, server.URL+"/notes", jsonType, `{"title":"a","body":"`+escaped+`"}`)
		assertStatus(t, response, http.StatusCreated)
		if response.body["body"] != want {
			t.Errorf("%s became %q, want %q", escaped, response.body["body"], want)
		}
	}
}

func TestACharsetOfUTF8IsAccepted(t *testing.T) {
	server := newServer(t)
	assertStatus(t, call(t, http.MethodPost, server.URL+"/notes", "application/json; charset=UTF-8", groceries), http.StatusCreated)
}

func TestAMissingContentTypeIsTreatedAsJSON(t *testing.T) {
	server := newServer(t)
	request, err := http.NewRequest(http.MethodPost, server.URL+"/notes", strings.NewReader(groceries))
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, send(t, request), http.StatusCreated)
}

func TestCreatingANoteSaysWhereItIs(t *testing.T) {
	server := newServer(t)
	created := call(t, http.MethodPost, server.URL+"/notes", jsonType, groceries)
	assertStatus(t, created, http.StatusCreated)
	location := created.header.Get("Location")
	if want := "/notes/" + created.body["id"].(string); location != want {
		t.Fatalf("Location = %q, want %q", location, want)
	}
	assertStatus(t, call(t, http.MethodGet, server.URL+location, jsonType, ""), http.StatusOK)
}

// percentEncoded writes the first character of an ID as an escape, which RFC
// 3986 treats as the same URL.
func percentEncoded(note string) string {
	id := strings.TrimPrefix(note, "/notes/")
	return fmt.Sprintf("/notes/%%%02X%s", id[0], id[1:])
}

func TestEveryRouteDecodesAPercentEncodedID(t *testing.T) {
	server := newServer(t)
	note := percentEncoded(createNote(t, server))
	assertStatus(t, call(t, http.MethodGet, server.URL+note, jsonType, ""), http.StatusOK)
	assertStatus(t, call(t, http.MethodPut, server.URL+note, jsonType, groceries), http.StatusOK)
	assertStatus(t, call(t, http.MethodPost, server.URL+note+"/archive", jsonType, ""), http.StatusOK)
	assertStatus(t, call(t, http.MethodPost, server.URL+note+"/restore", jsonType, ""), http.StatusOK)
	assertStatus(t, call(t, http.MethodDelete, server.URL+note, jsonType, ""), http.StatusNoContent)
}

func TestAnIDIsDecodedOnlyOnce(t *testing.T) {
	server := newServer(t)
	escaped := strings.Replace(percentEncoded(createNote(t, server)), "%", "%25", 1)
	assertStandardError(t, call(t, http.MethodGet, server.URL+escaped, jsonType, ""), http.StatusNotFound, "business", "NOTE_NOT_FOUND")
}

func TestSpaceAroundAnIDIsPartOfTheID(t *testing.T) {
	server := newServer(t)
	spaced := strings.Replace(createNote(t, server), "/notes/", "/notes/%20", 1)
	assertStandardError(t, call(t, http.MethodGet, server.URL+spaced, jsonType, ""), http.StatusNotFound, "business", "NOTE_NOT_FOUND")
}

func TestABrokenEscapeMatchesNoNote(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/notes/some-note", nil)
	request.URL.RawPath = "/notes/%zz"
	recorder := httptest.NewRecorder()
	newServer(t).Config.Handler.ServeHTTP(recorder, request)
	assertStandardError(t, recorded(t, recorder), http.StatusNotFound, "business", "NOTE_NOT_FOUND")
}

func TestTheHealthCheckResponds(t *testing.T) {
	assertStatus(t, call(t, http.MethodGet, newServer(t).URL+"/healthz", jsonType, ""), http.StatusNoContent)
}
