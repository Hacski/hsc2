package log

import (
	"strings"
	"testing"
)

func TestRedactPassword(t *testing.T) {
	r := New()
	out := r.Redact(`login password=hunter2 session active`)
	if strings.Contains(out, "hunter2") {
		t.Fatalf("password leaked: %q", out)
	}
	if !strings.Contains(out, "REDACTED") {
		t.Fatalf("expected redaction marker: %q", out)
	}
}

func TestRedactURLCredentials(t *testing.T) {
	r := New()
	out := r.Redact(`endpoint https://admin:s3cret@db.internal:5432/`)
	if strings.Contains(out, "s3cret") || strings.Contains(out, "admin") {
		t.Fatalf("url creds leaked: %q", out)
	}
}

func TestRedactAWSKey(t *testing.T) {
	r := New()
	out := r.Redact(`key=AKIAIOSFODNN7EXAMPLE`)
	if strings.Contains(out, "AKIAIOSFODNN7EXAMPLE") {
		t.Fatalf("aws key leaked: %q", out)
	}
}

func TestRedactPrivateKey(t *testing.T) {
	r := New()
	key := "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAC\n-----END RSA PRIVATE KEY-----"
	out := r.Redact("creds:\n" + key + "\nend")
	if strings.Contains(out, "MIIEowIBAAC") {
		t.Fatalf("private key leaked: %q", out)
	}
}

func TestRedactEmail(t *testing.T) {
	r := New()
	out := r.Redact("contact support@example.com now")
	if strings.Contains(out, "support@example.com") {
		t.Fatalf("email leaked: %q", out)
	}
}

func TestRedactToken(t *testing.T) {
	r := New()
	out := r.Redact("Authorization: Bearer abcdef1234567890xyz")
	if strings.Contains(out, "abcdef1234567890xyz") {
		t.Fatalf("token leaked: %q", out)
	}
}

func TestMaskKeepTail(t *testing.T) {
	r := New().KeepTail(4)
	m := r.Mask("SuperSecretPassword123")
	if strings.Contains(m, "SuperSecretPassword") {
		t.Fatalf("mask leaked prefix: %q", m)
	}
	if !strings.HasSuffix(m, "123") {
		t.Fatalf("mask should keep tail: %q", m)
	}
}

func TestRedactBytes(t *testing.T) {
	r := New()
	red := r.RedactBytes([]byte("pwd=topsecret"))
	if strings.Contains(string(red), "topsecret") {
		t.Fatal("bytes redaction failed")
	}
}

func TestNoRedactionOfPlain(t *testing.T) {
	r := New()
	out := r.Redact("operator issued task to session-12 on target")
	if !strings.Contains(out, "session-12") {
		t.Fatalf("harmless content should be untouched: %q", out)
	}
}
