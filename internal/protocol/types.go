package protocol

import "time"

type MsgType uint8

const (
	MsgHello MsgType = iota
	MsgTask
	MsgResult
	MsgBeacon
	MsgFileChunk
	MsgFileAck
	MsgPivot
	MsgKill
	MsgModule
)

type Envelope struct {
	Type      MsgType     `json:"type"`
	Version   uint32      `json:"version"`
	SessionID string      `json:"session_id"`
	Nonce     uint64      `json:"nonce"`
	Timestamp int64       `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

type Task struct {
	ID        string            `json:"id"`
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Payload   []byte            `json:"payload"`
	Requested time.Time         `json:"requested"`
	InMemory  bool              `json:"in_memory"`
	Env       map[string]string `json:"env"`
}

type Result struct {
	ID        string    `json:"id"`
	Output    []byte    `json:"output"`
	Err       string    `json:"err"`
	Completed time.Time `json:"completed"`
	ExitCode  int       `json:"exit_code"`
}

type Beacon struct {
	SessionID string   `json:"session_id"`
	Hostname  string   `json:"hostname"`
	OS        string   `json:"os"`
	Arch      string   `json:"arch"`
	PID       int      `json:"pid"`
	Username  string   `json:"username"`
	Modules   []string `json:"modules"`
	Jitter    float64  `json:"jitter"`
	Interval  int      `json:"interval"`
}
