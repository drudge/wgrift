package ui

import "embed"

// WebAssets embeds the built UI files (index.html, wgrift.wasm, wasm_exec.js).
// Build the WASM binary and copy wasm_exec.js into ui/web/ before compiling.
//
//go:embed web/index.html web/wgrift.wasm web/wasm_exec.js
var WebAssets embed.FS
