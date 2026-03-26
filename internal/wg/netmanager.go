package wg

import "errors"

// ErrNotSupported is returned on platforms where WireGuard interface management is not supported.
var ErrNotSupported = errors.New("wireguard interface management not supported on this platform")

// NetManager manages WireGuard network interfaces at the OS level.
type NetManager interface {
	// Create creates a new WireGuard interface.
	Create(name string) error
	// Delete removes a WireGuard interface.
	Delete(name string) error
	// SetAddress sets the address (CIDR) on the interface.
	SetAddress(name string, address string) error
	// SetMTU sets the MTU on the interface.
	SetMTU(name string, mtu int) error
	// SetUp brings the interface up.
	SetUp(name string) error
	// SetDown brings the interface down.
	SetDown(name string) error
	// Exists checks if the interface exists.
	Exists(name string) (bool, error)
	// QuickUp brings the interface up using wg-quick (handles PostUp, routes, etc).
	QuickUp(name string) error
	// QuickDown brings the interface down using wg-quick (handles PostDown, routes, etc).
	QuickDown(name string) error
	// SyncConf applies a WireGuard config using "wg syncconf" without restarting the interface.
	// This preserves routes, iptables rules, and existing handshake state.
	SyncConf(name string, confData string) error
	// SaveConf writes the full wg-quick config to /etc/wireguard/<name>.conf.
	SaveConf(name string, confData string) error
}
