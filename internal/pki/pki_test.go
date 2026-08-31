package pki

import (
	"testing"
)

func TestGenerateAndLoadCA(t *testing.T) {
	dir := t.TempDir()
	m, err := GenerateCA(dir, "hsc2-test")
	if err != nil {
		t.Fatal(err)
	}
	if m.CAPin == "" {
		t.Fatal("expected a ca pin")
	}
	m2, err := LoadCA(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m2.CAPin != m.CAPin {
		t.Fatal("ca pin must persist")
	}
}

func TestIssueServerAndOperator(t *testing.T) {
	dir := t.TempDir()
	m, _ := GenerateCA(dir, "hsc2-test")
	serverCert, serverKey, err := m.Issue(CertServer, "c2.example.com", []string{"c2.example.com", "127.0.0.1"}, 30)
	if err != nil {
		t.Fatal(err)
	}
	opCert, opKey, err := m.Issue(CertOperator, "alice", nil, 7)
	if err != nil {
		t.Fatal(err)
	}
	if DecodeCert(serverCert) == nil || DecodeCert(opCert) == nil {
		t.Fatal("certs did not parse")
	}
	srv := BuildServerTLS(m.CAPem, serverCert, serverKey)
	if srv == nil {
		t.Fatal("server tls build failed")
	}
	client, err := BuildClientTLS(m.CAPem, opCert, opKey, m.CAPin)
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("client tls build returned nil")
	}
}

func TestPinningRejectsWrongPeer(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	m1, _ := GenerateCA(dir1, "a")
	m2, _ := GenerateCA(dir2, "b")
	sc, _, _ := m2.Issue(CertServer, "s", []string{"s"}, 30)
	_, _, _ = m1.Issue(CertOperator, "op", nil, 30)
	// A peer presenting m2's server cert must fail verification when its SPKI
	// does not match the pinned m1 pin.
	if VerifyPin(m1.CAPin, sc) {
		t.Fatal("m2 server cert must not match m1 pin")
	}
}
