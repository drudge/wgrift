//go:build js && wasm

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"syscall/js"
	"time"
)

var csrfToken string

// apiFetch makes an HTTP request to the API and returns the parsed JSON response.
func apiFetch(method, path string, body any, result any) error {
	return apiFetchWithTimeout(method, path, body, result, 10*time.Second)
}

// apiFetchWithTimeout is like apiFetch but with a custom timeout.
func apiFetchWithTimeout(method, path string, body any, result any, timeout time.Duration) error {
	opts := js.Global().Get("Object").New()
	opts.Set("method", method)
	opts.Set("credentials", "same-origin")

	headers := js.Global().Get("Object").New()
	headers.Set("Content-Type", "application/json")
	if csrfToken != "" && method != "GET" {
		headers.Set("X-CSRF-Token", csrfToken)
	}
	opts.Set("headers", headers)

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling body: %w", err)
		}
		opts.Set("body", string(data))
	}

	// Set up an AbortController with timeout
	controller := js.Global().Get("AbortController").New()
	opts.Set("signal", controller.Get("signal"))
	timeoutID := js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
		controller.Call("abort")
		return nil
	}), int(timeout.Milliseconds()))

	// Make the fetch call synchronously using a channel
	done := make(chan error, 1)
	var responseText string

	thenFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		js.Global().Call("clearTimeout", timeoutID)
		response := args[0]
		status := response.Get("status").Int()

		// Session expired — reload to show login
		if status == 401 && path != "/api/v1/auth/login" && path != "/api/v1/auth/session" {
			js.Global().Get("window").Get("location").Call("reload")
			done <- fmt.Errorf("session expired")
			return nil
		}

		textPromise := response.Call("text")
		textPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			responseText = args[0].String()
			if status >= 400 {
				msg := extractErrorMessage(responseText)
				done <- &apiError{
					Status:  status,
					Message: msg,
					Raw:     fmt.Sprintf("HTTP %d: %s", status, responseText),
				}
			} else {
				done <- nil
			}
			return nil
		}))
		return nil
	})
	defer thenFn.Release()

	catchFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		js.Global().Call("clearTimeout", timeoutID)
		done <- fmt.Errorf("fetch error: %v", args[0])
		return nil
	})
	defer catchFn.Release()

	js.Global().Call("fetch", path, opts).Call("then", thenFn).Call("catch", catchFn)

	if err := <-done; err != nil {
		return err
	}

	if result != nil && responseText != "" {
		if err := json.Unmarshal([]byte(responseText), result); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
	}

	return nil
}

// extractErrorMessage tries to pull a human-readable message from a JSON error response.
// Handles wgRift format {"error":"msg"} and proxy/ingress format {"message":"...", "description":"..."}.
func extractErrorMessage(body string) string {
	var obj map[string]json.RawMessage
	if json.Unmarshal([]byte(body), &obj) != nil {
		return body
	}

	// wgRift: {"error": "human-readable string"}
	if raw, ok := obj["error"]; ok {
		var s string
		if json.Unmarshal(raw, &s) == nil && s != "" {
			return s
		}
	}

	// Proxy/ingress: {"message": "Bad Request", "description": "The server did not understand the request"}
	var message, description string
	if raw, ok := obj["message"]; ok {
		json.Unmarshal(raw, &message)
	}
	if raw, ok := obj["description"]; ok {
		json.Unmarshal(raw, &description)
	}
	if description != "" {
		return description
	}
	if message != "" {
		return message
	}

	return body
}

// apiError is returned when the server responds with an HTTP error status.
type apiError struct {
	Status  int
	Message string // human-friendly, extracted from JSON {"error":"..."}
	Raw     string // full response body for debugging
}

func (e *apiError) Error() string {
	return e.Message
}

// ErrorInfo holds a user-facing error message and optional technical detail.
type ErrorInfo struct {
	Message string
	Detail  string
}

// apiErrorInfo extracts ErrorInfo from an error, including raw details if available.
func apiErrorInfo(err error) ErrorInfo {
	var ae *apiError
	if errors.As(err, &ae) {
		return ErrorInfo{Message: ae.Message, Detail: ae.Raw}
	}
	return ErrorInfo{Message: err.Error()}
}

// API response types
type apiResponse struct {
	Data  json.RawMessage `json:"data"`
	Error string          `json:"error"`
	Total *int            `json:"total"`
}

type sessionData struct {
	User      userData `json:"user"`
	CSRFToken string   `json:"csrf_token"`
}

type userData struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	Role         string `json:"role"`
	IsInitial    bool   `json:"is_initial"`
	OIDCProvider string `json:"oidc_provider,omitempty"`
	OIDCIssuer   string `json:"oidc_issuer,omitempty"`
}

