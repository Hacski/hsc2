package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

//go:embed templates
var templateFS embed.FS

type DashboardData struct {
	Sessions    []SessionRow
	ServerTime  string
	Version     string
	ActiveCount int
}

type SessionRow struct {
	ID       string
	Hostname string
	OS       string
	Arch     string
	Username string
	LastSeen string
	State    string
}

type APIClient interface {
	Sessions() ([]map[string]interface{}, error)
}

type Handler struct {
	api     APIClient
	version string
	mux     *http.ServeMux
	tmpl    *template.Template
}

func New(api APIClient, version string) (*Handler, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	h := &Handler{
		api:     api,
		version: version,
		mux:     http.NewServeMux(),
		tmpl:    tmpl,
	}
	h.mux.HandleFunc("/", h.handleDashboard)
	h.mux.HandleFunc("/sessions", h.handleSessions)
	h.mux.HandleFunc("/api/sessions", h.handleAPISessions)
	return h, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	sessions, err := h.api.Sessions()
	if err != nil {
		http.Error(w, "failed to fetch sessions", http.StatusBadGateway)
		return
	}
	rows := sessionsToRows(sessions)
	data := DashboardData{
		Sessions:    rows,
		ServerTime:  time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
		Version:     h.version,
		ActiveCount: len(rows),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func (h *Handler) handleSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.api.Sessions()
	if err != nil {
		http.Error(w, "failed to fetch sessions", http.StatusBadGateway)
		return
	}
	rows := sessionsToRows(sessions)
	data := DashboardData{
		Sessions:    rows,
		ServerTime:  time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
		Version:     h.version,
		ActiveCount: len(rows),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.tmpl.ExecuteTemplate(w, "sessions.html", data)
}

func (h *Handler) handleAPISessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.api.Sessions()
	if err != nil {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func sessionsToRows(sessions []map[string]interface{}) []SessionRow {
	rows := make([]SessionRow, 0, len(sessions))
	for _, s := range sessions {
		row := SessionRow{
			ID:       stringField(s, "id"),
			Hostname: stringField(s, "hostname"),
			OS:       stringField(s, "os"),
			Arch:     stringField(s, "arch"),
			Username: stringField(s, "username"),
			LastSeen: stringField(s, "last_seen"),
			State:    "active",
		}
		rows = append(rows, row)
	}
	return rows
}

func stringField(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
