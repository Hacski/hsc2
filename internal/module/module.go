package module

import "context"

type Context struct {
	SessionID string
	Operator  string
	Args      []string
	Env       map[string]string
}

type Module interface {
	Name() string
	Version() string
	OnLoad(ctx context.Context) error
	OnUnload(ctx context.Context) error
	Execute(ctx context.Context, c Context) ([]byte, error)
}

type Info struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Loaded     bool   `json:"loaded"`
	AutoLoad   bool   `json:"auto_load"`
	Permission string `json:"permission"`
}
