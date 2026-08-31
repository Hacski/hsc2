package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Session struct {
	ID        string    `json:"id"`
	Hostname  string    `json:"hostname"`
	OS        string    `json:"os"`
	Arch      string    `json:"arch"`
	Username  string    `json:"username"`
	PID       int       `json:"pid"`
	LastSeen  time.Time `json:"last_seen"`
	BeaconID  string    `json:"beacon_id"`
}

type Task struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Command   string    `json:"command"`
	Args      []string  `json:"args"`
	Queued    time.Time `json:"queued"`
	Status    string    `json:"status"`
}

type TaskResult struct {
	TaskID    string    `json:"task_id"`
	Output    string    `json:"output"`
	ExitCode  int       `json:"exit_code"`
	Error     string    `json:"error,omitempty"`
	Completed time.Time `json:"completed"`
}

type Store interface {
	Sessions() ([]*Session, error)
	GetSession(id string) (*Session, error)
	QueueTask(t *Task) error
	GetTasks(sessionID string) ([]*Task, error)
	GetResult(taskID string) (*TaskResult, error)
}

type Handler struct {
	mu      sync.RWMutex
	store   Store
	version string
}

func New(store Store, version string) *Handler {
	return &Handler{store: store, version: version}
}

func (h *Handler) Mount(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/sessions", h.handleSessions)
	mux.HandleFunc("/api/v1/sessions/", h.handleSession)
	mux.HandleFunc("/api/v1/tasks/", h.handleTasks)
	mux.HandleFunc("/api/v1/results/", h.handleResults)
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/api/v1/version", h.handleVersion)
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"version": h.version})
}

func (h *Handler) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		sessions, err := h.store.Sessions()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, sessions)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleSession(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/sessions/")
	id = strings.TrimSuffix(id, "/tasks")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/tasks") {
		h.handleSessionTasks(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodGet:
		sess, err := h.store.GetSession(id)
		if err != nil {
			writeErr(w, http.StatusNotFound, "session not found")
			return
		}
		writeJSON(w, http.StatusOK, sess)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleSessionTasks(w http.ResponseWriter, r *http.Request, sessionID string) {
	switch r.Method {
	case http.MethodGet:
		tasks, err := h.store.GetTasks(sessionID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, tasks)
	case http.MethodPost:
		var t Task
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		t.SessionID = sessionID
		t.Queued = time.Now().UTC()
		t.Status = "queued"
		if err := h.store.QueueTask(&t); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, t)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleTasks(w http.ResponseWriter, r *http.Request) {
	taskID := strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/")
	if taskID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		tasks, err := h.store.GetTasks("")
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, t := range tasks {
			if t.ID == taskID {
				writeJSON(w, http.StatusOK, t)
				return
			}
		}
		writeErr(w, http.StatusNotFound, "task not found")
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleResults(w http.ResponseWriter, r *http.Request) {
	taskID := strings.TrimPrefix(r.URL.Path, "/api/v1/results/")
	if taskID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		result, err := h.store.GetResult(taskID)
		if err != nil {
			writeErr(w, http.StatusNotFound, "result not found")
			return
		}
		writeJSON(w, http.StatusOK, result)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