type setupCheck struct {
	NeedsSetup bool `json:"needs_setup"`
}

type oidcProviderInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	LoginURL string `json:"login_url"`
}

type authOptions struct {
	AuthRequired     bool               `json:"auth_required"`
	OIDCProviders    []oidcProviderInfo `json:"oidc_providers"`
	LocalAuthEnabled bool               `json:"local_auth_enabled"`
	Demo             bool               `json:"demo"`
}

type settingsData struct {
	ExternalURL      string             `json:"external_url"`
	OIDCProviders    []oidcProviderData `json:"oidc_providers"`
	LocalAuthEnabled bool               `json:"local_auth_enabled"`
}

type oidcProviderData struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Issuer       string `json:"issuer"`
	ClientID     string `json:"client_id"`
	Scopes       string `json:"scopes"`
	AutoDiscover bool   `json:"auto_discover"`
	AdminClaim   string `json:"admin_claim"`
	AdminValue   string `json:"admin_value"`
	DefaultRole  string `json:"default_role"`
	Enabled      bool   `json:"enabled"`
}

type interfaceData struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	ListenPort int    `json:"listen_port"`
	Address    string `json:"address"`
	DNS        string `json:"dns"`
	MTU        int    `json:"mtu"`
	Endpoint   string `json:"endpoint"`
	Enabled    bool   `json:"enabled"`
}

type peerData struct {
	ID                  string `json:"id"`
	InterfaceID         string `json:"interface_id"`
	Type                string `json:"type"`
	Name                string `json:"name"`
	PublicKey           string `json:"public_key"`
	Address             string `json:"address"`
	AllowedIPs          string `json:"allowed_ips"`
	ClientAllowedIPs    string `json:"client_allowed_ips"`
	DNS                 string `json:"dns"`
	Endpoint            string `json:"endpoint"`
	PersistentKeepalive int    `json:"persistent_keepalive"`
	Enabled             bool   `json:"enabled"`
	TransferRx          int64  `json:"transfer_rx"`
	TransferTx          int64  `json:"transfer_tx"`
}

type activeConnectionData struct {
	InterfaceID   string `json:"interface_id"`
	PeerID        string `json:"peer_id"`
	PeerName      string `json:"peer_name"`
	PeerType      string `json:"peer_type"`
	Address       string `json:"address"`
	Endpoint      string `json:"endpoint,omitempty"`
	LastHandshake string `json:"last_handshake"`
	TransferRx    int64  `json:"transfer_rx"`
	TransferTx    int64  `json:"transfer_tx"`
}

type dashboardData struct {
	Interfaces        []interfaceSummaryData `json:"interfaces"`
	ActiveConnections []activeConnectionData `json:"active_connections"`
	TotalPeers        int                    `json:"total_peers"`
	ActivePeers       int                    `json:"active_peers"`
	TotalRx           int64                  `json:"total_rx"`
	TotalTx           int64                  `json:"total_tx"`
}

type interfaceSummaryData struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	Address        string `json:"address"`
	ListenPort     int    `json:"listen_port"`
	Enabled        bool   `json:"enabled"`
	Running        bool   `json:"running"`
	PublicKey      string `json:"public_key"`
	TotalPeers     int    `json:"total_peers"`
	ConnectedPeers int    `json:"connected_peers"`
	TotalRx        int64  `json:"total_rx"`
	TotalTx        int64  `json:"total_tx"`
}

type peerStatusData struct {
	Peer          peerData `json:"peer"`
	HasPrivateKey bool     `json:"has_private_key"`
	LastHandshake string   `json:"last_handshake"`
	TransferRx    int64    `json:"transfer_rx"`
	TransferTx    int64    `json:"transfer_tx"`
	Connected     bool     `json:"connected"`
	Endpoint      string   `json:"endpoint,omitempty"`
}

type interfaceStatusData struct {
	Interface interfaceData    `json:"interface"`
	PublicKey string           `json:"public_key"`
	Running   bool             `json:"running"`
	Peers     []peerStatusData `json:"peers"`
}

type connectionLogData struct {
	ID          int64  `json:"id"`
	PeerID      string `json:"peer_id"`
	PeerName    string `json:"peer_name"`
	InterfaceID string `json:"interface_id"`
	Event       string `json:"event"`
	Endpoint    string `json:"endpoint,omitempty"`
	TransferRx  int64  `json:"transfer_rx"`
	TransferTx  int64  `json:"transfer_tx"`
	RecordedAt  string `json:"recorded_at"`
}
