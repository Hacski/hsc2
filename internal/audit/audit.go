package audit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type Entry struct {
	Seq       int64     `json:"seq"`
	Timestamp time.Time `json:"timestamp"`
	Operator  string    `json:"operator"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Detail    string    `json:"detail"`
	PrevHash  string    `json:"prev_hash"`
	Hash      string    `json:"hash"`
}

type Logger struct {
	mu       sync.Mutex
	path     string
	key      []byte
	seq      int64
	prevHash string
	f        *os.File
}

func New(path string, key []byte) (*Logger, error) {
	l := &Logger{path: path, key: key}
	if err := l.load(); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *Logger) load() error {
	data, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			l.prevHash = genesis()
			return nil
		}
		return err
	}
	entries := []*Entry{}
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}
	if len(entries) > 0 {
		last := entries[len(entries)-1]
		l.seq = last.Seq
		l.prevHash = last.Hash
	}
	return nil
}

func genesis() string {
	h := sha256.Sum256([]byte("hsc2-audit-genesis"))
	return hex.EncodeToString(h[:])
}

func (l *Logger) Record(operator, action, target, detail string) (Entry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e := Entry{
		Seq:       l.seq + 1,
		Timestamp: time.Now().UTC(),
		Operator:  operator,
		Action:    action,
		Target:    target,
		Detail:    detail,
		PrevHash:  l.prevHash,
	}
	payload := fmt.Sprintf("%d|%d|%s|%s|%s|%s|%s", e.Seq, e.Timestamp.UnixNano(), e.Operator, e.Action, e.Target, e.Detail, e.PrevHash)
	mac := hmac.New(sha256.New, l.key)
	mac.Write([]byte(payload))
	e.Hash = hex.EncodeToString(mac.Sum(nil))

	if err := l.appendLine(e); err != nil {
		return Entry{}, err
	}
	l.seq = e.Seq
	l.prevHash = e.Hash
	return e, nil
}

func (l *Logger) appendLine(e Entry) error {
	data, err := os.ReadFile(l.path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	entries := []*Entry{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &entries); err != nil {
			return err
		}
	}
	cp := e
	entries = append(entries, &cp)
	out, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.path, out, 0600)
}

func (l *Logger) Verify() (bool, error) {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return false, err
	}
	entries := []*Entry{}
	if err := json.Unmarshal(data, &entries); err != nil {
		return false, err
	}
	prev := genesis()
	for _, e := range entries {
		payload := fmt.Sprintf("%d|%d|%s|%s|%s|%s|%s", e.Seq, e.Timestamp.UnixNano(), e.Operator, e.Action, e.Target, e.Detail, prev)
		mac := hmac.New(sha256.New, l.key)
		mac.Write([]byte(payload))
		want := hex.EncodeToString(mac.Sum(nil))
		if e.Hash != want {
			return false, nil
		}
		prev = e.Hash
	}
	return true, nil
}

func (l *Logger) Entries() ([]*Entry, error) {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return nil, err
	}
	entries := []*Entry{}
	if len(data) == 0 {
		return entries, nil
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}
