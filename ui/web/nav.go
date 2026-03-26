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
		Apply(Attr{"class": "md:hidden fixed top-0 left-0 right-0 z-40 bg-white border-b border-gray-200 flex items-center justify-between px-4 h-14"}),
		// Brand
		Div(
			Apply(Attr{"class": "flex items-center gap-2"}),
			Div(
				Apply(Attr{"class": "w-7 h-7 rounded-lg bg-teal-50 border border-teal-200 flex items-center justify-center text-teal-600"}),
				Icon("shield", 16),
			),
			Div(
				Apply(Attr{"class": "text-sm font-semibold tracking-tight"}),
				Span(Apply(Attr{"class": "text-teal-600"}), Text("wg")),
				Span(Apply(Attr{"class": "text-gray-900"}), Text("Rift")),
			),
		),
		// Hamburger
		Button(
			Apply(Attr{"class": "w-9 h-9 rounded-md flex items-center justify-center text-gray-500 hover:bg-gray-100 transition-colors"}),
			Apply(On{"click": func() { setMobileNavOpen(!mobileNavOpen()) }}),
			Icon("menu", 20),
		),
	)
}

// MobileNavOverlay renders a slide-out nav drawer on mobile.
func MobileNavOverlay() loom.Node {
	return Bind(func() loom.Node {
		open := mobileNavOpen()
		backdropClass := "md:hidden fixed inset-0 z-40 bg-black/40"
		drawerClass := "fixed top-0 left-0 bottom-0 z-50 w-64 bg-white shadow-xl transform transition-transform"
		if !open {
			backdropClass = "hidden"
			drawerClass = "hidden"
		}
		return Div(
			// Backdrop
			Div(
				Apply(Attr{"class": backdropClass}),
				Apply(On{"click": func() { setMobileNavOpen(false) }}),
			),
			// Drawer
			Elem("nav",
				Apply(Attr{"class": drawerClass}),
				mobileNavContent(),
			),
		)
	})
}

