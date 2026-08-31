package exec

import (
	"bytes"
	"encoding/binary"
	"testing"
)

type sectionSpec struct {
	name        [8]byte
	virtualSz   uint32
	virtualAddr uint32
	sizeRaw     uint32
	ptrRaw      uint32
	char        uint32
	data        []byte
}

type symSpec struct {
	name  string
	value uint32
	sect  int16
	typ   uint16
	class byte
	aux   uint16
}

func buildCOFF(machine uint16, sections []sectionSpec, symbols []symSpec) []byte {
	numSect := len(sections)
	numSym := len(symbols)
	headerSize := 20
	sectOff := headerSize
	symOff := sectOff + numSect*40
	strOff := symOff + numSym*18

	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, machine)
	_ = binary.Write(&buf, binary.LittleEndian, uint16(numSect))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(0)) // timestamp
	_ = binary.Write(&buf, binary.LittleEndian, uint32(symOff))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(numSym))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(0)) // size of optional header
	_ = binary.Write(&buf, binary.LittleEndian, uint16(0)) // characteristics

	// string table: 4-byte size at strOff
	strSize := 4
	strTable := []byte{}
	strOffsets := map[string]uint32{}

	bodyOff := strOff + strSize
	// place section raw data after string table
	for i := range sections {
		sections[i].ptrRaw = uint32(bodyOff)
		bodyOff += len(sections[i].data)
	}

	for i := range sections {
		s := sections[i]
		buf.Write(s.name[:])
		le32 := func(v uint32) {
			var b [4]byte
			binary.LittleEndian.PutUint32(b[:], v)
			buf.Write(b[:])
		}
		le32(s.virtualSz)
		le32(s.virtualAddr)
		le32(s.sizeRaw)
		le32(s.ptrRaw)
		le32(0) // relocation ptr
		le32(0) // line ptr
		var nb [2]byte
		binary.LittleEndian.PutUint16(nb[:], 0)
		buf.Write(nb[:]) // num reloc
		buf.Write(nb[:]) // num line
		le32(s.char)     // characteristics
	}

	// symbols
	longNames := map[string]bool{}
	for _, s := range symbols {
		if len(s.name) > 8 {
			longNames[s.name] = true
		}
	}
	for _, s := range symbols {
		nameField := make([]byte, 8)
		if longNames[s.name] {
			binary.LittleEndian.PutUint32(nameField[0:4], 0)
			off := uint32(len(strTable))
			binary.LittleEndian.PutUint32(nameField[4:8], off+4)
			strTable = append(strTable, []byte(s.name)...)
			strTable = append(strTable, 0)
			strOffsets[s.name] = off
		} else {
			copy(nameField, s.name)
		}
		buf.Write(nameField)
		_ = binary.Write(&buf, binary.LittleEndian, s.value)
		_ = binary.Write(&buf, binary.LittleEndian, uint16(int16(s.sect)))
		_ = binary.Write(&buf, binary.LittleEndian, s.typ)
		buf.Write([]byte{s.class, 0})
		_ = binary.Write(&buf, binary.LittleEndian, s.aux)
	}

	// string table with size prefix (accounting for long names)
	_ = binary.Write(&buf, binary.LittleEndian, uint32(4+len(strTable)))
	buf.Write(strTable)

	// raw section data
	for _, s := range sections {
		buf.Write(s.data)
	}
	return buf.Bytes()
}

func TestParseValidCOFF(t *testing.T) {
	coffData := buildCOFF(coffMagicAMD64, []sectionSpec{
		{name: str8(".text"), virtualSz: 16, virtualAddr: 0, sizeRaw: 16, char: coffSectText | 0x60000020, data: make([]byte, 16)},
		{name: str8(".data"), virtualSz: 8, virtualAddr: 4096, sizeRaw: 8, char: coffSectData | 0x60000040, data: make([]byte, 8)},
	}, []symSpec{
		{name: "go", value: 0, sect: 1, typ: 0x20, class: coffinClassFunc},
	})

	c, err := Parse(coffData)
	if err != nil {
		t.Fatal(err)
	}
	if c.Machine != coffMagicAMD64 {
		t.Fatalf("machine mismatch 0x%x", c.Machine)
	}
	if len(c.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(c.Sections))
	}
	found := false
	for _, f := range c.Funcs {
		if f == "go" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected go entry func, got %v", c.Funcs)
	}
	if !c.HasFunction("go") {
		t.Fatal("HasFunction go should be true")
	}
}

func TestRejectNonCOFF(t *testing.T) {
	if _, err := Parse([]byte("this is not a coff object")); err == nil {
		t.Fatal("expected error for non-COFF input")
	}
}

func TestBackendRoundTrip(t *testing.T) {
	coffData := buildCOFF(coffMagicAMD64, []sectionSpec{
		{name: str8(".text"), virtualSz: 8, virtualAddr: 0, sizeRaw: 8, char: coffSectText | 0x60, data: []byte{0xC3}},
	}, []symSpec{
		{name: "main", value: 0, sect: 1, typ: 0x20, class: coffinClassFunc},
	})
	b := Default()
	if b == nil {
		t.Fatal("no default backend")
	}
	if b.Name() != "validate" && b.Name() != "native" {
		t.Skip("platform backend in use")
	}
	res, err := b.Run(t.Context(), Request{Payload: coffData, Entry: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Output) == 0 {
		t.Fatal("expected output")
	}
}

func str8(s string) [8]byte {
	var out [8]byte
	copy(out[:], s)
	return out
}
