package pki

import (
	"crypto/tls"
	"testing"
	"time"
)

func TestMutualTLSHandshake(t *testing.T) {
	dir := t.TempDir()
	m, _ := GenerateCA(dir, "hsc2-test")
	srvCert, srvKey, _ := m.Issue(CertServer, "localhost", []string{"localhost", "127.0.0.1"}, 30)
	opCert, opKey, _ := m.Issue(CertOperator, "alice", nil, 7)

	srvCfg := BuildServerTLS(m.CAPem, srvCert, srvKey)
	ln, err := tls.Listen("tcp", "127.0.0.1:0", srvCfg.Config)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	serverErr := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer conn.Close()
		// server requires client cert; absence fails before here
		conn.SetDeadline(time.Now().Add(3 * time.Second))
		buf := make([]byte, 4)
		_, err = conn.Read(buf)
		serverErr <- err
	}()

	cliCfg, err := BuildClientTLS(m.CAPem, opCert, opKey, SPKIPinFromPEM(srvCert))
	if err != nil {
		t.Fatal(err)
	}
	conn, err := tls.Dial("tcp", ln.Addr().String(), cliCfg)
	if err != nil {
		t.Fatalf("mTLS dial failed: %v", err)
	}
	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	conn.Close()

	if err := <-serverErr; err != nil {
		t.Fatalf("server side handshake/read error: %v", err)
	}
}

func TestTLSRejectsUntrustedClient(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	m1, _ := GenerateCA(dir1, "trusted")
	m2, _ := GenerateCA(dir2, "rogue")
	srvCert, srvKey, _ := m1.Issue(CertServer, "localhost", []string{"localhost", "127.0.0.1"}, 30)
	rogueCert, rogueKey, _ := m2.Issue(CertOperator, "intruder", nil, 7)

	srvCfg := BuildServerTLS(m1.CAPem, srvCert, srvKey)
	ln, err := tls.Listen("tcp", "127.0.0.1:0", srvCfg.Config)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			// handshake will fail because client is unsigned by our CA
			c.SetDeadline(time.Now().Add(2 * time.Second))
			go func() { c.Read(make([]byte, 4)) }()
		}
	}()

	cliCfg, _ := BuildClientTLS(m2.CAPem, rogueCert, rogueKey, m2.CAPin)
	conn, err := tls.Dial("tcp", ln.Addr().String(), cliCfg)
	if err == nil || err.Error() == "" {
		t.Fatal("rogue client signed by a different CA must be rejected")
	}
	if conn != nil {
		conn.Close()
	}
}
