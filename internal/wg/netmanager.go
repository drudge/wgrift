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
}
