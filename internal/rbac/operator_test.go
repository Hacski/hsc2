package rbac

import (
	"fmt"
	"sync"
	"testing"
)

func TestSessionOwnershipTransfer(t *testing.T) {
	so := NewSessionOwnership()
	if !so.Claim("sess-1", "", "alice") {
		t.Fatal("initial claim must succeed")
	}
	if so.Claim("sess-1", "bob", "steve") {
		t.Fatal("non-holder must not claim")
	}
	if !so.Claim("sess-1", "alice", "bob") {
		t.Fatal("holder alice must transfer to bob")
	}
	h, ok := so.Holder("sess-1")
	if !ok || h != "bob" {
		t.Fatalf("expected bob, got %q", h)
	}
}

func TestConcurrentClaims(t *testing.T) {
	so := NewSessionOwnership()
	var wg sync.WaitGroup
	wins := map[string]int{}
	var mu sync.Mutex
	so.Claim("sess-c", "", "seed")
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			who := fmt.Sprintf("op-%d", n)
			if so.Claim("sess-c", "seed", who) {
				mu.Lock()
				wins[who]++
				mu.Unlock()
				so.Claim("sess-c", who, "seed")
			}
		}(i)
	}
	wg.Wait()
	holder, _ := so.Holder("sess-c")
	if holder != "seed" {
		t.Fatalf("expected final holder seed, got %s", holder)
	}
}
