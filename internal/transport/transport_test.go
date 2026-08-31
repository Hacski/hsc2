package transport

import (
	"bytes"
	"encoding/base32"
	"strings"
	"testing"
)

func TestRegistryRoundTrip(t *testing.T) {
	Registry.Register(NewHTTPListener("http"))
	l, ok := Registry.Get("http")
	if !ok {
		t.Fatal("http listener not registered")
	}
	if l.Name() != "http" {
		t.Fatalf("wrong name %q", l.Name())
	}
}

func TestDNSEncodeDecode(t *testing.T) {
	payload := []byte("dns-beacon-checkin-001")
	qname := encodeQNameForTest(payload)
	decoded := decodeQNamePayload(qname)
	if !bytes.Equal(decoded, payload) {
		t.Fatalf("roundtrip mismatch: %q != %q", decoded, payload)
	}

	header := make([]byte, 12)
	copy(header, []byte{0x12, 0x34})
	resp := encodeTXTResponse(header, qname, payload)
	if len(resp) < 24 {
		t.Fatalf("response too short: %d", len(resp))
	}
	if !bytes.HasPrefix(resp, header[:2]) {
		t.Fatalf("response header mismatch")
	}
}

func encodeQNameForTest(payload []byte) []byte {
	enc := strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(payload))
	var out []byte
	for len(enc) > 0 {
		n := maxDNSLabel - 1
		if len(enc) < n {
			n = len(enc)
		}
		out = append(out, byte(n))
		out = append(out, enc[:n]...)
		enc = enc[n:]
	}
	out = append(out, byte(0))
	return out
}
