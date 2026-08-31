package zerotrust

import (
	"testing"
	"time"
)

func TestIssueAndValidateToken(t *testing.T) {
	store := NewTokenStore()
	tok, err := store.Issue("operator-alice", ScopeOperator, DefaultTokenTTL)
	if err != nil {
		t.Fatal(err)
	}
	if tok.ID == "" {
		t.Fatal("token ID should not be empty")
	}

	validated, err := store.Validate(tok.ID, ScopeOperator)
	if err != nil {
		t.Fatal(err)
	}
	if validated.OwnerID != "operator-alice" {
		t.Fatalf("expected alice, got %s", validated.OwnerID)
	}
}

func TestExpiredToken(t *testing.T) {
	store := NewTokenStore()
	tok, _ := store.Issue("operator-bob", ScopeSession, 1*time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	if _, err := store.Validate(tok.ID, ScopeSession); err == nil {
		t.Fatal("expected expired token error")
	}
}

func TestWrongScope(t *testing.T) {
	store := NewTokenStore()
	tok, _ := store.Issue("operator-carol", ScopeSession, DefaultTokenTTL)
	if _, err := store.Validate(tok.ID, ScopeAdmin); err == nil {
		t.Fatal("session scope token should not satisfy admin scope")
	}
}

func TestAdminScopeSatisfiesAll(t *testing.T) {
	store := NewTokenStore()
	tok, _ := store.Issue("operator-dave", ScopeAdmin, DefaultTokenTTL)
	if _, err := store.Validate(tok.ID, ScopeSession); err != nil {
		t.Fatalf("admin token should satisfy session scope: %v", err)
	}
}

func TestRevoke(t *testing.T) {
	store := NewTokenStore()
	tok, _ := store.Issue("operator-eve", ScopeOperator, DefaultTokenTTL)
	store.Revoke(tok.ID)
	if _, err := store.Validate(tok.ID, ScopeOperator); err == nil {
		t.Fatal("revoked token should not validate")
	}
}

func TestRevokeAll(t *testing.T) {
	store := NewTokenStore()
	store.Issue("operator-frank", ScopeSession, DefaultTokenTTL)
	store.Issue("operator-frank", ScopeOperator, DefaultTokenTTL)
	store.Issue("operator-grace", ScopeSession, DefaultTokenTTL)

	n := store.RevokeAll("operator-frank")
	if n != 2 {
		t.Fatalf("expected 2 tokens revoked, got %d", n)
	}
	if store.Count() != 1 {
		t.Fatalf("expected 1 token remaining, got %d", store.Count())
	}
}

func TestSweepExpired(t *testing.T) {
	store := NewTokenStore()
	store.Issue("operator-henry", ScopeSession, 1*time.Millisecond)
	store.Issue("operator-henry", ScopeOperator, DefaultTokenTTL)
	time.Sleep(20 * time.Millisecond)
	swept := store.Sweep()
	if swept != 1 {
		t.Fatalf("expected 1 swept, got %d", swept)
	}
}

func TestCertLedger(t *testing.T) {
	ledger := NewCertLedger()
	ledger.Register("implant-1", "deadbeef", DefaultCertTTL)

	if err := ledger.Verify("deadbeef"); err != nil {
		t.Fatal(err)
	}

	if err := ledger.Verify("unknown-fp"); err == nil {
		t.Fatal("unknown fingerprint should fail verify")
	}

	if err := ledger.Revoke("deadbeef"); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Verify("deadbeef"); err == nil {
		t.Fatal("revoked cert should fail verify")
	}
}
