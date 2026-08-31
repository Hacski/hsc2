package storage

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
)

const keyFileName = "install.key"
const keyFilePerm = 0600

func NewInstallKey() ([]byte, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func LoadOrCreateInstallKey(dir string) ([]byte, error) {
	path := filepath.Join(dir, keyFileName)
	data, err := os.ReadFile(path)
	if err == nil {
		key := make([]byte, hex.DecodedLen(len(data)))
		n, derr := hex.Decode(key, data)
		if derr != nil {
			return nil, derr
		}
		return key[:n], nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	key, err := NewInstallKey()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	enc := make([]byte, hex.EncodedLen(len(key)))
	hex.Encode(enc, key)
	if err := os.WriteFile(path, enc, keyFilePerm); err != nil {
		return nil, err
	}
	return key, nil
}

func KeyFingerprint(key []byte) string {
	sum := sha256.Sum256(key)
	return hex.EncodeToString(sum[:4])
}

func ValidateKey(key []byte, fingerprint string) error {
	if KeyFingerprint(key) != fingerprint {
		return errors.New("install key mismatch")
	}
	return nil
}
