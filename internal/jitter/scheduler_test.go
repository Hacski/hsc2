package jitter

import (
	"testing"
	"time"
)

func TestIntervalWithinJitterBounds(t *testing.T) {
	s := New(BaseInterval(60*time.Second), JitterFraction(0.2))
	for i := 0; i < 500; i++ {
		d := s.Next()
		min := 60*time.Second - time.Duration(float64(60*time.Second)*0.2)
		max := 60*time.Second + time.Duration(float64(60*time.Second)*0.2)
		if d < min || d > max {
			t.Fatalf("interval %v out of bounds", d)
		}
	}
}

func TestIntervalsVary(t *testing.T) {
	s := New(BaseInterval(10*time.Second), JitterFraction(0.5))
	seen := map[time.Duration]bool{}
	for i := 0; i < 200; i++ {
		seen[s.Next()] = true
		if len(seen) > 1 {
			return
		}
	}
	t.Fatal("intervals did not vary")
}

func TestTimeSkewApplied(t *testing.T) {
	s := New(BaseInterval(10*time.Second), JitterFraction(0), TimeSkew(func(t time.Time) time.Duration {
		return 5 * time.Second
	}))
	d := s.Next()
	if d != 15*time.Second {
		t.Fatalf("expected 15s with skew, got %v", d)
	}
}

func TestBusinessHoursSkew(t *testing.T) {
	s := New(BaseInterval(60*time.Second), TimeSkew(BusinessHoursSkew(30*time.Second)))
	day := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)
	night := time.Date(2026, 1, 5, 2, 0, 0, 0, time.UTC)
	dDay := s.base + BusinessHoursSkew(30*time.Second)(day)
	dNight := s.base + BusinessHoursSkew(30*time.Second)(night)
	if dDay != 90*time.Second {
		t.Fatalf("day interval wrong: %v", dDay)
	}
	if dNight != 30*time.Second {
		t.Fatalf("night interval wrong: %v", dNight)
	}
}

func TestStopPreventsSleep(t *testing.T) {
	s := New(BaseInterval(time.Hour))
	go func() {
		time.Sleep(10 * time.Millisecond)
		s.Stop()
	}()
	start := time.Now()
	s.Sleep()
	if time.Since(start) > time.Second {
		t.Fatal("sleep should have been interrupted by stop")
	}
}
