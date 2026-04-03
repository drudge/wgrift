package wg

import "golang.zx2c4.com/wireguard/wgctrl/wgtypes"

// WGClient abstracts the wgctrl.Client methods used by Manager.
// The real *wgctrl.Client satisfies this interface.
type WGClient interface {
	Device(name string) (*wgtypes.Device, error)
	Close() error
}
