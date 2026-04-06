//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"
	"time"

	. "github.com/loom-go/loom/components"
)

// Pre-Loom auth state — set by checkSessionPreload before Loom starts.
var (
	preloadNeedsSetup       bool
	preloadUser             *userData
	preloadCSRF             string
	preloadAuthed           bool
	preloadOIDCProviders    []oidcProviderInfo
	preloadLocalAuthEnabled bool
	preloadDemoMode         bool
	preloadSMTPEnabled      bool
)

// Loom reactive signals — set by initState, populated from preload values.
var (
	currentUser        func() *userData
	setCurrentUser     func(*userData)
	isAuthenticated    func() bool
	setIsAuthenticated func(bool)
	needsSetup         func() bool
	setNeedsSetup      func(bool)
	appLoading         func() bool
	setAppLoading      func(bool)
	appError           func() string
	setAppError        func(string)
	smtpEnabled        func() bool
	setSmtpEnabled     func(bool)
)

func initState() {
	currentUser, setCurrentUser = Signal(preloadUser)
	isAuthenticated, setIsAuthenticated = Signal(preloadAuthed)
	needsSetup, setNeedsSetup = Signal(preloadNeedsSetup)
	appLoading, setAppLoading = Signal(false)
	appError, setAppError = Signal("")
	smtpEnabled, setSmtpEnabled = Signal(preloadSMTPEnabled)
	csrfToken = preloadCSRF
}

// checkSessionPreload uses apiFetch to determine auth state before Loom starts.
// It retries a few times with backoff in case the server is still starting up.
func checkSessionPreload() {
	var resp apiResponse
	var err error

	for attempt := range 3 {
		resp = apiResponse{}
		err = apiFetchWithTimeout("GET", "/api/v1/auth/session", nil, &resp, 2*time.Second)
		if err == nil {
			break
		}
		// Brief pause before retry: 500ms, 1s, 2s
		sleepMs(500 << attempt)
	}

	if err != nil || resp.Error != "" {
		return
	}

	// Check for setup-needed response
	var setup setupCheck
	if err := unmarshalData(resp.Data, &setup); err == nil && setup.NeedsSetup {
		preloadNeedsSetup = true
		return
	}

	// Check for auth options response (not authenticated)
	var opts authOptions
	if err := unmarshalData(resp.Data, &opts); err == nil && opts.AuthRequired {
		preloadOIDCProviders = opts.OIDCProviders
		preloadLocalAuthEnabled = opts.LocalAuthEnabled
		preloadDemoMode = opts.Demo
		return
	}

	// Try parsing as session data (authenticated)
	var session sessionData
	if err := unmarshalData(resp.Data, &session); err == nil && session.User.ID != "" {
		preloadUser = &session.User
		preloadCSRF = session.CSRFToken
		preloadAuthed = true
		preloadSMTPEnabled = session.SMTPEnabled
	}
}

// sleepMs pauses the goroutine for the given number of milliseconds,
// yielding to the browser event loop via setTimeout.
func sleepMs(ms int) {
	done := make(chan struct{})
	js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
		close(done)
		return nil
	}), ms)
	<-done
}

func unmarshalData(raw json.RawMessage, v any) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, v)
}

// sessionWatchCb listens for visibilitychange to detect expired sessions
// when the user returns to the tab after being away.
var sessionWatchCb js.Func

func startSessionWatch() {
	sessionWatchCb = js.FuncOf(func(this js.Value, args []js.Value) any {
		if js.Global().Get("document").Get("visibilityState").String() == "visible" {
			go checkSessionAlive()
		}
		return nil
	})
	js.Global().Get("document").Call("addEventListener", "visibilitychange", sessionWatchCb)
}

func stopSessionWatch() {
	if sessionWatchCb.Truthy() {
		js.Global().Get("document").Call("removeEventListener", "visibilitychange", sessionWatchCb)
		sessionWatchCb.Release()
	}
}

func checkSessionAlive() {
	var resp apiResponse
	if err := apiFetchWithTimeout("GET", "/api/v1/auth/session", nil, &resp, 3*time.Second); err != nil {
		return
	}
	var opts authOptions
	if err := unmarshalData(resp.Data, &opts); err == nil && opts.AuthRequired {
		saveRedirectPath()
		js.Global().Get("window").Get("location").Call("reload")
	}
}

const redirectStorageKey = "wgrift_redirect"

// saveRedirectPath stores the current browser path in sessionStorage
// so we can return to it after login.
func saveRedirectPath() {
	loc := js.Global().Get("window").Get("location")
	path := loc.Get("pathname").String()
	search := loc.Get("search").String()
	if path != "" && path != "/" {
		js.Global().Get("sessionStorage").Call("setItem", redirectStorageKey, path+search)
	}
}

// consumeRedirectPath reads and removes the saved redirect path from sessionStorage.
// Returns empty string if none was saved.
func consumeRedirectPath() string {
	storage := js.Global().Get("sessionStorage")
	val := storage.Call("getItem", redirectStorageKey)
	if val.IsNull() || val.IsUndefined() {
		return ""
	}
	path := val.String()
	storage.Call("removeItem", redirectStorageKey)
	return path
}
