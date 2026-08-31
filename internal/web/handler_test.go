package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockAPI struct {
	sessions []map[string]interface{}
	err      error
}

func (m *mockAPI) Sessions() ([]map[string]interface{}, error) {
	return m.sessions, m.err
}

func TestDashboardRenders(t *testing.T) {
	api := &mockAPI{
		sessions: []map[string]interface{}{
			{"id": "s1", "hostname": "victim", "os": "linux", "arch": "amd64", "username": "root", "last_seen": "2026-01-01T00:00:00Z"},
		},
	}
	h, err := New(api, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if len(body) == 0 {
		t.Fatal("expected non-empty response body")
	}
}

func TestAPISessionsJSON(t *testing.T) {
	api := &mockAPI{
		sessions: []map[string]interface{}{
			{"id": "s1", "hostname": "target"},
		},
	}
	h, err := New(api, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var out []map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 session, got %d", len(out))
	}
}

func TestDashboardUpstreamError(t *testing.T) {
	api := &mockAPI{err: errors.New("upstream down")}
	h, err := New(api, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", rec.Code)
	}
}
