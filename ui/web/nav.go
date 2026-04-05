//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"

	"github.com/loom-go/loom"
	. "github.com/loom-go/loom/components"
	. "github.com/loom-go/web/components"
)

// Package-level state for mobile nav toggle
var (
	mobileNavOpen    func() bool
	setMobileNavOpen func(bool)
)

func initMobileNav() {
	mobileNavOpen, setMobileNavOpen = Signal(false)
}

// MobileTopBar renders a compact top bar visible only on mobile.
func MobileTopBar() loom.Node {
	return Div(
		Apply(Attr{"class": "md:hidden fixed top-0 left-0 right-0 z-40 bg-surface-0/95 backdrop-blur-md border-b border-line-1 flex items-center px-4 h-14"}),
		// Hamburger — left
		Button(
			Apply(Attr{"class": "w-9 h-9 rounded-md flex items-center justify-center text-ink-2 hover:bg-surface-2 transition-colors flex-shrink-0"}),
			Apply(On{"click": func() { setMobileNavOpen(!mobileNavOpen()) }}),
			Icon("menu", 20),
		),
		// Brand — centered, tappable to go home
		Div(
			Apply(Attr{"class": "flex-1 flex justify-center"}),
			Button(
				Apply(Attr{"class": "text-base font-black tracking-tight"}),
				Apply(On{"click": func() { navigate("/") }}),
				Span(Apply(Attr{"class": "text-wg-500"}), Text("wg")),
				Span(Apply(Attr{"class": "text-ink-1"}), Text("Rift")),
			),
		),
		// Spacer to balance hamburger width
		Div(Apply(Attr{"class": "w-9 flex-shrink-0"})),
	)
}

// MobileNavOverlay renders a slide-out nav drawer on mobile.
func MobileNavOverlay() loom.Node {
	return Bind(func() loom.Node {
		open := mobileNavOpen()
		route := currentRoute()

		backdropClass := "md:hidden fixed inset-0 z-40 bg-black/60 backdrop-blur-sm"
		drawerClass := "fixed top-0 left-0 bottom-0 z-50 w-64 bg-surface-1 border-r border-line-1 transform transition-transform"
		if !open {
			backdropClass = "hidden"
			drawerClass = "hidden"
		}
		return Div(
			Div(
				Apply(Attr{"class": backdropClass}),
				Apply(On{"click": func() { setMobileNavOpen(false) }}),
			),
			Elem("nav",
				Apply(Attr{"class": drawerClass}),
				mobileNavContent(route),
			),
		)
	})
}

func mobileNavContent(route string) loom.Node {
	return Div(
		Apply(Attr{"class": "flex flex-col h-full"}),
		// Header
		Div(
			Apply(Attr{"class": "px-5 pt-5 pb-4"}),
			Div(
				Apply(Attr{"class": "flex items-center justify-between"}),
				Div(
					Apply(Attr{"class": "text-lg font-black tracking-tight"}),
					Span(Apply(Attr{"class": "text-wg-500"}), Text("wg")),
					Span(Apply(Attr{"class": "text-ink-1"}), Text("Rift")),
				),
				Button(
					Apply(Attr{"class": "w-8 h-8 rounded-md flex items-center justify-center text-ink-3 hover:bg-surface-3"}),
					Apply(On{"click": func() { setMobileNavOpen(false) }}),
					Icon("x", 18),
				),
			),
		),
		Div(Apply(Attr{"class": "h-px bg-line-1 mx-4"})),
		// Nav items
		Div(
			Apply(Attr{"class": "flex-1 px-2 py-3"}),
			Div(
				Apply(Attr{"class": "space-y-0.5"}),
				mobileNavItem("/", "Status", "activity", route),
				mobileNavItem("/interfaces", "Interfaces", "chevrons-left-right-ellipsis", route),
				mobileNavItem("/logs", "Logs", "scroll-text", route),
			),
			Div(Apply(Attr{"class": "h-px bg-line-1 mx-3 my-4"})),
			Div(
				Div(Apply(Attr{"class": "px-3 mb-2 text-[10px] font-semibold text-ink-3 uppercase tracking-[0.15em]"}), Text("Admin")),
				mobileNavItem("/users", "Users", "users", route),
				mobileNavItem("/settings", "Settings", "settings", route),
			),
		),
		// User section
		Show(func() bool { return currentUser() != nil }, func() loom.Node {
			u := currentUser()
			initial := string([]rune(u.Username)[0:1])
			return Div(
				Apply(Attr{"class": "border-t border-line-1 px-4 py-4"}),
				Div(
					Apply(Attr{"class": "flex items-center justify-between"}),
					Div(
						Apply(Attr{"class": "flex items-center gap-3 min-w-0"}),
						Div(
							Apply(Attr{"class": "w-8 h-8 rounded-md bg-wg-600/15 flex items-center justify-center text-xs font-bold text-wg-400 uppercase flex-shrink-0"}),
							Text(initial),
						),
						Div(
							Apply(Attr{"class": "min-w-0"}),
							Div(Apply(Attr{"class": "text-sm font-medium text-ink-1 truncate"}), Text(u.Username)),
							Div(Apply(Attr{"class": "text-[10px] text-ink-3 uppercase tracking-wider"}), Text(u.Role)),
						),
					),
					Button(
						Apply(Attr{"class": "text-ink-4 hover:text-ink-2 transition-colors p-1.5 rounded-md hover:bg-surface-3 flex-shrink-0", "title": "Sign out"}),
						Apply(On{"click": func() {
							go func() {
								apiFetch("POST", "/api/v1/auth/logout", nil, nil)
								js.Global().Get("window").Get("location").Call("reload")
							}()
						}}),
						Icon("log-out", 15),
					),
				),
			)
		}),
	)
}

