package exec

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
)

const (
	coffMagicAMD64    = 0x8664
	coffMagicI386     = 0x14c
	coffMagicARM64    = 0xaa64
	coffMagicARM      = 0x1c0
	coffSectText      = 0x20
	coffSectData      = 0x40
	coffSectBSS       = 0x80
	coffSymAuxFormat1 = 0x105
	coffSymData       = 0x103
)

var ErrNotCOFF = errors.New("not a valid COFF object or BOF")
var ErrCOFFTooLarge = errors.New("COFF exceeds size limit")

type Section struct {
	Name        string
	VirtualSize uint32
	VirtualAddr uint32
	SizeOfRaw   uint32
	PointerRaw  uint32
	Char        uint32
}

type Symbol struct {
	Name    string
	Value   uint32
	Section int16
	Type    uint16
	Class   byte
}

type COFF struct {
	Machine  uint16
	Sections []Section
	Symbols  []Symbol
	Funcs    []string
	DataOnly bool
	raw      []byte
}

func Parse(data []byte) (*COFF, error) {
	if len(data) < 20 {
		return nil, ErrNotCOFF
	}
	if len(data) > 64*1024*1024 {
		return nil, ErrCOFFTooLarge
	}
	machine := binary.LittleEndian.Uint16(data[0:2])
	switch machine {
	case coffMagicAMD64, coffMagicI386, coffMagicARM64, coffMagicARM:
	default:
		return nil, fmt.Errorf("%w: unsupported machine type 0x%x", ErrNotCOFF, machine)
	}
	numSect := binary.LittleEndian.Uint16(data[2:4])
	ptrSym := binary.LittleEndian.Uint32(data[8:12])
	numSym := binary.LittleEndian.Uint32(data[12:16])

	off := 20
	c := &COFF{Machine: machine}
	if fits(off, numSect, 40, len(data)) {
		for i := uint16(0); i < numSect; i++ {
			base := off + int(i)*40
			if base+40 > len(data) {
				return nil, ErrNotCOFF
			}
			c.Sections = append(c.Sections, parseSectionHeader(data[base:base+40]))
		}
	}
	off += int(numSect) * 40

	strTabOff := int(ptrSym) + int(numSym)*18
	strTabSize := uint32(0)
	if strTabOff+4 <= len(data) {
		strTabSize = binary.LittleEndian.Uint32(data[strTabOff : strTabOff+4])
	} else {
		return nil, ErrNotCOFF
	}

	for i := uint32(0); i < numSym; i++ {
		base := int(ptrSym) + int(i)*18
		if base+18 > len(data) {
			return nil, ErrNotCOFF
		}
		sym := parseSymbol(data[base : base+18])
		if sym.Section == -1 || sym.Section == 0 {
			continue
		}
		if sym.Name != "" {
			c.Symbols = append(c.Symbols, sym)
			// section-relative function definitions
			if isFunctionType(sym.Type, sym.Class) {
				c.Funcs = append(c.Funcs, sym.Name)
			}
		}
		// skip aux records
		if base+18 <= len(data) {
			aux := binary.LittleEndian.Uint16(data[base+16 : base+18])
			i += uint32(aux)
		}
	}
	_ = strTabSize
	sort.Strings(c.Funcs)
	if len(c.Funcs) == 0 && len(c.Symbols) == 0 {
		c.DataOnly = true
	}
	return c, nil
}

func parseSectionHeader(b []byte) Section {
	return Section{
		Name:        cstr(b[0:8]),
		VirtualSize: binary.LittleEndian.Uint32(b[8:12]),
		VirtualAddr: binary.LittleEndian.Uint32(b[12:16]),
		SizeOfRaw:   binary.LittleEndian.Uint32(b[16:20]),
		PointerRaw:  binary.LittleEndian.Uint32(b[20:24]),
		Char:        binary.LittleEndian.Uint32(b[36:40]),
	}
}

func parseSymbol(b []byte) Symbol {
	var name string
	if binary.LittleEndian.Uint32(b[0:4]) == 0 {
		name = fmt.Sprintf(".unnamed")
	} else {
		name = cstr(b[0:8])
	}
	return Symbol{
		Name:    name,
		Value:   binary.LittleEndian.Uint32(b[8:12]),
		Section: int16(binary.LittleEndian.Uint16(b[12:14])),
		Type:    binary.LittleEndian.Uint16(b[14:16]),
		Class:   b[16],
	}
}

func isFunctionType(t uint16, class byte) bool {
	return (t&0x30) == 0x20 || class == coffinClassFunc
}

const coffinClassFunc = 101

func cstr(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func fits(off int, n uint16, size int, length int) bool {
	return off <= length && int(n)*size <= length-off
}

func (c *COFF) SectionBytes(data []byte, s Section) []byte {
	if s.PointerRaw+uint32(s.SizeOfRaw) <= uint32(len(data)) {
		return data[s.PointerRaw : s.PointerRaw+s.SizeOfRaw]
	}
	return nil
}

func (c *COFF) HasFunction(name string) bool {
	for _, f := range c.Funcs {
		if f == name {
			return true
		}
	}
	return false
}
