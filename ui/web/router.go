//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"

	"github.com/loom-go/loom"
	. "github.com/loom-go/loom/components"
	"github.com/loom-go/web"
	. "github.com/loom-go/web/components"
)

const routeContainerID = "route-content"

var (
	currentRoute func() string
	setRoute     func(string)
)

func initRouter() {
	currentRoute, setRoute = Signal("/")

	// Read initial path
	path := js.Global().Get("window").Get("location").Get("pathname").String()
	if path == "" {
		path = "/"
	}
	setRoute(path)

	// Listen for back/forward navigation
	callback := js.FuncOf(func(this js.Value, args []js.Value) any {
		path := js.Global().Get("window").Get("location").Get("pathname").String()
		if path == "" {
			path = "/"
		}
		setRoute(path)
		return nil
	})

	js.Global().Get("window").Call("addEventListener", "popstate", callback)
}

func navigate(path string) {
	js.Global().Get("window").Get("history").Call("pushState", nil, "", path)
	// Strip query string for route matching
	routePath := path
	if idx := strings.Index(routePath, "?"); idx >= 0 {
		routePath = routePath[:idx]
	}
	if routePath == currentRoute() {
		// Same route — force a refresh
		refreshRoute()
	} else {
		setRoute(routePath)
	}
}

func routeParam(prefix string) string {
	route := currentRoute()
	if strings.HasPrefix(route, prefix) {
		param := strings.TrimPrefix(route, prefix)
		if idx := strings.Index(param, "/"); idx >= 0 {
			param = param[:idx]
		}
		return param
	}
	return ""
}

// resolveRoute returns the view for the current route.
func resolveRoute() loom.Node {
	r := currentRoute()

	switch {
	case r == "/" || r == "":
		setPageTitle("Dashboard")
		return DashboardView()

	case r == "/interfaces":
		setPageTitle("Interfaces")
		return InterfacesView()

	case strings.Contains(r, "/peers/") && strings.HasSuffix(r, "/config"):
		parts := strings.Split(strings.TrimPrefix(r, "/interfaces/"), "/")
		if len(parts) >= 4 {
			setPageTitle("Peer Config")
			return PeerConfigView(parts[0], parts[2])
		}
		setPageTitle("")
		return DashboardView()

	case strings.HasPrefix(r, "/interfaces/") && !strings.Contains(r, "/peers/"):
		id := routeParam("/interfaces/")
		setPageTitle(id)
		return InterfaceDetailView(id)

	case r == "/logs":
		setPageTitle("Connection Logs")
		return LogsView("")

	case strings.HasPrefix(r, "/logs/"):
		setPageTitle("Connection Logs")
		return LogsView(routeParam("/logs/"))

	case r == "/users":
		setPageTitle("Users")
		return UsersView()

	case r == "/settings":
		setPageTitle("Settings")
		return SettingsView()

	default:
		setPageTitle("")
		return DashboardView()
	}
}

// stopPolling clears any active polling intervals from views.
func stopPolling() {
	for _, iv := range []*js.Value{&detailPollInterval, &dashboardPollInterval, &logsPollInterval} {
		if !iv.IsUndefined() && !iv.IsNull() {
			js.Global().Call("clearInterval", *iv)
			*iv = js.Undefined()
		}
	}
}

// mountRoute clears the route container and mounts a fresh view into it.
func mountRoute() {
	stopPolling()

	container := js.Global().Get("document").Call("getElementById", routeContainerID)
	if !container.Truthy() {
		return
	}

	container.Set("innerHTML", "")

	view := resolveRoute()
	subApp := web.NewApp()
	go func() {
		for range subApp.Run("#"+routeContainerID, func() loom.Node {
			return view
		}) {
		}
	}()
}

// refreshRoute re-mounts the current route view. Call this after mutations
// (create interface, add peer, etc.) to refresh data without a page reload.
func refreshRoute() {
	mountRoute()
}

// Router renders route content by clearing and re-mounting into a container
// div when the route changes.
func Router() loom.Node {
	var lastRoute string
	Effect(func() {
		r := currentRoute()
		if r == lastRoute {
			return
		}
		lastRoute = r

		// Defer mount to next tick so the #route-content div is in the DOM.
		js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
			mountRoute()
			return nil
		}), 0)
	})

	return Div(Apply(Attr{"id": routeContainerID}))
}
