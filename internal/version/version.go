package version

import (
	"fmt"
)

const (
	Major = 1
	Minor = 0
	Patch = 0

	Wire uint32 = (Major << 16) | (Minor << 8) | Patch
)

func String() string {
	return fmt.Sprintf("%d.%d.%d", Major, Minor, Patch)
}

func FromWire(w uint32) (major, minor, patch uint8) {
	major = uint8(w >> 16)
	minor = uint8((w >> 8) & 0xff)
	patch = uint8(w & 0xff)
	return
}

func WireString(w uint32) string {
	maj, min, pat := FromWire(w)
	return fmt.Sprintf("%d.%d.%d", maj, min, pat)
}

type CompatResult struct {
	OK             bool
	ServerVersion  string
	BeaconVersion  string
	Reason         string
}

func CheckCompat(serverWire, beaconWire uint32) CompatResult {
	sMaj, sMin, _ := FromWire(serverWire)
	bMaj, bMin, _ := FromWire(beaconWire)

	res := CompatResult{
		ServerVersion: WireString(serverWire),
		BeaconVersion: WireString(beaconWire),
	}

	if bMaj != sMaj {
		res.OK = false
		res.Reason = fmt.Sprintf("major version mismatch: server=%d beacon=%d", sMaj, bMaj)
		return res
	}

	if bMin > sMin {
		res.OK = false
		res.Reason = fmt.Sprintf("beacon minor %d is newer than server minor %d; upgrade server", bMin, sMin)
		return res
	}

	res.OK = true
	res.Reason = "compatible"
	return res
}
