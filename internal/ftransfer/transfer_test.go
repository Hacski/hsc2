package ftransfer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChunkedResumableIntegrity(t *testing.T) {
	src := filepath.Join(t.TempDir(), "payload.bin")
	data := make([]byte, 1_500_000)
	for i := range data {
		data[i] = byte(i % 251)
	}
	if err := os.WriteFile(src, data, 0600); err != nil {
		t.Fatal(err)
	}

	sender := NewSender()
	meta, err := sender.Meta(src, "xfer-1", "upload", "sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(data)) != meta.Size {
		t.Fatalf("size mismatch")
	}

	dstDir := filepath.Join(t.TempDir(), "incoming")
	receiver := NewReceiver()
	resume, err := receiver.Begin(meta, dstDir)
	if err != nil {
		t.Fatal(err)
	}
	if resume != 0 {
		t.Fatalf("expected fresh start at 0, got %d", resume)
	}

	var offset int64
	chunks := [][]byte{}
	for offset < meta.Size {
		c, err := sender.ChunkAt(src, offset)
		if err != nil {
			t.Fatal(err)
		}
		c.TransferID = meta.TransferID
		c.Total = meta.Size
		chunks = append(chunks, c.Data)
		offset += c.Size
		_, err = receiver.Write(c)
		if err != nil {
			t.Fatal(err)
		}
	}

	got, err := receiver.Finished()
	if err != nil {
		t.Fatal(err)
	}
	gotSum, _ := SumFile(got)
	if gotSum != meta.SHA256 {
		t.Fatalf("integrity checksum mismatch %s != %s", gotSum, meta.SHA256)
	}

	gotData, _ := os.ReadFile(got)
	if string(gotData) != string(data) {
		t.Fatal("content mismatch")
	}
}

func TestChecksumMismatchRejected(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.bin")
	if err := os.WriteFile(src, []byte("hello world"), 0600); err != nil {
		t.Fatal(err)
	}
	meta, _ := NewSender().Meta(src, "xfer-2", "upload", "sess-1")

	r := NewReceiver()
	if _, err := r.Begin(meta, filepath.Join(dir, "in")); err != nil {
		t.Fatal(err)
	}
	bad := Chunk{Offset: 0, Size: 5, Data: []byte("hello"), SHA256: "deadbeef"}
	if _, err := r.Write(bad); err != ErrChecksumMismatch {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
}

func TestResumeIntoExistingPartial(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "b.bin")
	data := []byte("0123456789abcdef")
	os.WriteFile(src, data, 0600)
	meta, _ := NewSender().Meta(src, "xfer-3", "upload", "sess-1")

	inDir := filepath.Join(dir, "in")
	// first pass writes only the first 4 bytes into the destination file
	r1 := NewReceiver()
	r1.Begin(meta, inDir)
	r1.Write(Chunk{Offset: 0, Size: 4, Data: data[:4], SHA256: SumBytes(data[:4])})

	// second receiver resumes into the same directory/file, expecting offset 4
	r2 := NewReceiver()
	resume, err := r2.Begin(meta, inDir)
	if err != nil {
		t.Fatal(err)
	}
	if resume != 4 {
		t.Fatalf("expected resume offset 4, got %d", resume)
	}
	r2.Write(Chunk{Offset: 4, Size: 4, Data: data[4:8], SHA256: SumBytes(data[4:8])})
	r2.Write(Chunk{Offset: 8, Size: 8, Data: data[8:], SHA256: SumBytes(data[8:])})

	got, err := r2.Finished()
	if err != nil {
		t.Fatal(err)
	}
	gotData, _ := os.ReadFile(got)
	if string(gotData) != string(data) {
		t.Fatalf("resumed content mismatch: %q", gotData)
	}
}
