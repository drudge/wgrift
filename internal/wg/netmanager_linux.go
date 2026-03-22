//go:build linux

package wg

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

type linuxNetManager struct{}

// NewNetManager returns a Linux netlink-based NetManager.
func NewNetManager() NetManager {
	return &linuxNetManager{}
}

func (m *linuxNetManager) Create(name string) error {
	attrs := netlink.NewLinkAttrs()
	attrs.Name = name
	link := &netlink.GenericLink{
		LinkAttrs: attrs,
		LinkType:  "wireguard",
	}
	if err := netlink.LinkAdd(link); err != nil {
		return fmt.Errorf("creating interface %s: %w", name, err)
	}
	return nil
}

func (m *linuxNetManager) Delete(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("finding interface %s: %w", name, err)
	}
	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("deleting interface %s: %w", name, err)
	}
	return nil
}

func (m *linuxNetManager) SetAddress(name string, address string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("finding interface %s: %w", name, err)
	}

	addr, err := netlink.ParseAddr(address)
	if err != nil {
		return fmt.Errorf("parsing address %s: %w", address, err)
	}

	if err := netlink.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("adding address to %s: %w", name, err)
	}
	return nil
}

func (m *linuxNetManager) SetMTU(name string, mtu int) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("finding interface %s: %w", name, err)
	}
	if err := netlink.LinkSetMTU(link, mtu); err != nil {
		return fmt.Errorf("setting MTU on %s: %w", name, err)
	}
	return nil
}

func (m *linuxNetManager) SetUp(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("finding interface %s: %w", name, err)
	}
	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("bringing up %s: %w", name, err)
	}
	return nil
}

func (m *linuxNetManager) SetDown(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("finding interface %s: %w", name, err)
	}
	if err := netlink.LinkSetDown(link); err != nil {
		return fmt.Errorf("bringing down %s: %w", name, err)
	}
	return nil
}

func (m *linuxNetManager) Exists(name string) (bool, error) {
	_, err := net.InterfaceByName(name)
	if err != nil {
		if _, ok := err.(*net.OpError); ok {
			return false, nil
		}
		return false, nil
	}
	return true, nil
}
