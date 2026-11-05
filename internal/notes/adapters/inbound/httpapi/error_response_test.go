package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/adapters/inbound/httpapi"
	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/application"
)

func TestEveryFailureReturnsTheStandardErrorResponse(t *testing.T) {
	server := newServer(t)
	note := createNote(t, server)
	archived := createNote(t, server)
	assertStatus(t, call(t, http.MethodPost, server.URL+archived+"/archive", jsonType, ""), http.StatusOK)

	const (
		business  = "business"
		technical = "technical"
	)
	testCases := []struct {
		name        string
		method      string
		path        string
		contentType string
		body        string
		status      int
		category    string
		code        string
	}{
		{"reading an unknown note", http.MethodGet, "/notes/missing", jsonType, "", http.StatusNotFound, business, "NOTE_NOT_FOUND"},
		{"editing an unknown note", http.MethodPut, "/notes/missing", jsonType, groceries, http.StatusNotFound, business, "NOTE_NOT_FOUND"},
		{"archiving an unknown note", http.MethodPost, "/notes/missing/archive", jsonType, "", http.StatusNotFound, business, "NOTE_NOT_FOUND"},
		{"restoring an unknown note", http.MethodPost, "/notes/missing/restore", jsonType, "", http.StatusNotFound, business, "NOTE_NOT_FOUND"},
		{"deleting an unknown note", http.MethodDelete, "/notes/missing", jsonType, "", http.StatusNotFound, business, "NOTE_NOT_FOUND"},
		{"blank note ID", http.MethodGet, "/notes/%20", jsonType, "", http.StatusUnprocessableEntity, business, "NOTE_ID_REQUIRED"},
		{"blank title", http.MethodPost, "/notes", jsonType, `{"title":"  ","body":""}`, http.StatusUnprocessableEntity, business, "NOTE_TITLE_REQUIRED"},
		{"blank title on edit", http.MethodPut, note, jsonType, `{"title":"","body":""}`, http.StatusUnprocessableEntity, business, "NOTE_TITLE_REQUIRED"},
		{"title of invisible characters", http.MethodPost, "/notes", jsonType, `{"title":"\u200b","body":""}`, http.StatusUnprocessableEntity, business, "NOTE_TITLE_REQUIRED"},
		{"long title", http.MethodPost, "/notes", jsonType, `{"title":"` + strings.Repeat("a", 201) + `","body":""}`, http.StatusUnprocessableEntity, business, "NOTE_TITLE_TOO_LONG"},
		{"long body", http.MethodPost, "/notes", jsonType, `{"title":"a","body":"` + strings.Repeat("a", 10001) + `"}`, http.StatusUnprocessableEntity, business, "NOTE_BODY_TOO_LONG"},
		{"NUL in a title", http.MethodPost, "/notes", jsonType, `{"title":"a\u0000b","body":""}`, http.StatusUnprocessableEntity, business, "NOTE_TITLE_HAS_CONTROL_CHARACTER"},
		{"line break in a title", http.MethodPost, "/notes", jsonType, `{"title":"a\nb","body":""}`, http.StatusUnprocessableEntity, business, "NOTE_TITLE_HAS_CONTROL_CHARACTER"},
		{"line break in a title on edit", http.MethodPut, note, jsonType, `{"title":"a\nb","body":""}`, http.StatusUnprocessableEntity, business, "NOTE_TITLE_HAS_CONTROL_CHARACTER"},
		{"NUL in a body", http.MethodPost, "/notes", jsonType, `{"title":"a","body":"a\u0000b"}`, http.StatusUnprocessableEntity, business, "NOTE_BODY_HAS_CONTROL_CHARACTER"},
		{"NUL in a body on edit", http.MethodPut, note, jsonType, `{"title":"a","body":"a\u0000b"}`, http.StatusUnprocessableEntity, business, "NOTE_BODY_HAS_CONTROL_CHARACTER"},
		{"archiving an archived note", http.MethodPost, archived + "/archive", jsonType, "", http.StatusConflict, business, "NOTE_ALREADY_ARCHIVED"},
		{"restoring an active note", http.MethodPost, note + "/restore", jsonType, "", http.StatusConflict, business, "NOTE_NOT_ARCHIVED"},
		{"editing an archived note", http.MethodPut, archived, jsonType, groceries, http.StatusConflict, business, "NOTE_ARCHIVED"},
		{"malformed JSON", http.MethodPost, "/notes", jsonType, "{", http.StatusBadRequest, technical, "MALFORMED_REQUEST"},
		{"empty body", http.MethodPost, "/notes", jsonType, "", http.StatusBadRequest, technical, "MALFORMED_REQUEST"},
		{"JSON that is not an object", http.MethodPost, "/notes", jsonType, "[]", http.StatusBadRequest, technical, "MALFORMED_REQUEST"},
		{"invalid UTF-8", http.MethodPost, "/notes", jsonType, "{\"title\":\"caf\xe9\",\"body\":\"\"}", http.StatusBadRequest, technical, "MALFORMED_REQUEST"},
		{"invalid UTF-8 on edit", http.MethodPut, note, jsonType, "{\"title\":\"a\",\"body\":\"\xff\xfe\"}", http.StatusBadRequest, technical, "MALFORMED_REQUEST"},
		{"lone high surrogate", http.MethodPost, "/notes", jsonType, `{"title":"a\ud800b","body":""}`, http.StatusBadRequest, technical, "MALFORMED_REQUEST"},
		{"lone low surrogate", http.MethodPost, "/notes", jsonType, `{"title":"a","body":"\udc00"}`, http.StatusBadRequest, technical, "MALFORMED_REQUEST"},
		{"reversed surrogate pair", http.MethodPost, "/notes", jsonType, `{"title":"a","body":"\ude00\ud83d"}`, http.StatusBadRequest, technical, "MALFORMED_REQUEST"},
		{"high surrogate before another escape", http.MethodPost, "/notes", jsonType, `{"title":"a","body":"\ud83d\u0041"}`, http.StatusBadRequest, technical, "MALFORMED_REQUEST"},
		{"high surrogate at the end of a string", http.MethodPost, "/notes", jsonType, `{"title":"a\ud83d","body":""}`, http.StatusBadRequest, technical, "MALFORMED_REQUEST"},
		{"missing field", http.MethodPost, "/notes", jsonType, `{"title":"a"}`, http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"null field", http.MethodPost, "/notes", jsonType, `{"title":null,"body":""}`, http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"null body", http.MethodPost, "/notes", jsonType, "null", http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"repeated field", http.MethodPost, "/notes", jsonType, `{"title":"a","title":"b","body":""}`, http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"repeated field on edit", http.MethodPut, note, jsonType, `{"title":"a","body":"","body":"b"}`, http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"field repeated through an escape", http.MethodPost, "/notes", jsonType, `{"title":"a","t\u0069tle":"b","body":""}`, http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"unknown field", http.MethodPost, "/notes", jsonType, `{"title":"a","body":"","archived":true}`, http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"a field in the wrong case", http.MethodPost, "/notes", jsonType, `{"Title":"a","title":"b","body":""}`, http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"wrong field type", http.MethodPost, "/notes", jsonType, `{"title":1,"body":""}`, http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"wrong field type on edit", http.MethodPut, note, jsonType, `{"title":"a","body":[]}`, http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"wrong content type", http.MethodPost, "/notes", "text/plain", groceries, http.StatusUnsupportedMediaType, technical, "UNSUPPORTED_MEDIA_TYPE"},
		{"unsupported charset", http.MethodPost, "/notes", "application/json; charset=utf-16", groceries, http.StatusUnsupportedMediaType, technical, "UNSUPPORTED_MEDIA_TYPE"},
		{"a body on reading a note", http.MethodGet, note, jsonType, "x", http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"a body on listing notes", http.MethodGet, "/notes", jsonType, "x", http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"a body on archiving", http.MethodPost, note + "/archive", jsonType, "x", http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"a body on restoring", http.MethodPost, archived + "/restore", jsonType, "x", http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"a body on deleting", http.MethodDelete, note, jsonType, "x", http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"a body on the health check", http.MethodGet, "/healthz", jsonType, "x", http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"an invalid archived filter", http.MethodGet, "/notes?archived=maybe", jsonType, "", http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"an empty archived filter", http.MethodGet, "/notes?archived=", jsonType, "", http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"a repeated archived filter", http.MethodGet, "/notes?archived=true&archived=false", jsonType, "", http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"an unknown filter", http.MethodGet, "/notes?color=red", jsonType, "", http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"a query with a semicolon", http.MethodGet, "/notes?archived=true;x=1", jsonType, "", http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"a query with a broken escape", http.MethodGet, "/notes?archived=%zz", jsonType, "", http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"an unknown filter hidden by a broken escape", http.MethodGet, "/notes?archived=true&color=%zz", jsonType, "", http.StatusUnprocessableEntity, technical, "VALIDATION_FAILED"},
		{"body too large", http.MethodPost, "/notes", jsonType, `{"title":"` + strings.Repeat("a", 1<<18) + `"}`, http.StatusRequestEntityTooLarge, technical, "REQUEST_TOO_LARGE"},
		{"unknown route", http.MethodGet, "/nothing-here", jsonType, "", http.StatusNotFound, technical, "ROUTE_NOT_FOUND"},
		{"wrong method", http.MethodPatch, "/notes", jsonType, "", http.StatusMethodNotAllowed, technical, "METHOD_NOT_ALLOWED"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assertStandardError(t, call(t, testCase.method, server.URL+testCase.path, testCase.contentType, testCase.body), testCase.status, testCase.category, testCase.code)
		})
	}
}

