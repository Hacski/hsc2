package ftransfer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const DefaultChunkSize = 256 * 1024

var ErrChecksumMismatch = errors.New("sha256 checksum mismatch")
var ErrResumeOffset = errors.New("offset beyond file size")

type Chunk struct {
	TransferID string `json:"transfer_id"`
	Offset     int64  `json:"offset"`
	Size       int64  `json:"size"`
	Data       []byte `json:"data"`
	Total      int64  `json:"total"`
	SHA256     string `json:"sha256"`
}

type Meta struct {
	TransferID string `json:"transfer_id"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	SHA256     string `json:"sha256"`
	Direction  string `json:"direction"`
	SessionID  string `json:"session_id"`
	ChunkSize  int64  `json:"chunk_size"`
}

func SumBytes(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

type Sender struct {
	ChunkSize int64
}

func NewSender() *Sender {
	return &Sender{ChunkSize: DefaultChunkSize}
}

func (s *Sender) Meta(path, transferID, dir, session string) (Meta, error) {
	st, err := os.Stat(path)
	if err != nil {
		return Meta{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return Meta{}, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return Meta{}, err
	}
	return Meta{
		TransferID: transferID,
		Name:       filepath.Base(path),
		Size:       st.Size(),
		SHA256:     hex.EncodeToString(h.Sum(nil)),
		Direction:  dir,
		SessionID:  session,
		ChunkSize:  s.ChunkSize,
	}, nil
}

func (s *Sender) ChunkAt(path string, offset int64) (Chunk, error) {
	f, err := os.Open(path)
	if err != nil {
		return Chunk{}, err
	}
	defer f.Close()
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return Chunk{}, err
	}
	buf := make([]byte, s.ChunkSize)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return Chunk{}, err
	}
	data := buf[:n]
	return Chunk{
		Offset: offset,
		Size:   int64(n),
		Data:   data,
		SHA256: SumBytes(data),
	}, nil
}

type Receiver struct {
	ChunkSize int64
	mu        sync.Mutex
	dst       string
	written   int64
	expected  int64
	srcSum    string
	f         *os.File
}

func NewReceiver() *Receiver {
	return &Receiver{ChunkSize: DefaultChunkSize}
}

func (r *Receiver) Begin(m Meta, baseDir string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.srcSum = m.SHA256
	r.expected = m.Size
	r.dst = filepath.Join(baseDir, m.Name)
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return 0, err
	}
	f, err := os.OpenFile(r.dst, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return 0, err
	}
	r.f = f
	st, _ := f.Stat()
	r.written = st.Size()
	return r.written, nil
}

func (r *Receiver) Write(c Chunk) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return 0, errors.New("receiver not begun")
	}
	if SumBytes(c.Data) != c.SHA256 {
		return 0, ErrChecksumMismatch
	}
	if c.Offset > r.written {
		return 0, fmt.Errorf("%w: have %d need %d", ErrResumeOffset, r.written, c.Offset)
	}
	if c.Offset < r.written {
		r.f.Seek(c.Offset, io.SeekStart)
		r.written = c.Offset
	}
	if _, err := r.f.Seek(c.Offset, io.SeekStart); err != nil {
		return 0, err
	}
	if _, err := r.f.Write(c.Data); err != nil {
		return 0, err
	}
	r.written = c.Offset + c.Size
	return r.written, nil
}

func (r *Receiver) Finished() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return "", errors.New("receiver not begun")
	}
	r.f.Close()
	r.f = nil
	if r.written != r.expected {
		return "", fmt.Errorf("incomplete: have %d expected %d", r.written, r.expected)
	}
	sum, err := SumFile(r.dst)
	if err != nil {
		return "", err
	}
	if sum != r.srcSum {
		return "", ErrChecksumMismatch
	}
	return r.dst, nil
}

func SumFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func EncodeJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
