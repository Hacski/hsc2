package pki

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
)

type ServerTLS struct {
	Config   *tls.Config
	CertFile string
	KeyFile  string
}

func BuildServerTLS(caPEM, serverCertPEM, serverKeyPEM []byte) *ServerTLS {
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caPEM)
	cert, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		return nil
	}
	return &ServerTLS{
		Config: &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientCAs:    pool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS13,
		},
	}
}

func BuildServerTLSFromFiles(caFile, certFile, keyFile string) (*ServerTLS, error) {
	ca, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	cert, err := os.ReadFile(certFile)
	if err != nil {
		return nil, err
	}
	key, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	st := BuildServerTLS(ca, cert, key)
	if st == nil {
		return nil, os.ErrInvalid
	}
	st.CertFile = certFile
	st.KeyFile = keyFile
	return st, nil
}

func BuildClientTLS(caPEM, clientCertPEM, clientKeyPEM []byte, serverPin string) (*tls.Config, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, os.ErrInvalid
	}
	cert, err := tls.X509KeyPair(clientCertPEM, clientKeyPEM)
	if err != nil {
		return nil, err
	}
	cfg := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            pool,
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true,
	}
	cfg.VerifyPeerCertificate = makePinnedVerifier(serverPin, pool)
	return cfg, nil
}

func makePinnedVerifier(pin string, pool *x509.CertPool) func([][]byte, [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return os.ErrInvalid
		}
		if pin != "" && !VerifyPin(pin, rawCerts[0]) {
			return os.ErrPermission
		}
		cert, err := x509.ParseCertificate(rawCerts[0])
		if err != nil {
			return err
		}
		_, err = cert.Verify(x509.VerifyOptions{Roots: pool, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageAny}})
		return err
	}
}

func WriteMaterial(path string, pemBytes []byte) error {
	return os.WriteFile(path, pemBytes, 0600)
}

func DecodeCert(pemBytes []byte) *x509.Certificate {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil
	}
	c, _ := x509.ParseCertificate(block.Bytes)
	return c
}
