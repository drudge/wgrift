//go:build js && wasm

package main

import (
	"log"

	"github.com/loom-go/loom"
	"github.com/loom-go/web"
	. "github.com/loom-go/web/components"
)

func App() loom.Node {
	return Div(
		H1(Text("wgRift")),
		P(Text("WireGuard VPN Management Platform")),
		P(Text("Loom Web renderer is working.")),
	)
}

func main() {
	app := web.NewApp()
	for err := range app.Run("#app", App) {
		log.Printf("Error: %v\n", err)
	}
}
