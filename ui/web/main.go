//go:build js && wasm

package main

import (
	"log"
	"syscall/js"

	"github.com/loom-go/loom"
	"github.com/loom-go/web"
)

func jsLog(args ...any) {
	js.Global().Get("console").Call("log", args...)
}

func App() loom.Node {
	initState()
	initConfirmModal()
	initEmailModal()
	initToast()
	initMobileNav()
	initRouter()
	initUptimeTimers()

	// Fade out loading overlay after a tick so the DOM has rendered
	js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
		if el := js.Global().Get("document").Call("getElementById", "loading"); el.Truthy() {
			el.Get("classList").Call("add", "fade-out")
			js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
				el.Call("remove")
				return nil
			}), 200)
		}
		return nil
	}), 50)

	// Auth state was determined by checkSessionPreload before Loom started.
	// Return the correct view directly — no Bind needed.
	if preloadNeedsSetup {
		return SetupView()
	}
	if !preloadAuthed {
		return LoginView()
	}
	return Layout(Router())
}

func main() {
	// Check session before starting Loom. Runs async fetch in a goroutine,
	// blocks until complete. Go WASM scheduler yields to the browser event
	// loop so the fetch callback can fire.
	go func() {
		checkSessionPreload()

		app := web.NewApp()
		for err := range app.Run("#app", App) {
			log.Printf("Error: %v\n", err)
		}
	}()

	select {}
}
