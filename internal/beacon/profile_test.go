package beacon

import (
	"testing"
	"time"
)

func TestNextSleepWithinBounds(t *testing.T) {
	p := Profile{Interval: 60 * time.Second, Jitter: 0.2}
	for i := 0; i < 100; i++ {
		s := p.NextSleep()
		if s <= 0 {
			t.Fatalf("sleep must be positive, got %v", s)
		}
		min := 60*time.Second - time.Duration(float64(60*time.Second)*0.2)
		max := 60*time.Second + time.Duration(float64(60*time.Second)*0.2)
		if s < min || s > max {
			t.Fatalf("sleep %v out of bounds [%v,%v]", s, min, max)
		}
	}
}

func TestZeroJitterFixedInterval(t *testing.T) {
	p := Profile{Interval: 10 * time.Second, Jitter: 0}
	for i := 0; i < 10; i++ {
		if s := p.NextSleep(); s != 10*time.Second {
			t.Fatalf("expected fixed interval, got %v", s)
		}
	}
}

func TestValidate(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatal(err)
	}
	bad := Profile{Interval: 0, Jitter: 2, URI: []string{}}
	if err := bad.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
