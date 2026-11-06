// Package e2e_test runs the real binary over real HTTP.
package e2e_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

const (
	startupTimeout  = 15 * time.Second
	shutdownTimeout = 15 * time.Second
	pollInterval    = 50 * time.Millisecond
)

// lockedBuffer can be read while the server is still writing to it.
type lockedBuffer struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
}

func (locked *lockedBuffer) Write(content []byte) (int, error) {
	locked.mutex.Lock()
	defer locked.mutex.Unlock()
	return locked.buffer.Write(content)
}

func (locked *lockedBuffer) String() string {
	locked.mutex.Lock()
	defer locked.mutex.Unlock()
	return locked.buffer.String()
}

// listeningAddress finds the port the server chose in its startup log.
func listeningAddress(output string) string {
	for line := range strings.Lines(output) {
		var entry struct {
			Message string `json:"msg"`
			Address string `json:"address"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err == nil && entry.Message == "listening" {
			return entry.Address
		}
	}
	return ""
}

func waitForAddress(t *testing.T, output *lockedBuffer) string {
	t.Helper()
	deadline := time.Now().Add(startupTimeout)
	for time.Now().Before(deadline) {
		if address := listeningAddress(output.String()); address != "" {
			return address
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("the server did not start: %s", output)
	return ""
}

func request(t *testing.T, method, url, body string) (int, map[string]any) {
	t.Helper()
	outgoing, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(outgoing)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	decoded := map[string]any{}
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil && response.StatusCode != http.StatusNoContent {
		t.Fatal(err)
	}
	return response.StatusCode, decoded
}

type runningServer struct {
	baseURL string
	command *exec.Cmd
	output  *lockedBuffer
	exited  <-chan error
}

func startServer(t *testing.T) runningServer {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "api")
	if output, err := exec.Command("go", "build", "-o", binary, "../../cmd/api").CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, output)
	}
	output := &lockedBuffer{}
	command := exec.Command(binary)
	command.Env = append(os.Environ(), "ADDR=127.0.0.1:0")
	command.Stdout = output
	command.Stderr = output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan error, 1)
	go func() { exited <- command.Wait() }()
	t.Cleanup(func() { _ = command.Process.Kill() })
	return runningServer{baseURL: "http://" + waitForAddress(t, output), command: command, output: output, exited: exited}
}

func (server runningServer) stopCleanly(t *testing.T) {
	t.Helper()
	if err := server.command.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-server.exited:
		if err != nil {
			t.Errorf("the server did not exit cleanly: %v\n%s", err, server.output)
		}
	case <-time.After(shutdownTimeout):
		t.Fatalf("the server did not stop: %s", server.output)
	}
}

func TestTheRealBinaryServesNotesAndStopsCleanly(t *testing.T) {
	server := startServer(t)
	baseURL := server.baseURL

	status, created := request(t, http.MethodPost, baseURL+"/notes", `{"title":"Groceries","body":"Milk"}`)
	if status != http.StatusCreated {
		t.Fatalf("create: %d %v", status, created)
	}
	note := baseURL + "/notes/" + created["id"].(string)
	if status, list := request(t, http.MethodGet, baseURL+"/notes", ""); status != http.StatusOK || len(list["notes"].([]any)) != 1 {
		t.Errorf("list: %d %v", status, list)
	}
	if status, archived := request(t, http.MethodPost, note+"/archive", ""); status != http.StatusOK || archived["archived"] != true {
		t.Errorf("archive: %d %v", status, archived)
	}
	if status, conflict := request(t, http.MethodPost, note+"/archive", ""); status != http.StatusConflict || conflict["code"] != "NOTE_ALREADY_ARCHIVED" {
		t.Errorf("second archive: %d %v", status, conflict)
	}
	if status, list := request(t, http.MethodGet, baseURL+"/notes?archived=true", ""); status != http.StatusOK || len(list["notes"].([]any)) != 1 {
		t.Errorf("list: %d %v", status, list)
	}
	if status, _ := request(t, http.MethodDelete, note, ""); status != http.StatusNoContent {
		t.Errorf("delete: %d", status)
	}
	if status, missing := request(t, http.MethodGet, note, ""); status != http.StatusNotFound || missing["code"] != "NOTE_NOT_FOUND" {
		t.Errorf("read after delete: %d %v", status, missing)
	}
	if !strings.Contains(server.output.String(), `"msg":"request handled"`) {
		t.Errorf("the request was not logged: %s", server.output)
	}
	server.stopCleanly(t)
}
