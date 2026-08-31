package exec

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sort"
	"sync"
)

var (
	mu       sync.RWMutex
	backends = map[string]Backend{}
	order    []string
)

func Register(b Backend) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := backends[b.Name()]; !ok {
		order = append(order, b.Name())
	}
	backends[b.Name()] = b
}

func Get(name string) (Backend, bool) {
	mu.RLock()
	defer mu.RUnlock()
	b, ok := backends[name]
	return b, ok
}

func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := append([]string{}, order...)
	sort.Strings(out)
	return out
}

func Default() Backend {
	mu.RLock()
	defer mu.RUnlock()
	if platform, ok := backends["native_"+runtime.GOOS]; ok {
		return platform
	}
	if validate, ok := backends["validate"]; ok {
		return validate
	}
	return nil
}

type ValidateBackend struct{}

func (ValidateBackend) Name() string { return "validate" }

func (ValidateBackend) Run(ctx context.Context, r Request) (Result, error) {
	coff, err := Parse(r.Payload)
	if err != nil {
		return Result{}, err
	}
	entry := r.Entry
	if entry == "" && len(coff.Funcs) > 0 {
		entry = coff.Funcs[0]
	}
	out := fmt.Sprintf("parsed COFF machine=0x%x sections=%d funcs=%v entry=%s",
		coff.Machine, len(coff.Sections), coff.Funcs, entry)
	return Result{Output: []byte(out)}, nil
}

func init() {
	Register(ValidateBackend{})
}

func NoDiskGuard(path string, payload []byte) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("refusing to write payload to disk: %s exists", path)
	}
	if payload != nil {
		return nil
	}
	return nil
}
