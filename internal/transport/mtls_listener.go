package transport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"io"
	"net"
	"os"
	"sync"
)

type MTLSListener struct {
	name    string
	mu      sync.Mutex
	ln      net.Listener
	handler Handler
}

func NewMTLSListener(name string) *MTLSListener {
	return &MTLSListener{name: name}
}

func (m *MTLSListener) Name() string { return m.name }

func (m *MTLSListener) Listen(ctx context.Context, addr string, handler Handler) error {
	certFile := os.Getenv("HSC2_SERVER_CRT")
	keyFile := os.Getenv("HSC2_SERVER_KEY")
	if certFile == "" {
		certFile = DefaultCertFile
	}
	if keyFile == "" {
		keyFile = DefaultKeyFile
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}
	cfg := &tls.Config{Certificates: []tls.Certificate{cert}}
	if ca := os.Getenv("HSC2_CA_CRT"); ca != "" {
		pem, err := os.ReadFile(ca)
		if err != nil {
			return err
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(pem)
		cfg.ClientCAs = pool
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}
	raw, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	m.ln = tls.NewListener(raw, cfg)
	m.handler = handler
	for {
		conn, err := m.ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		go m.handleConn(conn)
	}
}

func (m *MTLSListener) handleConn(conn net.Conn) {
	defer conn.Close()
	for {
		var sz uint32
		if err := binary.Read(conn, binary.BigEndian, &sz); err != nil {
			if err == io.EOF {
				return
			}
			return
		}
		if sz > 16*1024*1024 {
			return
		}
		buf := make([]byte, sz)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return
		}
		resp, err := m.handler.Handle(context.Background(), buf)
		if err != nil {
			return
		}
		if err := binary.Write(conn, binary.BigEndian, uint32(len(resp))); err != nil {
			return
		}
		if _, err := conn.Write(resp); err != nil {
			return
		}
	}
}

func (m *MTLSListener) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ln != nil {
		return m.ln.Close()
	}
	return nil
}
