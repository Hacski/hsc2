package obfuscate

import (
	"bytes"
	"testing"
)

func TestXORRoundTrip(t *testing.T) {
	key, err := RandomKey(16)
	if err != nil {
		t.Fatal(err)
	}
	plaintext := []byte("hello obfuscation world")
	ct := XOREncrypt(plaintext, key)
	if bytes.Equal(ct, plaintext) {
		t.Fatal("encrypted output should differ from plaintext")
	}
	pt := XORDecrypt(ct, key)
	if !bytes.Equal(pt, plaintext) {
		t.Fatalf("round trip failed: got %q", pt)
	}
}

func TestStringEncryptDecrypt(t *testing.T) {
	key, _ := RandomKey(8)
	original := "C:\\Windows\\System32\\whoami"
	ct := StringEncrypt(original, key)
	got := StringDecrypt(ct, key)
	if got != original {
		t.Fatalf("expected %q, got %q", original, got)
	}
}

func TestGarbleStrings(t *testing.T) {
	src := []byte("func main() { loadLibrary(\"kernel32.dll\") }")
	replacements := map[string]string{
		"loadLibrary": "xK3qLoad",
		"kernel32":    "k3rn3132",
	}
	out := GarbleStrings(src, replacements)
	if bytes.Contains(out, []byte("loadLibrary")) {
		t.Fatal("garble should have replaced loadLibrary")
	}
	if !bytes.Contains(out, []byte("xK3qLoad")) {
		t.Fatal("garble should contain replacement xK3qLoad")
	}
}

func TestPad(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	padded := Pad(data, 8)
	if len(padded)%8 != 0 {
		t.Fatalf("padded length %d is not multiple of 8", len(padded))
	}
	if !bytes.HasPrefix(padded, data) {
		t.Fatal("padded data should start with original data")
	}
}

func TestDecoyHeader(t *testing.T) {
	data := []byte("payload")
	headerSize := 16
	wrapped, err := AddDecoyHeader(data, headerSize)
	if err != nil {
		t.Fatal(err)
	}
	if len(wrapped) != len(data)+headerSize {
		t.Fatalf("expected len %d, got %d", len(data)+headerSize, len(wrapped))
	}
	stripped := StripDecoyHeader(wrapped, headerSize)
	if !bytes.Equal(stripped, data) {
		t.Fatalf("expected %q, got %q", data, stripped)
	}
}

func TestIPv4Encoding(t *testing.T) {
	shellcode := []byte{0x90, 0x90, 0xCC, 0xCC}
	encoded := EncodeIPv4Shellcode(shellcode)
	if len(encoded) == 0 {
		t.Fatal("expected non-empty IPv4 encoded output")
	}
}

func TestRollNonce(t *testing.T) {
	n1, err := RollNonce()
	if err != nil {
		t.Fatal(err)
	}
	n2, _ := RollNonce()
	if n1 == n2 {
		t.Fatal("two nonces should differ (birthday bound, not guaranteed but astronomically unlikely)")
	}
}
