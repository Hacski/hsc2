package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

var ErrNotFound = errors.New("record not found")

type Cipher struct {
	aead cipher.AEAD
}

func NewCipher(installKey []byte) (*Cipher, error) {
	sum := sha256.Sum256(installKey)
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

func (c *Cipher) Encrypt(pt []byte) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return c.aead.Seal(nonce, nonce, pt, nil), nil
}

func (c *Cipher) Decrypt(ct []byte) ([]byte, error) {
	if len(ct) < c.aead.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	nonce := ct[:c.aead.NonceSize()]
	return c.aead.Open(nil, nonce, ct[c.aead.NonceSize():], nil)
}

type Store struct {
	mu   sync.RWMutex
	dir  string
	ciph *Cipher
}

func Open(dir string, installKey []byte) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	c, err := NewCipher(installKey)
	if err != nil {
		return nil, err
	}
	return &Store{dir: dir, ciph: c}, nil
}

func (s *Store) path(kind, id string) string {
	return filepath.Join(s.dir, kind, id+".enc")
}

func (s *Store) Put(kind, id string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	ct, err := s.ciph.Encrypt(data)
	if err != nil {
		return err
	}
	sub := filepath.Dir(s.path(kind, id))
	if err := os.MkdirAll(sub, 0700); err != nil {
		return err
	}
	tmp := s.path(kind, id) + ".tmp"
	if err := os.WriteFile(tmp, ct, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path(kind, id))
}

func (s *Store) Get(kind, id string, out interface{}) error {
	ct, err := os.ReadFile(s.path(kind, id))
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}
	pt, err := s.ciph.Decrypt(ct)
	if err != nil {
		return err
	}
	return json.Unmarshal(pt, out)
}

func (s *Store) Delete(kind, id string) error {
	err := os.Remove(s.path(kind, id))
	if os.IsNotExist(err) {
		return ErrNotFound
	}
	return err
}

func (s *Store) List(kind string) ([]string, error) {
	sub := filepath.Join(s.dir, kind)
	entries, err := os.ReadDir(sub)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	out := []string{}
	for _, e := range entries {
		name := e.Name()
		if len(name) > 4 && name[len(name)-4:] == ".enc" {
			out = append(out, name[:len(name)-4])
		}
	}
	return out, nil
}
