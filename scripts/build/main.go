package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type target struct {
	os   string
	arch string
	ext  string
	suffix string
}

var matrix = []target{
	{"windows", "amd64", ".exe", "windows_amd64"},
	{"windows", "386", ".exe", "windows_386"},
	{"windows", "arm64", ".exe", "windows_arm64"},
	{"linux", "amd64", "", "linux_amd64"},
	{"linux", "386", "", "linux_386"},
	{"linux", "arm", "", "linux_arm"},
	{"linux", "arm64", "", "linux_arm64"},
	{"darwin", "amd64", "", "darwin_amd64"},
	{"darwin", "arm64", "", "darwin_arm64"},
	{"freebsd", "amd64", "", "freebsd_amd64"},
	{"openbsd", "amd64", "", "openbsd_amd64"},
	{"netbsd", "amd64", "", "netbsd_amd64"},
	{"android", "arm64", "", "android_arm64"},
}

var binaries = map[string]string{
	"server":  "./cmd/server",
	"implant": "./cmd/implant",
	"client":  "./cmd/client",
}

func main() {
	out := "out"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	any := false
	for name, pkg := range binaries {
		for _, t := range matrix {
			hostOS := runtime.GOOS
			hostArch := runtime.GOARCH
			cgo := "0"
			_ = hostOS
			_ = hostArch
			_ = cgo
			dest := filepath.Join(out, t.suffix)
			binName := name + t.suffix + t.ext
			if len(os.Args) > 2 && os.Args[2] != "all" {
				only := os.Args[2]
				if !strings.Contains(binName, only) {
					continue
				}
			}
			cmd := exec.Command("go", "build", "-trimpath", "-ldflags", "-s -w", "-o", filepath.Join(dest, binName), pkg)
			cmd.Env = append(os.Environ(),
				"GOOS="+t.os,
				"GOARCH="+t.arch,
				"CGO_ENABLED=0",
			)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "FAILED %s/%s %s: %v\n", t.os, t.arch, name, err)
				os.Exit(1)
			}
			fmt.Printf("built %s -> %s\n", binName, filepath.Join(dest, binName))
			any = true
		}
	}
	if !any {
		fmt.Fprintln(os.Stderr, "no binaries built")
		os.Exit(1)
	}
}
