package httpapi_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/adapters/inbound/httpapi"
	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/adapters/outbound/memory"
	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/adapters/outbound/system"
	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/application"
)

const (
	jsonType    = "application/json"
	problemType = "application/problem+json"
	groceries   = `{"title":"Groceries","body":"Milk"}`
)

type apiResponse struct {
	status int
	header http.Header
	body   map[string]any
}

func newRouter(useCases application.NoteUseCasesPort, logs io.Writer) http.Handler {
	return httpapi.NewRouter(useCases, slog.New(slog.NewTextHandler(logs, nil)))
}

// newServer wires the real application and storage behind a real server.
func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	service := application.NewNoteApplicationService(memory.NewInMemoryNoteRepository(), system.SystemClock{}, system.RandomIDGenerator{})
	server := httptest.NewServer(newRouter(service, io.Discard))
	t.Cleanup(server.Close)
	return server
}

func call(t *testing.T, method, url, contentType, body string) apiResponse {
	t.Helper()
	request, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", contentType)
	return send(t, request)
}

func send(t *testing.T, request *http.Request) apiResponse {
	t.Helper()
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	content, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	decoded := map[string]any{}
	if len(content) > 0 {
		if err := json.Unmarshal(content, &decoded); err != nil {
			t.Fatalf("the body is not a JSON object: %q", content)
		}
	}
	return apiResponse{status: response.StatusCode, header: response.Header, body: decoded}
}

func recorded(t *testing.T, recorder *httptest.ResponseRecorder) apiResponse {
	t.Helper()
	decoded := map[string]any{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("the body is not a JSON object: %q", recorder.Body)
	}
	return apiResponse{status: recorder.Code, header: recorder.Header(), body: decoded}
}

func createNote(t *testing.T, server *httptest.Server) string {
	t.Helper()
	response := call(t, http.MethodPost, server.URL+"/notes", jsonType, groceries)
	if response.status != http.StatusCreated {
		t.Fatalf("create: %d %v", response.status, response.body)
	}
	return "/notes/" + response.body["id"].(string)
}

func assertStatus(t *testing.T, response apiResponse, status int) {
	t.Helper()
	if response.status != status {
		t.Fatalf("status = %d, want %d: %v", response.status, status, response.body)
	}
}

func assertStandardError(t *testing.T, response apiResponse, status int, category, code string) {
	t.Helper()
	if response.status != status {
		t.Errorf("status = %d, want %d: %v", response.status, status, response.body)
	}
	if contentType := response.header.Get("Content-Type"); contentType != problemType {
		t.Errorf("content type = %q", contentType)
	}
	for field, value := range map[string]any{"title": http.StatusText(status), "status": float64(status), "category": category, "code": code} {
		if response.body[field] != value {
			t.Errorf("%s = %v, want %v", field, response.body[field], value)
		}
	}
	for _, field := range []string{"detail", "requestId"} {
		if text, ok := response.body[field].(string); !ok || text == "" {
			t.Errorf("missing %s: %v", field, response.body)
		}
	}
	if response.header.Get("X-Request-Id") != response.body["requestId"] {
		t.Errorf("header %q does not match body %v", response.header.Get("X-Request-Id"), response.body["requestId"])
	}
}
