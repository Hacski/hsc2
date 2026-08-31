package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type memStore struct {
	mu      sync.RWMutex
	sessions map[string]*Session
	tasks    []*Task
	results  map[string]*TaskResult
}

func newMemStore() *memStore {
	return &memStore{
		sessions: map[string]*Session{},
		results:  map[string]*TaskResult{},
	}
}

func (m *memStore) Sessions() ([]*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, s)
	}
	return out, nil
}

func (m *memStore) GetSession(id string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return s, nil
}

func (m *memStore) QueueTask(t *Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks = append(m.tasks, t)
	return nil
}

func (m *memStore) GetTasks(sessionID string) ([]*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*Task
	for _, t := range m.tasks {
		if sessionID == "" || t.SessionID == sessionID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (m *memStore) GetResult(taskID string) (*TaskResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.results[taskID]
	if !ok {
		return nil, errors.New("not found")
	}
	return r, nil
}

func TestHealthEndpoint(t *testing.T) {
	h := New(newMemStore(), "test-1")
	mux := http.NewServeMux()
	h.Mount(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestSessionsEndpoint(t *testing.T) {
	store := newMemStore()
	store.sessions["sess-1"] = &Session{
		ID: "sess-1", Hostname: "victim-host", OS: "linux",
		Arch: "amd64", Username: "root", LastSeen: time.Now(),
	}

	h := New(store, "test-1")
	mux := http.NewServeMux()
	h.Mount(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var sessions []*Session
	if err := json.NewDecoder(rec.Body).Decode(&sessions); err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
}

func TestQueueTask(t *testing.T) {
	store := newMemStore()
	store.sessions["sess-2"] = &Session{ID: "sess-2"}

	h := New(store, "test-1")
	mux := http.NewServeMux()
	h.Mount(mux)

	body, _ := json.Marshal(Task{ID: "task-1", Command: "whoami"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/sess-2/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	tasks, _ := store.GetTasks("sess-2")
	if len(tasks) != 1 || tasks[0].Command != "whoami" {
		t.Fatalf("expected queued task, got %v", tasks)
	}
}

func TestResultNotFound(t *testing.T) {
	h := New(newMemStore(), "test-1")
	mux := http.NewServeMux()
	h.Mount(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/results/no-such-task", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
