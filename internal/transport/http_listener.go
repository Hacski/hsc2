package transport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
)

const DefaultCertFile = "server.crt"
const DefaultKeyFile = "server.key"

type HTTPListener struct {
	name   string
	server *http.Server
	mu     sync.Mutex
	closed bool
}

func NewHTTPListener(name string) *HTTPListener {
	return &HTTPListener{name: name}
}

func (h *HTTPListener) Name() string { return h.name }

func (h *HTTPListener) Listen(ctx context.Context, addr string, handler Handler) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}
		resp, err := handler.Handle(ctx, body)
		if err != nil {
			http.Error(w, "handler error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(resp)
	})
	h.server = &http.Server{Addr: addr, Handler: mux}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return h.server.Serve(ln)
}

func (h *HTTPListener) ListenTLS(ctx context.Context, addr, certFile, keyFile string, handler Handler, caFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}
	cfg := &tls.Config{Certificates: []tls.Certificate{cert}}
	if caFile != "" {
		cas := x509.NewCertPool()
		pem, err := os.ReadFile(caFile)
		if err != nil {
			return err
		}
		if !cas.AppendCertsFromPEM(pem) {
			return fmt.Errorf("failed to parse CA certs")
		}
		cfg.ClientCAs = cas
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		resp, err := handler.Handle(ctx, body)
		if err != nil {
			http.Error(w, "handler error", http.StatusInternalServerError)
			return
		}
		w.Write(resp)
	})
	h.server = &http.Server{Addr: addr, Handler: mux, TLSConfig: cfg}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return h.server.ServeTLS(ln, "", "")
}

func (h *HTTPListener) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil
	}
	h.closed = true
	if h.server != nil {
		return h.server.Close()
	}
	return nil
}

func EncodeFrame(b []byte) []byte {
	return []byte(base64.StdEncoding.EncodeToString(b))
}
