package selfdestruct

import (
	"sync"
	"testing"
	"time"
)

func TestCommandKill(t *testing.T) {
	var mu sync.Mutex
	var reason string
	c := New(Policy{}, func(r string) {
		mu.Lock()
		reason = r
		mu.Unlock()
	})
	c.Arm()
	c.Kill()
	if !c.IsActivated() {
		t.Fatal("should be activated after kill")
	}
	mu.Lock()
	if reason != "command_kill" {
		t.Fatalf("wrong reason %q", reason)
	}
	mu.Unlock()
}

func TestMaxBeaconsSelfDestruct(t *testing.T) {
	var mu sync.Mutex
	reasons := []string{}
	c := New(Policy{MaxBeacons: 3}, func(r string) {
		mu.Lock()
		reasons = append(reasons, r)
		mu.Unlock()
	})
	c.Arm()
	c.RecordBeacon()
	c.RecordBeacon()
	c.RecordBeacon()
	if !c.IsActivated() {
		t.Fatal("implant must self-destruct after max beacons")
	}
}

func TestKillDateSelfDestruct(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	var mu sync.Mutex
	reason := ""
	c := New(Policy{KillDate: &past}, func(r string) {
		mu.Lock()
		reason = r
		mu.Unlock()
	})
	c.Arm()
	c.RecordBeacon()
	mu.Lock()
	if reason != "kill_date_reached" {
		t.Fatalf("wrong reason %q", reason)
	}
	mu.Unlock()
}

func TestMaxIdleSelfDestruct(t *testing.T) {
	var mu sync.Mutex
	reason := ""
	c := New(Policy{MaxIdle: 100 * time.Millisecond}, func(r string) {
		mu.Lock()
		reason = r
		mu.Unlock()
	})
	c.Arm()
	time.Sleep(150 * time.Millisecond)
	c.RecordBeacon()
	mu.Lock()
	if reason != "max_idle" {
		t.Fatalf("wrong reason %q", reason)
	}
	mu.Unlock()
}

func TestDisarmPrevents(t *testing.T) {
	c := New(Policy{MaxBeacons: 1}, nil)
	c.Arm()
	c.Disarm()
	c.RecordBeacon()
	if c.IsActivated() {
		t.Fatal("disarmed controller must not self-destruct")
	}
}

func TestActivationOnlyOnce(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	c := New(Policy{MaxBeacons: 1}, func(string) {
		mu.Lock()
		calls++
		mu.Unlock()
	})
	c.Arm()
	c.RecordBeacon()
	c.Kill()
	c.Kill()
	mu.Lock()
	if calls != 1 {
		t.Fatalf("destroy callback must fire once, got %d", calls)
	}
	mu.Unlock()
}