func mobileNavItem(href, label, iconName, route string) loom.Node {
	active := route == href || (href != "/" && strings.HasPrefix(route, href))

	class := "w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm font-medium transition-all duration-100 "
	if active {
		class += "bg-wg-600/10 border border-wg-600/20 text-wg-400"
	} else {
		class += "border border-transparent text-ink-2 hover:text-ink-1 hover:bg-surface-2/50"
	}

	return Button(
		Apply(Attr{"class": class}),
		Apply(On{"click": func() {
			navigate(href)
			setMobileNavOpen(false)
		}}),
		Icon(iconName, 17),
		Span(Text(label)),
	)
}

// NavRail renders the sidebar navigation.
func NavRail() loom.Node {
	return Elem("nav",
		Apply(Attr{"class": "w-[220px] bg-surface-0 border-r border-line-1 flex flex-col flex-shrink-0 overflow-y-auto"}),

		// Brand
		Div(
			Apply(Attr{"class": "px-5 pt-7 pb-6"}),
			Div(
				Apply(Attr{"class": "text-xl font-black tracking-tight"}),
				Span(Apply(Attr{"class": "text-wg-500"}), Text("wg")),
				Span(Apply(Attr{"class": "text-ink-1"}), Text("Rift")),
			),
			Div(Apply(Attr{"class": "text-[10px] text-ink-3 mt-0.5 uppercase tracking-[0.15em]"}), Text("VPN Management")),
		),

		// Main nav
		Div(
			Apply(Attr{"class": "flex-1 px-2"}),
			Div(
				Apply(Attr{"class": "space-y-0.5"}),
				navItem("/", "Status", "activity"),
				navItem("/interfaces", "Interfaces", "chevrons-left-right-ellipsis"),
				navItem("/logs", "Logs", "scroll-text"),
			),
			Div(Apply(Attr{"class": "h-px bg-line-1 mx-3 my-4"})),
			Div(
				Div(Apply(Attr{"class": "px-3 mb-2 text-[10px] font-semibold text-ink-3 uppercase tracking-[0.15em]"}), Text("Admin")),
				navItem("/users", "Users", "users"),
				navItem("/settings", "Settings", "settings"),
			),
		),

		// User section
		Show(func() bool { return currentUser() != nil }, func() loom.Node {
			u := currentUser()
			initial := string([]rune(u.Username)[0:1])
			return Div(
				Apply(Attr{"class": "border-t border-line-1 px-3 py-3"}),
				Div(
					Apply(Attr{"class": "flex items-center justify-between px-2 py-2"}),
					Div(
						Apply(Attr{"class": "flex items-center gap-2.5 min-w-0"}),
						Div(
							Apply(Attr{"class": "w-8 h-8 rounded-md bg-wg-600/15 flex items-center justify-center text-xs font-bold text-wg-400 uppercase flex-shrink-0"}),
							Text(initial),
						),
						Div(
							Apply(Attr{"class": "min-w-0"}),
							Div(Apply(Attr{"class": "text-sm font-medium text-ink-1 truncate"}), Text(u.Username)),
							Div(Apply(Attr{"class": "text-[10px] text-ink-3 uppercase tracking-wider"}), Text(u.Role)),
						),
					),
					Button(
						Apply(Attr{"class": "text-ink-4 hover:text-ink-2 transition-colors p-1.5 rounded-md hover:bg-surface-2 flex-shrink-0", "title": "Sign out"}),
						Apply(On{"click": func() {
							go func() {
								apiFetch("POST", "/api/v1/auth/logout", nil, nil)
								js.Global().Get("window").Get("location").Call("reload")
							}()
						}}),
						Icon("log-out", 15),
					),
				),
			)
		}),
	)
}

func navItem(href, label string, iconName string) loom.Node {
	return Bind(func() loom.Node {
		route := currentRoute()
		active := route == href || (href != "/" && strings.HasPrefix(route, href))

		class := "w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-[13px] font-medium transition-all duration-100 "
		if active {
			class += "bg-wg-600/10 border border-wg-600/20 text-wg-400"
		} else {
			class += "border border-transparent text-ink-2 hover:text-ink-1 hover:bg-surface-2/50"
		}

		return Button(
			Apply(Attr{"class": class}),
			Apply(On{"click": func() { navigate(href) }}),
			Icon(iconName, 17),
			Span(Text(label)),
		)
	})
}
