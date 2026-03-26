//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"
)

var csrfToken string

// apiFetch makes an HTTP request to the API and returns the parsed JSON response.
func apiFetch(method, path string, body any, result any) error {
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

	// Make the fetch call synchronously using a channel
	done := make(chan error, 1)
	var responseText string

	thenFn := js.FuncOf(func(this js.Value, args []js.Value) any {
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
				done <- fmt.Errorf("HTTP %d: %s", status, responseText)
			} else {
				done <- nil
			}
			return nil
		}))
		return nil
	})
	defer thenFn.Release()

	catchFn := js.FuncOf(func(this js.Value, args []js.Value) any {
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
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	IsInitial   bool   `json:"is_initial"`
}

type setupCheck struct {
	NeedsSetup bool `json:"needs_setup"`
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
	Name                string `json:"name"`
	PublicKey           string `json:"public_key"`
	Address             string `json:"address"`
	AllowedIPs          string `json:"allowed_ips"`
	ClientAllowedIPs    string `json:"client_allowed_ips"`
	Endpoint            string `json:"endpoint"`
	PersistentKeepalive int    `json:"persistent_keepalive"`
	Enabled             bool   `json:"enabled"`
	TransferRx          int64  `json:"transfer_rx"`
	TransferTx          int64  `json:"transfer_tx"`
}

type dashboardData struct {
	Interfaces  []interfaceSummaryData `json:"interfaces"`
	TotalPeers  int                    `json:"total_peers"`
	ActivePeers int                    `json:"active_peers"`
	TotalRx     int64                  `json:"total_rx"`
	TotalTx     int64                  `json:"total_tx"`
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
	TransferRx  int64  `json:"transfer_rx"`
	TransferTx  int64  `json:"transfer_tx"`
	RecordedAt  string `json:"recorded_at"`
}