func TestAMethodNotAllowedResponseListsTheAllowedMethods(t *testing.T) {
	server := newServer(t)
	testCases := map[string]string{
		"/notes":                   "GET, POST",
		"/notes/some-note":         "GET, PUT, DELETE",
		"/notes/some-note/archive": "POST",
		"/healthz":                 "GET",
		"/notes/a%2Fb":             "GET, PUT, DELETE",
		"/notes/a%2Farchive":       "GET, PUT, DELETE",
		"/notes/%41BC/archive":     "POST",
	}
	for path, allowed := range testCases {
		response := call(t, http.MethodPatch, server.URL+path, jsonType, "")
		assertStandardError(t, response, http.StatusMethodNotAllowed, "technical", "METHOD_NOT_ALLOWED")
		if got := response.header.Get("Allow"); got != allowed {
			t.Errorf("%s: Allow = %q, want %q", path, got, allowed)
		}
	}
}

func TestValidationErrorsNameEachProblemWithoutEchoingInput(t *testing.T) {
	server := newServer(t)
	testCases := []struct {
		name      string
		url       string
		method    string
		body      string
		locations []string
	}{
		{"missing and unknown fields", "/notes", http.MethodPost, `{"title":"a","private text":1}`, []string{"body", "body"}},
		{"wrong type", "/notes", http.MethodPost, `{"title":"a","body":42424242}`, []string{"body.body"}},
		{"unknown query parameter", "/notes?private=1", http.MethodGet, "", []string{"query"}},
		{"bad filter value", "/notes?archived=private", http.MethodGet, "", []string{"query.archived"}},
		{"broken query", "/notes?archived=%zz-private", http.MethodGet, "", []string{"query"}},
		{"repeated field", "/notes", http.MethodPost, `{"title":"private","title":"b","body":""}`, []string{"body"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			response := call(t, testCase.method, server.URL+testCase.url, jsonType, testCase.body)
			assertStandardError(t, response, http.StatusUnprocessableEntity, "technical", "VALIDATION_FAILED")
			problems, ok := response.body["errors"].([]any)
			if !ok {
				t.Fatalf("errors = %v", response.body["errors"])
			}
			var locations []string
			for _, problem := range problems {
				fields := problem.(map[string]any)
				locations = append(locations, fields["location"].(string))
				if message := fields["message"].(string); message == "" || strings.Contains(message, "private") || strings.Contains(message, "42424242") {
					t.Errorf("message = %q", message)
				}
			}
			if !slices.Equal(locations, testCase.locations) {
				t.Errorf("locations = %q, want %q", locations, testCase.locations)
			}
		})
	}
}

