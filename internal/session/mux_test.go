package session

import (
	"testing"
	"time"
)

func TestRegisterAndGet(t *testing.T) {
	mux := NewMux()
	sess := &Session{
		ID:         "sess-1",
		OperatorID: "op-alice",
		BeaconID:   "beacon-1",
		Hostname:   "victim",
		OS:         "linux",
		Arch:       "amd64",
		Username:   "root",
		PID:        1234,
	}
	mux.Register(sess)

	got, err := mux.Get("sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Hostname != "victim" {
		t.Fatalf("expected victim, got %s", got.Hostname)
	}
	if got.State != StateActive {
		t.Fatalf("expected active state, got %s", got.State)
	}
}

func TestForOperator(t *testing.T) {
	mux := NewMux()
	mux.Register(&Session{ID: "s1", OperatorID: "op-alice"})
	mux.Register(&Session{ID: "s2", OperatorID: "op-alice"})
	mux.Register(&Session{ID: "s3", OperatorID: "op-bob"})

	alice := mux.ForOperator("op-alice")
	if len(alice) != 2 {
		t.Fatalf("expected 2 sessions for alice, got %d", len(alice))
	}
	bob := mux.ForOperator("op-bob")
	if len(bob) != 1 {
		t.Fatalf("expected 1 session for bob, got %d", len(bob))
	}
}

func TestKill(t *testing.T) {
	mux := NewMux()
	mux.Register(&Session{ID: "s1", OperatorID: "op-alice"})

	if err := mux.Kill("s1"); err != nil {
		t.Fatal(err)
	}
	if _, err := mux.Get("s1"); err == nil {
		t.Fatal("expected error after kill")
	}
	if len(mux.ForOperator("op-alice")) != 0 {
		t.Fatal("killed session should be removed from operator index")
	}
}

func TestTaskQueue(t *testing.T) {
	mux := NewMux()
	mux.Register(&Session{ID: "s1", OperatorID: "op-alice"})

	sess, _ := mux.Get("s1")
	sess.EnqueueTask("task-1")
	sess.EnqueueTask("task-2")

	if sess.PendingTasks() != 2 {
		t.Fatalf("expected 2 pending, got %d", sess.PendingTasks())
	}
	id, ok := sess.DequeueTask()
	if !ok || id != "task-1" {
		t.Fatalf("expected task-1, got %s", id)
	}
}

func TestReap(t *testing.T) {
	mux := NewMux()
	mux.Register(&Session{ID: "s-old", OperatorID: "op-alice"})
	mux.Register(&Session{ID: "s-new", OperatorID: "op-alice"})

	old, _ := mux.Get("s-old")
	old.LastSeen = time.Now().UTC().Add(-10 * time.Minute)

	dead := mux.Reap(5 * time.Minute)
	if len(dead) != 1 || dead[0] != "s-old" {
		t.Fatalf("expected s-old to be reaped, got %v", dead)
	}
	if _, err := mux.Get("s-old"); err == nil {
		t.Fatal("expected s-old to be gone after reap")
	}
	if _, err := mux.Get("s-new"); err != nil {
		t.Fatal("s-new should still be alive")
	}
}
