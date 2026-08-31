package zerotrust

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	TokenBytes        = 32
	DefaultTokenTTL   = 15 * time.Minute
	DefaultCertTTL    = 24 * time.Hour
)

type Scope string

const (
	ScopeSession  Scope = "session"
	ScopeOperator Scope = "operator"
	ScopeAdmin    Scope = "admin"
)

type Token struct {
	ID        string
	OwnerID   string
	Scope     Scope
	IssuedAt  time.Time
	ExpiresAt time.Time
	Used      bool
}

func (t *Token) Valid() error {
	if t == nil {
		return errors.New("token is nil")
	}
	if time.Now().After(t.ExpiresAt) {
		return fmt.Errorf("token %s expired at %s", t.ID, t.ExpiresAt.Format(time.RFC3339))
	}
	return nil
}

type TokenStore struct {
	mu     sync.RWMutex
	tokens map[string]*Token
}

func NewTokenStore() *TokenStore {
	return &TokenStore{tokens: map[string]*Token{}}
}

func (s *TokenStore) Issue(ownerID string, scope Scope, ttl time.Duration) (*Token, error) {
	raw := make([]byte, TokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	h := sha256.Sum256(raw)
	id := hex.EncodeToString(h[:])
	now := time.Now().UTC()
	t := &Token{
		ID:        id,
		OwnerID:   ownerID,
		Scope:     scope,
		IssuedAt:  now,
		ExpiresAt: now.Add(ttl),
	}
	s.mu.Lock()
	s.tokens[id] = t
	s.mu.Unlock()
	return t, nil
}

func (s *TokenStore) Validate(id string, requiredScope Scope) (*Token, error) {
	s.mu.RLock()
	t, ok := s.tokens[id]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown token %s", id)
	}
	if err := t.Valid(); err != nil {
		s.Revoke(id)
		return nil, err
	}
	if t.Scope != requiredScope && t.Scope != ScopeAdmin {
		return nil, fmt.Errorf("token scope %s does not satisfy required %s", t.Scope, requiredScope)
	}
	return t, nil
}

func (s *TokenStore) Revoke(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, id)
}

func (s *TokenStore) RevokeAll(ownerID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for id, t := range s.tokens {
		if t.OwnerID == ownerID {
			delete(s.tokens, id)
			count++
		}
	}
	return count
}

func (s *TokenStore) Sweep() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	count := 0
	for id, t := range s.tokens {
		if now.After(t.ExpiresAt) {
			delete(s.tokens, id)
			count++
		}
	}
	return count
}

func (s *TokenStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.tokens)
}

type CertRecord struct {
	CommonName  string
	Fingerprint string
	IssuedAt    time.Time
	ExpiresAt   time.Time
	Revoked     bool
}

type CertLedger struct {
	mu    sync.RWMutex
	certs map[string]*CertRecord
}

func NewCertLedger() *CertLedger {
	return &CertLedger{certs: map[string]*CertRecord{}}
}

func (l *CertLedger) Register(cn, fingerprint string, ttl time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now().UTC()
	l.certs[fingerprint] = &CertRecord{
		CommonName:  cn,
		Fingerprint: fingerprint,
		IssuedAt:    now,
		ExpiresAt:   now.Add(ttl),
	}
}

func (l *CertLedger) Verify(fingerprint string) error {
	l.mu.RLock()
	defer l.mu.RUnlock()
	r, ok := l.certs[fingerprint]
	if !ok {
		return fmt.Errorf("unknown certificate %s", fingerprint)
	}
	if r.Revoked {
		return fmt.Errorf("certificate %s has been revoked", fingerprint)
	}
	if time.Now().After(r.ExpiresAt) {
		return fmt.Errorf("certificate %s expired at %s", fingerprint, r.ExpiresAt.Format(time.RFC3339))
	}
	return nil
}

func (l *CertLedger) Revoke(fingerprint string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	r, ok := l.certs[fingerprint]
	if !ok {
		return fmt.Errorf("unknown certificate %s", fingerprint)
	}
	r.Revoked = true
	return nil
}
