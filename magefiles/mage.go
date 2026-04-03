//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var Default = Build

// Build compiles the wgrift binary (builds WASM first).
func Build() error {
	mg.Deps(Wasm)
	flags, err := ldflags()
	if err != nil {
		return err
	}
	return sh.RunV("go", "build", "-ldflags", flags, "-o", "bin/wgrift", "./cmd/wgrift")
}

// Test runs all tests.
func Test() error {
	return sh.RunV("go", "test", "./internal/...")
}

// Lint runs golangci-lint.
func Lint() error {
	return sh.RunV("golangci-lint", "run", "./...")
}

// Clean removes build artifacts.
func Clean() error {
	fmt.Println("Cleaning bin/ and dist/")
	if err := os.RemoveAll("bin"); err != nil {
		return err
	}
	return os.RemoveAll("dist")
}

// Wasm compiles the web UI to WebAssembly and copies wasm_exec.js.
func Wasm() error {
	flags, err := ldflags()
	if err != nil {
		return err
	}
	env := map[string]string{"GOOS": "js", "GOARCH": "wasm"}
	if err := sh.RunWithV(env, "go", "build", "-ldflags", flags, "-o", "ui/web/wgrift.wasm", "./ui/web"); err != nil {
		return err
	}
	return copyWasmExecJS()
}

// ServeWeb starts the WASM-only dev server on :8080 (no backend).
func ServeWeb() error {
	mg.Deps(Wasm)
	return sh.RunV("go", "run", "./cmd/serve-web")
}

// Serve builds and runs the full server in demo mode.
func Serve() error {
	mg.Deps(Wasm)
	// Build inline so we can run with env vars
	flags, err := ldflags()
	if err != nil {
		return err
	}
	if err := sh.RunV("go", "build", "-ldflags", flags, "-o", "bin/wgrift", "./cmd/wgrift"); err != nil {
		return err
	}

	masterKey := os.Getenv("WGRIFT_MASTER_KEY")
	if masterKey == "" {
		masterKey = "dev-master-key"
	}
	env := map[string]string{
		"WGRIFT_MASTER_KEY": masterKey,
		"WGRIFT_DEMO_MODE":  "true",
	}
	return sh.RunWithV(env, "./bin/wgrift", "serve")
}

// Dist creates a production distribution for linux/amd64.
func Dist() error {
	mg.Deps(Wasm)
	if err := os.MkdirAll("dist", 0755); err != nil {
		return err
	}

	flags, err := ldflags()
	if err != nil {
		return err
	}
	env := map[string]string{
		"GOOS":        "linux",
		"GOARCH":      "amd64",
		"CGO_ENABLED": "0",
	}
	if err := sh.RunWithV(env, "go", "build", "-ldflags", flags, "-o", "dist/wgrift", "./cmd/wgrift"); err != nil {
		return err
	}

	for _, f := range []string{"wgrift.service", "config.yaml", "install.sh"} {
		if err := sh.Copy(filepath.Join("dist", f), filepath.Join("deploy", f)); err != nil {
			return err
		}
	}

	fmt.Println("Distribution files ready in dist/")
	fmt.Println("  Binary has embedded web assets — single file deploy")
	return nil
}

// Dev starts Air for live-reload development (requires air to be installed).
func Dev() error {
	mg.Deps(Wasm)
	if _, err := exec.LookPath("air"); err != nil {
		return fmt.Errorf("air not found — install with: go install github.com/air-verse/air@latest")
	}

	masterKey := os.Getenv("WGRIFT_MASTER_KEY")
	if masterKey == "" {
		masterKey = "dev-master-key"
	}
	env := map[string]string{
		"WGRIFT_MASTER_KEY": masterKey,
		"WGRIFT_DEMO_MODE":  "true",
	}
	return sh.RunWithV(env, "air")
}

// ldflags returns the linker flags for version injection.
func ldflags() (string, error) {
	version, err := sh.Output("git", "describe", "--tags", "--always", "--dirty")
	if err != nil {
		version = "dev"
	}
	commit, err := sh.Output("git", "rev-parse", "--short", "HEAD")
	if err != nil {
		commit = "unknown"
	}
	date := time.Now().UTC().Format(time.RFC3339)

	pkg := "github.com/drudge/wgrift/pkg/version"
	flags := strings.Join([]string{
		"-s", "-w",
		fmt.Sprintf("-X %s.Version=%s", pkg, version),
		fmt.Sprintf("-X %s.Commit=%s", pkg, commit),
		fmt.Sprintf("-X %s.Date=%s", pkg, date),
	}, " ")
	return flags, nil
}

// copyWasmExecJS copies wasm_exec.js from the Go installation to ui/web/.
func copyWasmExecJS() error {
	dst := filepath.Join("ui", "web", "wasm_exec.js")

	goRoot, err := sh.Output("go", "env", "GOROOT")
	if err != nil {
		return fmt.Errorf("cannot determine GOROOT: %w", err)
	}

	candidates := []string{
		filepath.Join(goRoot, "lib", "wasm", "wasm_exec.js"),
		filepath.Join(goRoot, "misc", "wasm", "wasm_exec.js"),
	}
	for _, src := range candidates {
		if _, err := os.Stat(src); err == nil {
			fmt.Printf("Copying wasm_exec.js from %s\n", src)
			return sh.Copy(dst, src)
		}
	}
	return fmt.Errorf("wasm_exec.js not found in GOROOT (%s)", goRoot)
}
