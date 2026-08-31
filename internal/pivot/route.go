package pivot

import (
	"context"
	"net"
	"strings"
	"sync"
)

type Dialer interface {
	Name() string
	Dial(ctx context.Context, target string) (net.Conn, error)
}

type Table struct {
	mu       sync.RWMutex
	longest  map[string]*route
	sessions map[string][]*route
}

type route struct {
	prefix   string
	session  string
	dialer   Dialer
	hopCount int
}

func NewTable() *Table {
	return &Table{
		longest:  map[string]*route{},
		sessions: map[string][]*route{},
	}
}

func (t *Table) Add(prefix, session string, d Dialer, hopCount int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	r := &route{prefix: prefix, session: session, dialer: d, hopCount: hopCount}
	// longest-prefix: only keep if this prefix is longer or equal
	existing, ok := t.longest[prefix]
	if ok && existing.hopCount <= hopCount {
		return
	}
	t.longest[prefix] = r
	t.sessions[session] = append(t.sessions[session], r)
}

func (t *Table) RemoveSession(session string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, r := range t.sessions[session] {
		delete(t.longest, r.prefix)
	}
	delete(t.sessions, session)
}

func (t *Table) Resolve(target string) (Dialer, string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	var bestR *route
	bestSpec := -1
	for prefix, r := range t.longest {
		if matchPrefix(prefix, target) {
			spec := specificity(prefix)
			if spec > bestSpec {
				bestSpec = spec
				bestR = r
			}
		}
	}
	if bestR == nil {
		return nil, "", false
	}
	return bestR.dialer, bestR.session, true
}

func specificity(prefix string) int {
	if prefix == "" {
		return 0
	}
	if strings.Contains(prefix, "/") {
		_, cidr, err := net.ParseCIDR(prefix)
		if err != nil {
			return 0
		}
		ones, _ := cidr.Mask.Size()
		return ones
	}
	return 256
}

func matchPrefix(prefix, target string) bool {
	if prefix == "" {
		return true
	}
	host, _, err := net.SplitHostPort(target)
	if err != nil {
		host = target
	}
	if strings.Contains(prefix, "/") {
		_, cidr, err := net.ParseCIDR(prefix)
		if err != nil {
			return false
		}
		ip := net.ParseIP(host)
		if ip == nil {
			return false
		}
		return cidr.Contains(ip)
	}
	if prefix == host || prefix == target {
		return true
	}
	if len(prefix) < len(target) && prefix == target[:len(prefix)] {
		return true
	}
	return false
}

func (t *Table) Sessions() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := []string{}
	for s := range t.sessions {
		out = append(out, s)
	}
	return out
}
