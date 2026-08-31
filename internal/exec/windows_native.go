//go:build windows

package exec

import (
	"context"
	"fmt"
	"syscall"
	"unsafe"
)

const (
	memCommit  = 0x1000
	pageExecRC = 0x20
	pageRW     = 0x04
	pageRX     = 0x20
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procVirtualAlloc   = kernel32.NewProc("VirtualAlloc")
	procVirtualProtect = kernel32.NewProc("VirtualProtect")
	procVirtualFree    = kernel32.NewProc("VirtualFree")
)

type NativeWindowsBackend struct{}

func (NativeWindowsBackend) Name() string { return "native_windows" }

func (NativeWindowsBackend) Run(ctx context.Context, r Request) (Result, error) {
	coff, err := Parse(r.Payload)
	if err != nil {
		return Result{}, err
	}
	total := len(coff.Sections) * 4096
	base, _, _ := procVirtualAlloc.Call(0, uintptr(total), memCommit, pageRW)
	if base == 0 {
		return Result{}, fmt.Errorf("VirtualAlloc failed")
	}
	defer freeMem(base, uintptr(total))
	for _, s := range coff.Sections {
		src := coff.SectionBytes(r.Payload, s)
		if src == nil {
			continue
		}
		dst := base + uintptr(s.VirtualAddr)
		copyMem(dst, src)
		prot := uintptr(pageExecRC)
		if s.Char&coffSectData != 0 || s.Char&coffSectBSS != 0 {
			prot = pageRW
		}
		var old uint32
		procVirtualProtect.Call(dst, uintptr(len(src)), prot, uintptr(unsafe.Pointer(&old)))
	}
	entry := r.Entry
	if entry == "" && len(coff.Funcs) > 0 {
		entry = coff.Funcs[0]
	}
	if entry == "" {
		return Result{}, fmt.Errorf("no entry point")
	}
	rva := findSymbolRVA(codeToUint64(base), coff, entry)
	_ = rva
	return Result{Output: []byte(fmt.Sprintf("mapped COFF in memory entry=%s base=0x%x", entry, base))}, nil
}

func freeMem(base, size uintptr) {
	procVirtualFree.Call(base, size, 0x8000)
}

func copyMem(dst uintptr, src []byte) {
	for i, b := range src {
		*(*byte)(unsafe.Pointer(dst + uintptr(i))) = b
	}
}

func codeToUint64(v uintptr) uint64 { return uint64(v) }

func findSymbolRVA(base uint64, c *COFF, name string) uint64 {
	_ = base
	for _, s := range c.Symbols {
		if s.Name == name {
			return uint64(s.Value)
		}
	}
	return 0
}

func init() {
	Register(NativeWindowsBackend{})
}
