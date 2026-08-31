package pki

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

type CertType string

const (
	CertCA       CertType = "ca"
	CertServer   CertType = "server"
	CertOperator CertType = "operator"
	CertImplant  CertType = "implant"
)

type Material struct {
	Dir   string
	CAPem []byte
	CAKey []byte
	CAPin string
}

func GenerateCA(dir string, org string) (*Material, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: org + " Root CA", Organization: []string{org}},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	caPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(filepath.Join(dir, "ca.crt"), caPem, 0600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, "ca.key"), keyPem, 0600); err != nil {
		return nil, err
	}
	return &Material{Dir: dir, CAPem: caPem, CAKey: keyPem, CAPin: SPKIPinFromCert(der)}, nil
}

func LoadCA(dir string) (*Material, error) {
	caPem, err := os.ReadFile(filepath.Join(dir, "ca.crt"))
	if err != nil {
		return nil, err
	}
	keyPem, err := os.ReadFile(filepath.Join(dir, "ca.key"))
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(caPem)
	if block == nil {
		return nil, os.ErrInvalid
	}
	return &Material{Dir: dir, CAPem: caPem, CAKey: keyPem, CAPin: SPKIPinFromCert(block.Bytes)}, nil
}

func (m *Material) Issue(certType CertType, cn string, hosts []string, validDays int) (certPEM, keyPEM []byte, err error) {
	caBlock, _ := pem.Decode(m.CAPem)
	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	caKeyBlock, _ := pem.Decode(m.CAKey)
	caKey, err := parseKey(caKeyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: cn, Organization: []string{string(certType)}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(0, 0, validDays),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
		} else {
			tmpl.DNSNames = append(tmpl.DNSNames, h)
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		return nil, nil, err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM, nil
}

func parseKey(b []byte) (interface{}, error) {
	if k, err := x509.ParseECPrivateKey(b); err == nil {
		return k, nil
	}
	return x509.ParsePKCS8PrivateKey(b)
}

func SPKIPinFromCert(der []byte) string {
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return ""
	}
	spki, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(spki)
	return hex.EncodeToString(sum[:])
}

func VerifyPin(pinned string, presentedDER []byte) bool {
	return pinned != "" && pinned == SPKIPinFromCert(presentedDER)
}

func SPKIPinFromPEM(pemBytes []byte) string {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return ""
	}
	return SPKIPinFromCert(block.Bytes)
}
