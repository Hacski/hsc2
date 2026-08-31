package transport

import (
	"context"
	"fmt"
	"io"
	"sync"
)

type Handler interface {
	Handle(context.Context, []byte) ([]byte, error)
}

type Listener interface {
	Name() string
	Listen(ctx context.Context, addr string, handler Handler) error
	Close() error
}

type registry struct {
	mu        sync.RWMutex
	listeners map[string]Listener
}

var Registry = &registry{listeners: map[string]Listener{}}

func (r *registry) Register(l Listener) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listeners[l.Name()] = l
}

func (r *registry) Get(name string) (Listener, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	l, ok := r.listeners[name]
	return l, ok
}

func (r *registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []string{}
	for n := range r.listeners {
		out = append(out, n)
	}
	return out
}

type streamHandler struct {
	fn func(context.Context, []byte) ([]byte, error)
}

func (s streamHandler) Handle(ctx context.Context, b []byte) ([]byte, error) {
	return s.fn(ctx, b)
}

func StreamHandler(fn func(context.Context, []byte) ([]byte, error)) Handler {
	return streamHandler{fn: fn}
}

func NopWriter(sink io.Writer) *syncWriter {
	return &syncWriter{sink: sink}
}

type syncWriter struct {
	mu   sync.Mutex
	sink io.Writer
}

func (w *syncWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.sink.Write(p)
}

func Unsupported(name string) error {
	return fmt.Errorf("transport %q not registered", name)
}