func TestMalformedJSONIsNotEchoed(t *testing.T) {
	response := call(t, http.MethodPost, newServer(t).URL+"/notes", jsonType, `{private text`)
	assertStandardError(t, response, http.StatusBadRequest, "technical", "MALFORMED_REQUEST")
	message := response.body["errors"].([]any)[0].(map[string]any)["message"].(string)
	if strings.Contains(message, "private") || strings.Contains(message, "'p'") {
		t.Errorf("message = %q", message)
	}
}

func TestBusinessErrorsNeverEchoTheNoteID(t *testing.T) {
	const clientInput = "private-text"
	response := call(t, http.MethodGet, newServer(t).URL+"/notes/"+clientInput, jsonType, "")
	assertStandardError(t, response, http.StatusNotFound, "business", "NOTE_NOT_FOUND")
	if detail := response.body["detail"].(string); strings.Contains(detail, clientInput) {
		t.Errorf("detail = %q", detail)
	}
}

type stubUseCases struct {
	application.NoteUseCasesPort
	createNote  func() (application.NoteViewDTO, error)
	archiveNote func() (application.NoteViewDTO, error)
	listNotes   func() ([]application.NoteViewDTO, error)
}

func (stub stubUseCases) CreateNote(context.Context, application.CreateNoteCommand) (application.NoteViewDTO, error) {
	return stub.createNote()
}

