//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/loom-go/loom"
	. "github.com/loom-go/loom/components"
	. "github.com/loom-go/web/components"
)

// Layout renders the app shell: sidebar nav on left, scrollable content on right.
// On mobile: top bar with hamburger, slide-out nav overlay.
func Layout(content loom.Node) loom.Node {
	return Div(
		Apply(Attr{"class": "flex h-screen overflow-hidden bg-surface-0"}),
		// Desktop sidebar (hidden on mobile)
		Div(
			Apply(Attr{"class": "hidden md:flex"}),
			NavRail(),
		),
		// Mobile top bar (hidden on desktop)
		MobileTopBar(),
		// Mobile nav overlay
		MobileNavOverlay(),
		// Main content area
		Div(
			Apply(Attr{"class": "flex-1 flex flex-col overflow-hidden"}),
			Div(
				Apply(Attr{"class": "flex-1 overflow-auto pt-14 md:pt-0"}),
				Div(
					Apply(Attr{"class": "px-5 py-6 md:px-10 md:py-8 lg:px-12 max-w-6xl"}),
					content,
				),
				Elem("footer",
					Apply(Attr{"class": "py-8 text-center text-[11px] text-ink-4/50"}),
					Text(fmt.Sprintf("© %d ", js.Global().Get("Date").New().Call("getFullYear").Int())),
					Elem("a",
						Apply(Attr{
							"href":   "https://www.penree.com",
							"target": "_blank",
							"rel":    "noopener noreferrer",
							"class":  "text-ink-4/70 hover:text-wg-500 transition-colors",
						}),
						Text("Nicholas Penree"),
					),
					Text(" · MIT License"),
				),
			),
		),
		ConfirmModal(),
		Toast(),
	)
}
