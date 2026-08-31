package module

import (
	"context"
	"fmt"
)

type fakeMod struct {
	name    string
	calls   int
	loaded  bool
	version string
}

func (f *fakeMod) Name() string    { return f.name }
func (f *fakeMod) Version() string { return f.version }
func (f *fakeMod) OnLoad(ctx context.Context) error {
	f.loaded = true
	f.calls++
	return nil
}
func (f *fakeMod) OnUnload(ctx context.Context) error {
	f.loaded = false
	return nil
}
func (f *fakeMod) Execute(ctx context.Context, c Context) ([]byte, error) {
	f.calls++
	return []byte(fmt.Sprintf("ran %s by %s", f.name, c.Operator)), nil
}
