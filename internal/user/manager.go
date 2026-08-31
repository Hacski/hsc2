package user

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleViewer   Role = "viewer"
)

type Account struct {
	ID           string
	Username     string
	Role         Role
	PasswordHash string
	Salt         string
	Engagements  map[string]bool
	CreatedAt    time.Time
	LastLoginAt  time.Time
	Disabled     bool
}

type Manager struct {
	mu       sync.RWMutex
	accounts map[string]*Account
}

func NewManager() *Manager {
	return &Manager{accounts: map[string]*Account{}}
}

func (m *Manager) Create(username string, password string, role Role) (*Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.accounts[username]; exists {
		return nil, fmt.Errorf("username %s already exists", username)
	}
	salt, hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	acc := &Account{
		ID:           newID(),
		Username:     username,
		Role:         role,
		PasswordHash: hash,
		Salt:         salt,
		Engagements:  map[string]bool{},
		CreatedAt:    time.Now().UTC(),
	}
	m.accounts[username] = acc
	return acc, nil
}

func (m *Manager) Authenticate(username, password string) (*Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	acc, ok := m.accounts[username]
	if !ok {
		return nil, errors.New("invalid credentials")
	}
	if acc.Disabled {
		return nil, errors.New("account disabled")
	}
	if !verifyPassword(password, acc.Salt, acc.PasswordHash) {
		return nil, errors.New("invalid credentials")
	}
	acc.LastLoginAt = time.Now().UTC()
	return acc, nil
}

func (m *Manager) SetRole(username string, role Role) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	acc, ok := m.accounts[username]
	if !ok {
		return fmt.Errorf("user %s not found", username)
	}
	acc.Role = role
	return nil
}

func (m *Manager) Disable(username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	acc, ok := m.accounts[username]
	if !ok {
		return fmt.Errorf("user %s not found", username)
	}
	acc.Disabled = true
	return nil
}

func (m *Manager) GrantEngagement(username, engagementID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	acc, ok := m.accounts[username]
	if !ok {
		return fmt.Errorf("user %s not found", username)
	}
	acc.Engagements[engagementID] = true
	return nil
}

func (m *Manager) RevokeEngagement(username, engagementID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	acc, ok := m.accounts[username]
	if !ok {
		return fmt.Errorf("user %s not found", username)
	}
	delete(acc.Engagements, engagementID)
	return nil
}

func (m *Manager) HasEngagement(username, engagementID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	acc, ok := m.accounts[username]
	if !ok {
		return false
	}
	return acc.Engagements[engagementID]
}

func (m *Manager) List() []*Account {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Account, 0, len(m.accounts))
	for _, acc := range m.accounts {
		out = append(out, acc)
	}
	return out
}

func (m *Manager) ChangePassword(username, oldPassword, newPassword string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	acc, ok := m.accounts[username]
	if !ok {
		return fmt.Errorf("user %s not found", username)
	}
	if !verifyPassword(oldPassword, acc.Salt, acc.PasswordHash) {
		return errors.New("invalid credentials")
	}
	salt, hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	acc.Salt = salt
	acc.PasswordHash = hash
	return nil
}

func hashPassword(password string) (salt, hash string, err error) {
	saltBytes := make([]byte, 16)
	if _, err = rand.Read(saltBytes); err != nil {
		return
	}
	salt = hex.EncodeToString(saltBytes)
	h := sha256.Sum256(append(saltBytes, []byte(password)...))
	hash = hex.EncodeToString(h[:])
	return
}

func verifyPassword(password, salt, storedHash string) bool {
	saltBytes, err := hex.DecodeString(salt)
	if err != nil {
		return false
	}
	h := sha256.Sum256(append(saltBytes, []byte(password)...))
	computed := hex.EncodeToString(h[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(storedHash)) == 1
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
