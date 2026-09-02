package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/algonormative/bosun/internal/config"
)

func newTestServer() *Server {
	cfg := &config.Config{
		AgentName:    "test-agent",
		AgentCommand: "echo hello",
		ServeAddr:    ":0",
	}
	return New(cfg)
}

func TestHealth_OK(t *testing.T) {
	s := newTestServer()
	// Mark as running to get "ok"
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	s.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %q", body["status"])
	}
}

func TestHealth_Degraded(t *testing.T) {
	s := newTestServer()
	// running is false by default

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	s.handleHealth(w, req)

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "degraded" {
		t.Errorf("expected status degraded, got %q", body["status"])
	}
}

func TestStatus(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()
	s.handleStatus(w, req)

	var st Status
	json.NewDecoder(w.Body).Decode(&st)
	if st.State != "stopped" {
		t.Errorf("expected state stopped, got %q", st.State)
	}
	if st.AgentName != "test-agent" {
		t.Errorf("expected agent name test-agent, got %q", st.AgentName)
	}
}

func TestStop_AlreadyStopped(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/stop", nil)
	w := httptest.NewRecorder()
	s.handleStop(w, req)

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "already_stopped" {
		t.Errorf("expected already_stopped, got %q", body["status"])
	}
}