func (stub stubUseCases) ListNotes(context.Context, application.ListNotesQuery) ([]application.NoteViewDTO, error) {
	return stub.listNotes()
}

func (stub stubUseCases) ArchiveNote(context.Context, application.ArchiveNoteCommand) (application.NoteViewDTO, error) {
	return stub.archiveNote()
}

func stubServer(t *testing.T, archiveNote func() (application.NoteViewDTO, error)) (*httptest.Server, *strings.Builder) {
	t.Helper()
	logs := &strings.Builder{}
	server := httptest.NewServer(newRouter(stubUseCases{archiveNote: archiveNote}, logs))
	t.Cleanup(server.Close)
	return server, logs
}

const stubbedArchive = "/notes/some-note/archive"

func TestConcurrentChangesAreReportedAsAConflict(t *testing.T) {
	server, _ := stubServer(t, func() (application.NoteViewDTO, error) {
		return application.NoteViewDTO{}, application.ErrNoteModifiedConcurrently
	})
	assertStandardError(t, call(t, http.MethodPost, server.URL+stubbedArchive, jsonType, ""), http.StatusConflict, "business", "NOTE_MODIFIED_CONCURRENTLY")
}

func TestUnexpectedErrorsAreLoggedButNeverShown(t *testing.T) {
	server, logs := stubServer(t, func() (application.NoteViewDTO, error) {
		return application.NoteViewDTO{}, errors.New("secret connection string")
	})
	response := call(t, http.MethodPost, server.URL+stubbedArchive, jsonType, "")
	assertStandardError(t, response, http.StatusInternalServerError, "technical", "INTERNAL_ERROR")
	if response.body["detail"] != "internal error" {
		t.Errorf("detail = %v", response.body["detail"])
	}
	if !strings.Contains(logs.String(), "secret connection string") || !strings.Contains(logs.String(), response.body["requestId"].(string)) {
		t.Errorf("logs = %s", logs)
	}
}

func TestPanicsReturnTheStandardErrorResponse(t *testing.T) {
	server, logs := stubServer(t, func() (application.NoteViewDTO, error) { panic("secret panic value") })
	response := call(t, http.MethodPost, server.URL+stubbedArchive, jsonType, "")
	assertStandardError(t, response, http.StatusInternalServerError, "technical", "INTERNAL_ERROR")
	if strings.Contains(response.body["detail"].(string), "secret") {
		t.Errorf("detail = %v", response.body["detail"])
	}
	for _, expected := range []string{"secret panic value", response.body["requestId"].(string), "status=500"} {
		if !strings.Contains(logs.String(), expected) {
			t.Errorf("%q is missing from the logs: %s", expected, logs)
		}
	}
}

func TestAnAbortedResponseStaysAborted(t *testing.T) {
	server, _ := stubServer(t, func() (application.NoteViewDTO, error) { panic(http.ErrAbortHandler) })
	response, err := http.Post(server.URL+stubbedArchive, jsonType, nil)
	if err == nil {
		_ = response.Body.Close()
		t.Errorf("the connection stayed open, with status %d", response.StatusCode)
	}
}

