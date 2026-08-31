package audit

import (
	"bytes"
	"os"
	"testing"
)

func TestChainIntegrity(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/audit.json"
	key := []byte("test-key-001")

	l, err := New(path, key)
	if err != nil {
		t.Fatal(err)
	}
	l.Record("alice", "exec", "sess-1", "run whoami")
	l.Record("bob", "download", "sess-2", "loot file.txt")
	l.Record("alice", "kill", "sess-1", "kill session")

	ok, err := l.Verify()
	if err != nil || !ok {
		t.Fatalf("chain should verify: ok=%v err=%v", ok, err)
	}

	l2, err := New(path, key)
	if err != nil {
		t.Fatal(err)
	}
	l2.Record("carol", "upload", "sess-3", "stage payload")

	ok, _ = l2.Verify()
	if !ok {
		t.Fatal("chain should still verify after appending via second instance")
	}

	data, _ := os.ReadFile(path)
	tampered := bytes.Replace(data, []byte("run whoami"), []byte("run rm -rf /"), 1)
	if err := os.WriteFile(path, tampered, 0600); err != nil {
		t.Fatal(err)
	}

	fresh, _ := New(path, key)
	ok, _ = fresh.Verify()
	if ok {
		t.Fatal("tampered chain must NOT verify")
	}
}
