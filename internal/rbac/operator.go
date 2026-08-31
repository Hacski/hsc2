package rbac

import (
	"sync"
	"time"
)

type Peer struct {
	Username  string
	Role      Role
	Connected time.Time
	LastSeen  time.Time
	Addr      string
}

type SessionOwnership struct {
	mu        sync.RWMutex
	holders   map[string]string
	peers     map[string]*Peer
	transfers chan ownershipTransfer
}

type ownershipTransfer struct {
	from  string
	to    string
	sess  string
	reply chan bool
}

func NewSessionOwnership() *SessionOwnership {
	so := &SessionOwnership{
		holders:   map[string]string{},
		peers:     map[string]*Peer{},
		transfers: make(chan ownershipTransfer, 64),
	}
	go so.monitor()
	return so
}

func (so *SessionOwnership) monitor() {
	for t := range so.transfers {
		ok := so.claimLocked(t.sess, t.from, t.to)
		select {
		case t.reply <- ok:
		default:
		}
	}
}

func (so *SessionOwnership) claimLocked(sess, from, to string) bool {
	cur, ok := so.holders[sess]
	if !ok || cur == from {
		so.holders[sess] = to
		return true
	}
	return false
}

func (so *SessionOwnership) Claim(sess, from, to string) bool {
	reply := make(chan bool, 1)
	so.transfers <- ownershipTransfer{from: from, to: to, sess: sess, reply: reply}
	return <-reply
}

func (so *SessionOwnership) Release(sess string) {
	so.mu.Lock()
	defer so.mu.Unlock()
	delete(so.holders, sess)
}

func (so *SessionOwnership) Holder(sess string) (string, bool) {
	so.mu.RLock()
	defer so.mu.RUnlock()
	h, ok := so.holders[sess]
	return h, ok
}

func (so *SessionOwnership) AddPeer(p *Peer) {
	so.mu.Lock()
	defer so.mu.Unlock()
	if existing, ok := so.peers[p.Username]; ok {
		existing.LastSeen = time.Now()
		existing.Role = p.Role
		return
	}
	so.peers[p.Username] = p
}

func (so *SessionOwnership) RemovePeer(username string) {
	so.mu.Lock()
	defer so.mu.Unlock()
	delete(so.peers, username)
}

func (so *SessionOwnership) Peers() []*Peer {
	so.mu.RLock()
	defer so.mu.RUnlock()
	out := []*Peer{}
	for _, p := range so.peers {
		out = append(out, p)
	}
	return out
}