func TestAnAbortedRequestIsStillLogged(t *testing.T) {
	server, logs := stubServer(t, func() (application.NoteViewDTO, error) { panic(http.ErrAbortHandler) })
	request, err := http.NewRequest(http.MethodPost, server.URL+stubbedArchive, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-Request-Id", "trace-123")
	if response, err := http.DefaultClient.Do(request); err == nil {
		_ = response.Body.Close()
	}
	// Closing waits for the handler, so the logs are safe to read.
	server.Close()
	for _, expected := range []string{"requestId=trace-123", "method=POST", "path=" + stubbedArchive} {
		if !strings.Contains(logs.String(), expected) {
			t.Errorf("%q is missing from the logs: %q", expected, logs)
		}
	}
}

func TestAResponseThatCantBeEncodedReturnsTheStandardErrorResponse(t *testing.T) {
	unencodable := application.NoteViewDTO{ID: "some-note", CreatedAt: time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)}
	logs := &strings.Builder{}
	server := httptest.NewServer(newRouter(stubUseCases{
		createNote:  func() (application.NoteViewDTO, error) { return unencodable, nil },
		archiveNote: func() (application.NoteViewDTO, error) { return unencodable, nil },
		listNotes:   func() ([]application.NoteViewDTO, error) { return []application.NoteViewDTO{unencodable}, nil },
	}, logs))
	t.Cleanup(server.Close)
	testCases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"a created note", http.MethodPost, "/notes", groceries},
		{"a changed note", http.MethodPost, stubbedArchive, ""},
		{"a list of notes", http.MethodGet, "/notes", ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			response := call(t, testCase.method, server.URL+testCase.path, jsonType, testCase.body)
			assertStandardError(t, response, http.StatusInternalServerError, "technical", "INTERNAL_ERROR")
			if location := response.header.Get("Location"); location != "" {
				t.Errorf("Location = %q", location)
			}
		})
	}
	if strings.Contains(logs.String(), "status=2") {
		t.Errorf("logs = %s", logs)
	}
}

func TestTheNoteLimitIsReportedAsAConflict(t *testing.T) {
	server := httptest.NewServer(newRouter(stubUseCases{createNote: func() (application.NoteViewDTO, error) {
		return application.NoteViewDTO{}, application.ErrNoteLimitReached
	}}, io.Discard))
	t.Cleanup(server.Close)
	assertStandardError(t, call(t, http.MethodPost, server.URL+"/notes", jsonType, groceries), http.StatusConflict, "business", "NOTE_LIMIT_REACHED")
}

func TestAFailedListIsLoggedButNeverShown(t *testing.T) {
	logs := &strings.Builder{}
	server := httptest.NewServer(newRouter(stubUseCases{listNotes: func() ([]application.NoteViewDTO, error) {
		return nil, errors.New("secret damaged record")
	}}, logs))
	t.Cleanup(server.Close)
	response := call(t, http.MethodGet, server.URL+"/notes", jsonType, "")
	assertStandardError(t, response, http.StatusInternalServerError, "technical", "INTERNAL_ERROR")
	if strings.Contains(response.body["detail"].(string), "secret") || !strings.Contains(logs.String(), "secret damaged record") {
		t.Errorf("detail = %v, logs = %s", response.body["detail"], logs)
	}
}

type failingBody struct{}

func (failingBody) Read([]byte) (int, error) { return 0, errors.New("connection lost") }

func TestAFailedBodyReadIsReportedByItsCause(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	testCases := []struct {
		name   string
		ctx    context.Context
		status int
		code   string
	}{
		{"the client left", cancelled, http.StatusRequestTimeout, "REQUEST_CANCELLED"},
		{"the read failed", context.Background(), http.StatusBadRequest, "MALFORMED_REQUEST"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(testCase.ctx, http.MethodPost, "/notes", failingBody{})
			newRouter(stubUseCases{}, io.Discard).ServeHTTP(recorder, request)
			assertStandardError(t, recorded(t, recorder), testCase.status, "technical", testCase.code)
		})
	}
}

func TestAnAPIErrorCanStopARequestFromInsideTheAdapter(t *testing.T) {
	server, _ := stubServer(t, func() (application.NoteViewDTO, error) {
		return application.NoteViewDTO{}, httpapi.NewBusinessError(http.StatusConflict, "CUSTOM_CODE", "custom detail")
	})
	assertStandardError(t, call(t, http.MethodPost, server.URL+stubbedArchive, jsonType, ""), http.StatusConflict, "business", "CUSTOM_CODE")
}

func TestACancelledRequestIsNotAServerFault(t *testing.T) {
	for _, cancelled := range []error{context.Canceled, context.DeadlineExceeded} {
		server, logs := stubServer(t, func() (application.NoteViewDTO, error) {
			return application.NoteViewDTO{}, fmt.Errorf("while saving: %w", cancelled)
		})
		response := call(t, http.MethodPost, server.URL+stubbedArchive, jsonType, "")
		assertStandardError(t, response, http.StatusRequestTimeout, "technical", "REQUEST_CANCELLED")
		if strings.Contains(logs.String(), "level=ERROR") {
			t.Errorf("logs = %s", logs)
		}
	}
}

