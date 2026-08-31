package jitter

import (
	"math/rand"
	"sync"
	"time"
)

type Scheduler struct {
	mu      sync.Mutex
	base    time.Duration
	jitter  float64
	rng     *rand.Rand
	skew    func(time.Time) time.Duration
	last    time.Time
	stopped bool
	stopCh  chan struct{}
}

type Option func(*Scheduler)

func BaseInterval(d time.Duration) Option {
	return func(s *Scheduler) { s.base = d }
}

func JitterFraction(f float64) Option {
	return func(s *Scheduler) { s.jitter = f }
}

func TimeSkew(fn func(time.Time) time.Duration) Option {
	return func(s *Scheduler) { s.skew = fn }
}

func New(opts ...Option) *Scheduler {
	s := &Scheduler{
		base:   time.Minute,
		jitter: 0.1,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
		stopCh: make(chan struct{}),
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *Scheduler) Next() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.base
	if s.jitter > 0 {
		delta := time.Duration(float64(d) * s.jitter * s.rng.Float64())
		if s.rng.Intn(2) == 0 {
			d -= delta
		} else {
			d += delta
		}
	}
	if d <= 0 {
		d = time.Second
	}
	if s.skew != nil {
		d += s.skew(time.Now())
		if d <= 0 {
			d = time.Second
		}
	}
	return d
}

func (s *Scheduler) Sleep() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	d := s.intervalLocked()
	s.last = time.Now()
	stopCh := s.stopCh
	s.mu.Unlock()
	timer := time.NewTimer(d)
	select {
	case <-timer.C:
		return
	case <-stopCh:
		if !timer.Stop() {
			<-timer.C
		}
		return
	}
}

func (s *Scheduler) intervalLocked() time.Duration {
	d := s.base
	if s.jitter > 0 {
		delta := time.Duration(float64(d) * s.jitter * s.rng.Float64())
		if s.rng.Intn(2) == 0 {
			d -= delta
		} else {
			d += delta
		}
	}
	if d <= 0 {
		d = time.Second
	}
	if s.skew != nil {
		d += s.skew(time.Now())
		if d <= 0 {
			d = time.Second
		}
	}
	return d
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	close(s.stopCh)
	s.mu.Unlock()
}

func (s *Scheduler) IsStopped() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopped
}

func (s *Scheduler) LastCheckIn() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.last
}

func BusinessHoursSkew(active time.Duration) func(time.Time) time.Duration {
	return func(t time.Time) time.Duration {
		h := t.Hour()
		if h >= 9 && h < 17 {
			return active
		}
		return -active
	}
}
