package obfuscate

import (
	"crypto/rand"
	"encoding/binary"
	"math/big"
	"strings"
)

func XOREncrypt(data []byte, key []byte) []byte {
	if len(key) == 0 {
		return data
	}
	out := make([]byte, len(data))
	for i, b := range data {
		out[i] = b ^ key[i%len(key)]
	}
	return out
}

func XORDecrypt(data []byte, key []byte) []byte {
	return XOREncrypt(data, key)
}

func RandomKey(n int) ([]byte, error) {
	key := make([]byte, n)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

func StringEncrypt(s string, key []byte) []byte {
	return XOREncrypt([]byte(s), key)
}

func StringDecrypt(ct []byte, key []byte) string {
	return string(XORDecrypt(ct, key))
}

func GarbleStrings(src []byte, replacements map[string]string) []byte {
	result := string(src)
	for from, to := range replacements {
		result = strings.ReplaceAll(result, from, to)
	}
	return []byte(result)
}

func Pad(data []byte, blockSize int) []byte {
	if blockSize <= 0 {
		return data
	}
	padLen := blockSize - (len(data) % blockSize)
	if padLen == blockSize {
		padLen = 0
	}
	junk := make([]byte, padLen)
	rand.Read(junk)
	return append(data, junk...)
}

func AddDecoyHeader(data []byte, headerSize int) ([]byte, error) {
	hdr := make([]byte, headerSize)
	if _, err := rand.Read(hdr); err != nil {
		return nil, err
	}
	return append(hdr, data...), nil
}

func StripDecoyHeader(data []byte, headerSize int) []byte {
	if len(data) < headerSize {
		return data
	}
	return data[headerSize:]
}

func EncodeIPv4Shellcode(shellcode []byte) []byte {
	padded := shellcode
	for len(padded)%4 != 0 {
		padded = append(padded, 0x90)
	}
	buf := make([]byte, 0, len(padded)/4*16)
	for i := 0; i < len(padded); i += 4 {
		b := padded[i : i+4]
		seg := []byte{b[0], b[1], b[2], b[3]}
		buf = append(buf, []byte(ipv4String(seg)+",")...)
	}
	if len(buf) > 0 && buf[len(buf)-1] == ',' {
		buf = buf[:len(buf)-1]
	}
	return buf
}

func ipv4String(b []byte) string {
	out := make([]byte, 0, 15)
	for i, octet := range b {
		out = appendUint(out, octet)
		if i < 3 {
			out = append(out, '.')
		}
	}
	return string(out)
}

func appendUint(b []byte, v byte) []byte {
	if v >= 100 {
		b = append(b, byte('0'+v/100))
		v %= 100
		b = append(b, byte('0'+v/10))
		v %= 10
	} else if v >= 10 {
		b = append(b, byte('0'+v/10))
		v %= 10
	}
	return append(b, byte('0'+v))
}

func RollNonce() (uint64, error) {
	n, err := rand.Int(rand.Reader, new(big.Int).SetUint64(^uint64(0)))
	if err != nil {
		return 0, err
	}
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], n.Uint64())
	return binary.BigEndian.Uint64(buf[:]), nil
}
