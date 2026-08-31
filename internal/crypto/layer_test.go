package crypto

import (
	"bytes"
	"testing"
)

func TestXORStreamRoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{0xAB}, XORKeySize)
	x, err := NewXORStream(key)
	if err != nil {
		t.Fatal(err)
	}
	plaintext := []byte("beacon checkin payload data 0xdeadbeef")
	ct := x.Encrypt(plaintext)
	if bytes.Equal(ct, plaintext) {
		t.Fatal("encrypted output must differ from plaintext")
	}
	pt := x.Decrypt(ct)
	if !bytes.Equal(pt, plaintext) {
		t.Fatalf("XOR round trip failed: got %q", pt)
	}
}

func TestXORStreamBadKeySize(t *testing.T) {
	if _, err := NewXORStream([]byte("tooshort")); err == nil {
		t.Fatal("expected error for wrong key size")
	}
}

func TestAESGCMRoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 32)
	a, err := NewAESGCM(key)
	if err != nil {
		t.Fatal(err)
	}
	plaintext := []byte("task result from implant: whoami output")
	ct, err := a.Seal(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(ct, plaintext) {
		t.Fatal("ciphertext must differ from plaintext")
	}
	pt, err := a.Open(ct)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(pt, plaintext) {
		t.Fatalf("AESGCM round trip failed: got %q", pt)
	}
}

func TestAESGCMTamper(t *testing.T) {
	key := bytes.Repeat([]byte{0x99}, 32)
	a, _ := NewAESGCM(key)
	ct, _ := a.Seal([]byte("sensitive data"))
	ct[len(ct)-1] ^= 0xff
	if _, err := a.Open(ct); err == nil {
		t.Fatal("expected authentication failure after tampering")
	}
}

func TestFrameEncodeDecodeRoundTrip(t *testing.T) {
	body := []byte("frame payload")
	encoded := EncodeFrame(42, body)
	frame, err := DecodeFrame(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if frame.Version != FrameVersion {
		t.Fatalf("expected version %d, got %d", FrameVersion, frame.Version)
	}
	if frame.Sequence != 42 {
		t.Fatalf("expected seq 42, got %d", frame.Sequence)
	}
	if !bytes.Equal(frame.Body, body) {
		t.Fatalf("expected body %q, got %q", body, frame.Body)
	}
}

func TestFrameDecodeTruncated(t *testing.T) {
	if _, err := DecodeFrame([]byte{0x01, 0x00, 0x00}); err == nil {
		t.Fatal("expected error for truncated frame")
	}
}

func TestHMAC(t *testing.T) {
	key := []byte("hmac-key-32-bytes-padded-to-fit!!")
	data := []byte("authenticate this payload")
	mac := HMACSHA256(key, data)
	if !VerifyHMAC(key, data, mac) {
		t.Fatal("HMAC verification failed")
	}
	mac[0] ^= 0xFF
	if VerifyHMAC(key, data, mac) {
		t.Fatal("tampered MAC should not verify")
	}
}

func TestDeriveKey(t *testing.T) {
	master := bytes.Repeat([]byte{0x11}, 32)
	k1 := DeriveKey(master, "beacon-traffic")
	k2 := DeriveKey(master, "operator-auth")
	if bytes.Equal(k1, k2) {
		t.Fatal("derived keys for different contexts must differ")
	}
	k1b := DeriveKey(master, "beacon-traffic")
	if !bytes.Equal(k1, k1b) {
		t.Fatal("same context must produce same derived key")
	}
}
