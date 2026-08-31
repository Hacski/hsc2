package session

import (
	"errors"
	"sync"
	"time"
)

type State string

const (
	StateActive   State = "active"
	StateIdle     State = "idle"
	StateDead     State = "dead"
)

type Session struct {
	ID          string
	OperatorID  string
	BeaconID    string
	Hostname    string
	OS          string
	Arch        string
	Username    string
	PID         int
	State       State
	OpenedAt    time.Time
	LastSeen    time.Time
	Tags        []string
	taskQueue   []string
	mu          sync.Mutex
}

func (s *Session) EnqueueTask(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.taskQueue = append(s.taskQueue, taskID)
}

func (s *Session) DequeueTask() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.taskQueue) == 0 {
		return "", false
	}
	id := s.taskQueue[0]
	s.taskQueue = s.taskQueue[1:]
	return id, true
}

func (s *Session) PendingTasks() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.taskQueue)
}

func (s *Session) Touch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastSeen = time.Now().UTC()
	s.State = StateActive
}

type Mux struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	byOp     map[string][]string
}

func NewMux() *Mux {
	return &Mux{
		sessions: map[string]*Session{},
		byOp:     map[string][]string{},
	}
}

func (m *Mux) Register(sess *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sess.OpenedAt = time.Now().UTC()
	sess.LastSeen = sess.OpenedAt
	sess.State = StateActive
	m.sessions[sess.ID] = sess
	m.byOp[sess.OperatorID] = append(m.byOp[sess.OperatorID], sess.ID)
}

func (m *Mux) Get(id string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, errors.New("session not found: " + id)
	}
	return s, nil
}

func (m *Mux) Kill(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return errors.New("session not found: " + id)
	}
	s.State = StateDead
	delete(m.sessions, id)
	ids := m.byOp[s.OperatorID]
	for i, sid := range ids {
		if sid == id {
			m.byOp[s.OperatorID] = append(ids[:i], ids[i+1:]...)
			break
		}
	}
	return nil
}

func (m *Mux) All() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, s)
	}
	return out
}

func (m *Mux) ForOperator(operatorID string) []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := m.byOp[operatorID]
	out := make([]*Session, 0, len(ids))
	for _, id := range ids {
		if s, ok := m.sessions[id]; ok {
			out = append(out, s)
		}
	}
	return out
}

func (m *Mux) Touch(id string) error {
	s, err := m.Get(id)
	if err != nil {
		return err
	}
	s.Touch()
	return nil
}

func (m *Mux) Reap(maxIdle time.Duration) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	var dead []string
	for id, s := range m.sessions {
		s.mu.Lock()
		if now.Sub(s.LastSeen) > maxIdle {
			s.State = StateDead
			dead = append(dead, id)
			delete(m.sessions, id)
		}
		s.mu.Unlock()
	}
	for _, id := range dead {
		for op, ids := range m.byOp {
			for i, sid := range ids {
				if sid == id {
					m.byOp[op] = append(ids[:i], ids[i+1:]...)
					break
				}
			}
		}
	}
	return dead
}
