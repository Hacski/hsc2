package storage

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func readFileRaw(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func contains(haystack []byte, needle string) bool {
	return bytes.Contains(haystack, []byte(needle))
}

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	key, err := NewInstallKey()
	if err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir, key)
	if err != nil {
		t.Fatal(err)
	}
	cred := Credential{ID: "c1", Target: "dc01", Username: "svc_account", Secret: "SuperS3cret!", Kind: "password", Captured: time.Now()}
	if err := s.Put(KindCreds, cred.ID, cred); err != nil {
		t.Fatal(err)
	}
	var got Credential
	if err := s.Get(KindCreds, cred.ID, &got); err != nil {
		t.Fatal(err)
	}
	if got.Secret != "SuperS3cret!" {
		t.Fatalf("secret mismatch: %q", got.Secret)
	}

	ids, _ := s.List(KindCreds)
	if len(ids) != 1 || ids[0] != "c1" {
		t.Fatalf("list mismatch: %v", ids)
	}
}

func TestAtRestCiphertextDoesNotContainPlaintext(t *testing.T) {
	dir := t.TempDir()
	key, _ := NewInstallKey()
	s, _ := Open(dir, key)
	secret := "T0pS3cr3t-Creds"
	cred := Credential{ID: "c2", Username: "u", Secret: secret}
	if err := s.Put(KindCreds, cred.ID, cred); err != nil {
		t.Fatal(err)
	}
	data, err := readFileRaw(dir + "/creds/c2.enc")
	if err != nil {
		t.Fatal(err)
	}
	if contains(data, secret) {
		t.Fatal("plaintext secret must not appear in ciphertext at rest")
	}
}

func TestWrongKeyFails(t *testing.T) {
	dir := t.TempDir()
	k1, _ := NewInstallKey()
	s, _ := Open(dir, k1)
	if err := s.Put(KindLoot, "l1", Loot{ID: "l1", Data: []byte("x")}); err != nil {
		t.Fatal(err)
	}
	k2, _ := NewInstallKey()
	s2, _ := Open(dir, k2)
	var out Loot
	if err := s2.Get(KindLoot, "l1", &out); err == nil {
		t.Fatal("wrong key must fail to decrypt")
	}
}

func TestInstallKeyPersistence(t *testing.T) {
	dir := t.TempDir()
	k1, err := LoadOrCreateInstallKey(dir)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := LoadOrCreateInstallKey(dir)
	if err != nil {
		t.Fatal(err)
	}
	if string(k1) != string(k2) {
		t.Fatal("install key must persist across loads")
	}
}
