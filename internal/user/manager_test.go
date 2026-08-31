package user

import (
	"testing"
)

func TestCreateAndAuthenticate(t *testing.T) {
	m := NewManager()
	acc, err := m.Create("alice", "supersecret", RoleOperator)
	if err != nil {
		t.Fatal(err)
	}
	if acc.Username != "alice" {
		t.Fatalf("expected alice, got %s", acc.Username)
	}

	got, err := m.Authenticate("alice", "supersecret")
	if err != nil {
		t.Fatal(err)
	}
	if got.Username != "alice" {
		t.Fatalf("expected alice, got %s", got.Username)
	}
}

func TestAuthenticateBadPassword(t *testing.T) {
	m := NewManager()
	m.Create("bob", "correct-horse", RoleViewer)
	if _, err := m.Authenticate("bob", "wrong-password"); err == nil {
		t.Fatal("expected authentication failure with wrong password")
	}
}

func TestDuplicateCreate(t *testing.T) {
	m := NewManager()
	m.Create("carol", "pass1", RoleOperator)
	if _, err := m.Create("carol", "pass2", RoleAdmin); err == nil {
		t.Fatal("expected error for duplicate username")
	}
}

func TestDisableAccount(t *testing.T) {
	m := NewManager()
	m.Create("dave", "password", RoleOperator)
	m.Disable("dave")
	if _, err := m.Authenticate("dave", "password"); err == nil {
		t.Fatal("expected error for disabled account")
	}
}

func TestEngagementScope(t *testing.T) {
	m := NewManager()
	m.Create("eve", "password", RoleOperator)

	if err := m.GrantEngagement("eve", "eng-001"); err != nil {
		t.Fatal(err)
	}
	if !m.HasEngagement("eve", "eng-001") {
		t.Fatal("expected eve to have access to eng-001")
	}
	if m.HasEngagement("eve", "eng-002") {
		t.Fatal("eve should not have access to eng-002")
	}

	m.RevokeEngagement("eve", "eng-001")
	if m.HasEngagement("eve", "eng-001") {
		t.Fatal("expected eng-001 to be revoked")
	}
}

func TestChangePassword(t *testing.T) {
	m := NewManager()
	m.Create("frank", "old-pass", RoleAdmin)
	if err := m.ChangePassword("frank", "old-pass", "new-pass"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Authenticate("frank", "new-pass"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Authenticate("frank", "old-pass"); err == nil {
		t.Fatal("old password should no longer work")
	}
}

func TestSetRole(t *testing.T) {
	m := NewManager()
	m.Create("grace", "pass", RoleViewer)
	if err := m.SetRole("grace", RoleAdmin); err != nil {
		t.Fatal(err)
	}
	for _, acc := range m.List() {
		if acc.Username == "grace" && acc.Role != RoleAdmin {
			t.Fatalf("expected admin role, got %s", acc.Role)
		}
	}
}
