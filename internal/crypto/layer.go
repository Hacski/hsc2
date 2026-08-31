package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
)

const (
	XORKeySize   = 32
	FrameVersion = 0x01
)

type XORStream struct {
	key []byte
}

func NewXORStream(key []byte) (*XORStream, error) {
	if len(key) != XORKeySize {
		return nil, errors.New("xor key must be 32 bytes")
	}
	cp := make([]byte, XORKeySize)
	copy(cp, key)
	return &XORStream{key: cp}, nil
}

func (x *XORStream) Encrypt(plaintext []byte) []byte {
	out := make([]byte, len(plaintext))
	for i, b := range plaintext {
		out[i] = b ^ x.key[i%XORKeySize]
	}
	return out
}

func (x *XORStream) Decrypt(ct []byte) []byte {
	return x.Encrypt(ct)
}

type AESGCMLayer struct {
	aead cipher.AEAD
}

func NewAESGCM(key []byte) (*AESGCMLayer, error) {
	h := sha256.Sum256(key)
	block, err := aes.NewCipher(h[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &AESGCMLayer{aead: aead}, nil
}

func (a *AESGCMLayer) Seal(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, a.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return a.aead.Seal(nonce, nonce, plaintext, nil), nil
}

func (a *AESGCMLayer) Open(ct []byte) ([]byte, error) {
	ns := a.aead.NonceSize()
	if len(ct) < ns {
		return nil, errors.New("ciphertext too short")
	}
	return a.aead.Open(nil, ct[:ns], ct[ns:], nil)
}

type Frame struct {
	Version  uint8
	Sequence uint32
	Body     []byte
}

func EncodeFrame(seq uint32, body []byte) []byte {
	buf := make([]byte, 1+4+4+len(body))
	buf[0] = FrameVersion
	binary.BigEndian.PutUint32(buf[1:5], seq)
	binary.BigEndian.PutUint32(buf[5:9], uint32(len(body)))
	copy(buf[9:], body)
	return buf
}

func DecodeFrame(data []byte) (*Frame, error) {
	if len(data) < 9 {
		return nil, errors.New("frame too short")
	}
	ver := data[0]
	if ver != FrameVersion {
		return nil, errors.New("unsupported frame version")
	}
	seq := binary.BigEndian.Uint32(data[1:5])
	bodyLen := binary.BigEndian.Uint32(data[5:9])
	if len(data) < 9+int(bodyLen) {
		return nil, errors.New("frame body truncated")
	}
	body := make([]byte, bodyLen)
	copy(body, data[9:9+bodyLen])
	return &Frame{Version: ver, Sequence: seq, Body: body}, nil
}

func HMACSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

func VerifyHMAC(key, data, expected []byte) bool {
	got := HMACSHA256(key, data)
	return hmac.Equal(got, expected)
}

func DeriveKey(masterKey []byte, context string) []byte {
	h := sha256.Sum256(append(masterKey, []byte(context)...))
	return h[:]
}
