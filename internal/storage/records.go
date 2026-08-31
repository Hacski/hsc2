package storage

import "time"

const (
	KindLoot    = "loot"
	KindCreds   = "creds"
	KindSessKey = "sesskeys"
)

type Loot struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Source    string    `json:"source"`
	Data      []byte    `json:"data"`
	SHA256    string    `json:"sha256"`
	Captured  time.Time `json:"captured"`
}

type Credential struct {
	ID       string    `json:"id"`
	Target   string    `json:"target"`
	Username string    `json:"username"`
	Secret   string    `json:"secret"`
	Kind     string    `json:"kind"`
	Captured time.Time `json:"captured"`
}

type SessionKey struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Key       []byte    `json:"key"`
	RotatedAt time.Time `json:"rotated_at"`
}
