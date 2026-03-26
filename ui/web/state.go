//go:build js && wasm

package main

import (
	"encoding/json"

	. "github.com/loom-go/loom/components"
)

// Pre-Loom auth state — set by checkSessionPreload before Loom starts.
var (
	preloadNeedsSetup bool
	preloadUser       *userData
	preloadCSRF       string
	preloadAuthed     bool
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
)

func initState() {
	currentUser, setCurrentUser = Signal(preloadUser)
	isAuthenticated, setIsAuthenticated = Signal(preloadAuthed)
	needsSetup, setNeedsSetup = Signal(preloadNeedsSetup)
	appLoading, setAppLoading = Signal(false)
	appError, setAppError = Signal("")
	csrfToken = preloadCSRF
}

// checkSessionPreload uses apiFetch to determine auth state before Loom starts.
func checkSessionPreload() {
	var resp apiResponse
	if err := apiFetch("GET", "/api/v1/auth/session", nil, &resp); err != nil {
		return
	}

	if resp.Error != "" {
		return
	}

	// Check for setup-needed response
	var setup setupCheck
	if err := unmarshalData(resp.Data, &setup); err == nil && setup.NeedsSetup {
		preloadNeedsSetup = true
		return
	}

	// Try parsing as session data
	var session sessionData
	if err := unmarshalData(resp.Data, &session); err == nil && session.User.ID != "" {
		preloadUser = &session.User
		preloadCSRF = session.CSRFToken
		preloadAuthed = true
	}
}

func unmarshalData(raw json.RawMessage, v any) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, v)
}