func requestWithID(t *testing.T, url, requestID string) *http.Request {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-Request-Id", requestID)
	return request
}

func TestASafeRequestIDFollowsTheRequest(t *testing.T) {
	server := newServer(t)
	for _, requestID := range []string{"abc-123", "A_b.c-9", strings.Repeat("a", 128)} {
		response := send(t, requestWithID(t, server.URL+"/nothing-here", requestID))
		if response.body["requestId"] != requestID || response.header.Get("X-Request-Id") != requestID {
			t.Errorf("sent %q, got %v and %q", requestID, response.body["requestId"], response.header.Get("X-Request-Id"))
		}
	}
}

func TestAnUnsafeRequestIDIsReplaced(t *testing.T) {
	server := newServer(t)
	for _, requestID := range []string{"has space", `quote"id`, "a=b", "ünicode", strings.Repeat("a", 129)} {
		response := send(t, requestWithID(t, server.URL+"/nothing-here", requestID))
		if response.body["requestId"] == requestID || response.body["requestId"] == "" {
			t.Errorf("sent %q, got %v", requestID, response.body["requestId"])
		}
		if response.header.Get("X-Request-Id") != response.body["requestId"] {
			t.Errorf("header %q does not match body %v", response.header.Get("X-Request-Id"), response.body["requestId"])
		}
	}
}

func TestGeneratedRequestIDsAreUniqueAndHideTheHostname(t *testing.T) {
	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}
	server := newServer(t)
	first := call(t, http.MethodGet, server.URL+"/nothing-here", jsonType, "")
	second := call(t, http.MethodGet, server.URL+"/nothing-here", jsonType, "")
	if first.body["requestId"] == second.body["requestId"] {
		t.Errorf("both requests have the ID %v", first.body["requestId"])
	}
	if strings.Contains(first.body["requestId"].(string), hostname) {
		t.Errorf("request ID %v contains the hostname", first.body["requestId"])
	}
}

func TestEverySuccessfulResponseCarriesARequestID(t *testing.T) {
	server := newServer(t)
	for _, response := range []apiResponse{
		call(t, http.MethodPost, server.URL+"/notes", jsonType, groceries),
		call(t, http.MethodGet, server.URL+"/healthz", jsonType, ""),
	} {
		if response.header.Get("X-Request-Id") == "" {
			t.Errorf("missing X-Request-Id: %v", response.header)
		}
	}
}

func TestEverySuccessfulRequestIsLoggedWithItsRequestID(t *testing.T) {
	server, logs := stubServer(t, func() (application.NoteViewDTO, error) {
		return application.NoteViewDTO{ID: "some-note"}, nil
	})
	request, err := http.NewRequest(http.MethodPost, server.URL+stubbedArchive, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-Request-Id", "trace-123")
	assertStatus(t, send(t, request), http.StatusOK)
	for _, expected := range []string{"requestId=trace-123", "method=POST", "path=" + stubbedArchive, "status=200"} {
		if !strings.Contains(logs.String(), expected) {
			t.Errorf("%q is missing from the logs: %s", expected, logs)
		}
	}
}

type overlappingResult struct {
	status int
	code   string
	err    error
}

// attemptArchive runs on its own goroutine, where t.Fatal is not allowed, so it
// reports failures in its result.
func attemptArchive(url string) overlappingResult {
	response, err := http.Post(url, jsonType, nil)
	if err != nil {
		return overlappingResult{err: err}
	}
	defer func() { _ = response.Body.Close() }()
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return overlappingResult{err: err}
	}
	return overlappingResult{status: response.StatusCode, code: body.Code}
}

func TestOverlappingRequestsCannotBothChangeANote(t *testing.T) {
	server := newServer(t)
	note := createNote(t, server)
	const attempts = 16
	results := make(chan overlappingResult, attempts)
	start := make(chan struct{})
	for range attempts {
		go func() {
			<-start
			results <- attemptArchive(server.URL + note + "/archive")
		}()
	}
	close(start)
	successes := 0
	for range attempts {
		result := <-results
		switch {
		case result.err != nil:
			t.Error(result.err)
		case result.status == http.StatusOK:
			successes++
		case result.code != "NOTE_ALREADY_ARCHIVED" && result.code != "NOTE_MODIFIED_CONCURRENTLY":
			t.Errorf("status = %d, code = %q", result.status, result.code)
		}
	}
	if successes != 1 {
		t.Errorf("%d overlapping requests succeeded", successes)
	}
}