func mobileNavContent() loom.Node {
	return Div(
		Apply(Attr{"class": "flex flex-col h-full"}),
		// Header
		Div(
			Apply(Attr{"class": "px-5 py-5 flex items-center justify-between"}),
			Div(
				Apply(Attr{"class": "flex items-center gap-2.5"}),
				Div(
					Apply(Attr{"class": "w-8 h-8 rounded-lg bg-teal-50 border border-teal-200 flex items-center justify-center text-teal-600"}),
					Icon("shield", 18),
				),
				Div(
					Apply(Attr{"class": "text-base font-semibold tracking-tight"}),
					Span(Apply(Attr{"class": "text-teal-600"}), Text("wg")),
					Span(Apply(Attr{"class": "text-gray-900"}), Text("Rift")),
				),
			),
			Button(
				Apply(Attr{"class": "w-8 h-8 rounded-md flex items-center justify-center text-gray-400 hover:bg-gray-100"}),
				Apply(On{"click": func() { setMobileNavOpen(false) }}),
				Icon("x", 18),
			),
		),
		// Nav items
		Div(
			Apply(Attr{"class": "flex-1 px-3 py-2"}),
			Div(
				Apply(Attr{"class": "space-y-0.5"}),
				mobileNavItem("/", "Status", "activity"),
				mobileNavItem("/interfaces", "Interfaces", "network"),
				mobileNavItem("/logs", "Logs", "scroll-text"),
			),
			Div(
				Apply(Attr{"class": "mt-6"}),
				Div(Apply(Attr{"class": "px-3 mb-2 text-[10px] font-semibold text-gray-400 uppercase tracking-widest"}), Text("Admin")),
				mobileNavItem("/users", "Users", "users"),
			),
		),
		// User section
		Show(func() bool { return currentUser() != nil }, func() loom.Node {
			u := currentUser()
			initial := string([]rune(u.Username)[0:1])
			return Div(
				Apply(Attr{"class": "border-t border-gray-200 px-3 py-3"}),
				Div(
					Apply(Attr{"class": "flex items-center justify-between px-3 py-2"}),
					Div(
						Apply(Attr{"class": "flex items-center gap-2.5 min-w-0"}),
						Div(
							Apply(Attr{"class": "w-7 h-7 rounded-full bg-gray-100 border border-gray-200 flex items-center justify-center text-[11px] font-medium text-gray-500 uppercase flex-shrink-0"}),
							Text(initial),
						),
						Div(
							Apply(Attr{"class": "min-w-0"}),
							Div(Apply(Attr{"class": "text-sm font-medium text-gray-700 truncate"}), Text(u.Username)),
							Div(Apply(Attr{"class": "text-[11px] text-gray-400"}), Text(u.Role)),
						),
					),
					Button(
						Apply(Attr{"class": "text-gray-400 hover:text-gray-600 transition-colors p-1 flex-shrink-0", "title": "Sign out"}),
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

func mobileNavItem(href, label string, iconName string) loom.Node {
	return Bind(func() loom.Node {
		route := currentRoute()
		active := route == href || (href != "/" && strings.HasPrefix(route, href))

		class := "w-full flex items-center gap-2.5 px-3 py-2.5 rounded-md text-sm font-medium transition-colors "
		if active {
			class += "bg-teal-50 text-teal-700"
		} else {
			class += "text-gray-600 hover:text-gray-900 hover:bg-gray-50"
		}

		return Button(
			Apply(Attr{"class": class}),
			Apply(On{"click": func() {
				setMobileNavOpen(false)
				navigate(href)
			}}),
			Icon(iconName, 18),
			Span(Text(label)),
		)
	})
}

// NavRail renders the sidebar navigation with text labels.
func NavRail() loom.Node {
	return Elem("nav",
		Apply(Attr{"class": "w-56 bg-white border-r border-gray-200 flex flex-col flex-shrink-0 overflow-y-auto"}),

		// Header / brand
		Div(
			Apply(Attr{"class": "px-5 py-5 flex items-center gap-2.5"}),
			Div(
				Apply(Attr{"class": "w-8 h-8 rounded-lg bg-teal-50 border border-teal-200 flex items-center justify-center text-teal-600"}),
				Icon("shield", 18),
			),
			Div(
				Apply(Attr{"class": "text-base font-semibold tracking-tight"}),
				Span(Apply(Attr{"class": "text-teal-600"}), Text("wg")),
				Span(Apply(Attr{"class": "text-gray-900"}), Text("Rift")),
			),
		),

		// Main nav
		Div(
			Apply(Attr{"class": "flex-1 px-3 py-2"}),
			Div(
				Apply(Attr{"class": "space-y-0.5"}),
				navItem("/", "Status", "activity"),
				navItem("/interfaces", "Interfaces", "network"),
				navItem("/logs", "Logs", "scroll-text"),
			),
			Div(
				Apply(Attr{"class": "mt-6"}),
				Div(Apply(Attr{"class": "px-3 mb-2 text-[10px] font-semibold text-gray-400 uppercase tracking-widest"}), Text("Admin")),
				navItem("/users", "Users", "users"),
			),
		),

		// User section
		Show(func() bool { return currentUser() != nil }, func() loom.Node {
			u := currentUser()
			initial := string([]rune(u.Username)[0:1])
			return Div(
				Apply(Attr{"class": "border-t border-gray-200 px-3 py-3"}),
				Div(
					Apply(Attr{"class": "flex items-center justify-between px-3 py-2"}),
					Div(
						Apply(Attr{"class": "flex items-center gap-2.5 min-w-0"}),
						Div(
							Apply(Attr{"class": "w-7 h-7 rounded-full bg-gray-100 border border-gray-200 flex items-center justify-center text-[11px] font-medium text-gray-500 uppercase flex-shrink-0"}),
							Text(initial),
						),
						Div(
							Apply(Attr{"class": "min-w-0"}),
							Div(Apply(Attr{"class": "text-sm font-medium text-gray-700 truncate"}), Text(u.Username)),
							Div(Apply(Attr{"class": "text-[11px] text-gray-400"}), Text(u.Role)),
						),
					),
					Button(
						Apply(Attr{"class": "text-gray-400 hover:text-gray-600 transition-colors p-1 flex-shrink-0", "title": "Sign out"}),
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

		class := "w-full flex items-center gap-2.5 px-3 py-2 rounded-md text-sm font-medium transition-colors "
		if active {
			class += "bg-teal-50 text-teal-700"
		} else {
			class += "text-gray-600 hover:text-gray-900 hover:bg-gray-50"
		}

		return Button(
			Apply(Attr{"class": class}),
			Apply(On{"click": func() { navigate(href) }}),
			Icon(iconName, 18),
			Span(Text(label)),
		)
	})
}
