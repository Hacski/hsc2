package pivot

import (
	"context"
	"net"
	"sync"
)

type routeKey struct {
	prefix  string
	session string
}

type Forward struct {
	table    *Table
	mu       sync.Mutex
	forwards map[string]context.CancelFunc
}

func NewForward(table *Table) *Forward {
	return &Forward{
		table:    table,
		forwards: map[string]context.CancelFunc{},
	}
}

func (f *Forward) Start(ctx context.Context, localAddr, remoteAddr string) error {
	ln, err := net.Listen("tcp", localAddr)
	if err != nil {
		return err
	}
	cctx, cancel := context.WithCancel(ctx)
	f.mu.Lock()
	f.forwards[localAddr] = cancel
	f.mu.Unlock()
	go f.serve(cctx, ln, remoteAddr)
	return nil
}

func (f *Forward) Stop(localAddr string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if cancel, ok := f.forwards[localAddr]; ok {
		cancel()
		delete(f.forwards, localAddr)
	}
}

func (f *Forward) serve(ctx context.Context, ln net.Listener, remoteAddr string) {
	defer ln.Close()
	go func() {
		<-ctx.Done()
		ln.Close()
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go f.handle(ctx, conn, remoteAddr)
	}
}

func (f *Forward) handle(ctx context.Context, local net.Conn, remoteAddr string) {
	defer local.Close()
	dialer, _, ok := f.table.Resolve(remoteAddr)
	if !ok {
		return
	}
	remote, err := dialer.Dial(ctx, remoteAddr)
	if err != nil {
		return
	}
	defer remote.Close()
	relay(local, remote)
}

func (f *Forward) Active() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.forwards))
	for addr := range f.forwards {
		out = append(out, addr)
	}
	return out
}
