package exec

import "context"

type Result struct {
	Output   []byte `json:"output"`
	ExitCode int    `json:"exit_code"`
}

type Request struct {
	Payload []byte
	Entry   string
	Args    []string
}

type Backend interface {
	Name() string
	Run(ctx context.Context, r Request) (Result, error)
}
