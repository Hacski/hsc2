package version

import (
	"testing"
)

func TestWireRoundTrip(t *testing.T) {
	w := Wire
	maj, min, pat := FromWire(w)
	if maj != Major || min != Minor || pat != Patch {
		t.Fatalf("expected %d.%d.%d, got %d.%d.%d", Major, Minor, Patch, maj, min, pat)
	}
}

func TestWireString(t *testing.T) {
	w := (uint32(2) << 16) | (uint32(3) << 8) | uint32(7)
	got := WireString(w)
	if got != "2.3.7" {
		t.Fatalf("expected 2.3.7, got %s", got)
	}
}

func TestCompatSameMajor(t *testing.T) {
	server := (uint32(1) << 16) | (uint32(5) << 8)
	beacon := (uint32(1) << 16) | (uint32(3) << 8)
	res := CheckCompat(server, beacon)
	if !res.OK {
		t.Fatalf("expected compatible: %s", res.Reason)
	}
}

func TestCompatMajorMismatch(t *testing.T) {
	server := (uint32(1) << 16) | (uint32(0) << 8)
	beacon := (uint32(2) << 16) | (uint32(0) << 8)
	res := CheckCompat(server, beacon)
	if res.OK {
		t.Fatal("major version mismatch should be incompatible")
	}
}

func TestCompatBeaconNewerMinor(t *testing.T) {
	server := (uint32(1) << 16) | (uint32(2) << 8)
	beacon := (uint32(1) << 16) | (uint32(5) << 8)
	res := CheckCompat(server, beacon)
	if res.OK {
		t.Fatal("beacon with newer minor than server should be incompatible")
	}
}
