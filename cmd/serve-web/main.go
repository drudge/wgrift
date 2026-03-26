package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	dir := "ui/web"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		dir = "."
	}

	// Copy wasm_exec.js from Go installation
	goRoot := os.Getenv("GOROOT")
	if goRoot == "" {
		out, err := exec.Command("go", "env", "GOROOT").Output()
		if err == nil {
			goRoot = string(out[:len(out)-1])
		}
	}
	if goRoot != "" {
		wasmExecDst := filepath.Join(dir, "wasm_exec.js")
		if _, err := os.Stat(wasmExecDst); os.IsNotExist(err) {
			// Try both known locations (changed in Go 1.25+)
			candidates := []string{
				filepath.Join(goRoot, "lib", "wasm", "wasm_exec.js"),
				filepath.Join(goRoot, "misc", "wasm", "wasm_exec.js"),
			}
			for _, src := range candidates {
				data, err := os.ReadFile(src)
				if err == nil {
					os.WriteFile(wasmExecDst, data, 0644)
					log.Printf("Copied wasm_exec.js from %s", src)
					break
				}
			}
		}
	}

	addr := ":8080"
	fmt.Printf("Serving %s at http://localhost%s\n", dir, addr)
	fs := http.FileServer(http.Dir(dir))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		fs.ServeHTTP(w, r)
	})
	log.Fatal(http.ListenAndServe(addr, handler))
}
