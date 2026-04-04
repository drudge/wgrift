package ui

import "embed"

// WebAssets embeds the built UI files (index.html, wgrift.wasm + compressed variants, wasm_exec.js).
// Build the WASM binary and copy wasm_exec.js into ui/web/ before compiling.
//
//go:embed web/index.html web/wgrift.wasm web/wgrift.wasm.gz web/wgrift.wasm.br web/wasm_exec.js
var WebAssets embed.FS
