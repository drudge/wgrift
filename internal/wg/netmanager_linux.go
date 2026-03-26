//go:build linux

package wg

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"

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

	if err := netlink.AddrReplace(link, addr); err != nil {
		return fmt.Errorf("setting address on %s: %w", name, err)
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

func (m *linuxNetManager) QuickUp(name string) error {
	out, err := exec.Command("wg-quick", "up", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("wg-quick up %s: %w: %s", name, err, out)
	}
	return nil
}

func (m *linuxNetManager) QuickDown(name string) error {
	out, err := exec.Command("wg-quick", "down", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("wg-quick down %s: %w: %s", name, err, out)
	}
	return nil
}

func (m *linuxNetManager) SyncConf(name string, confData string) error {
	// Write stripped config to temp file
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("wgrift-%s-*.conf", name))
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(confData); err != nil {
		tmpFile.Close()
		return fmt.Errorf("writing temp config: %w", err)
	}
	tmpFile.Close()

	out, err := exec.Command("wg", "syncconf", name, tmpFile.Name()).CombinedOutput()
	if err != nil {
		return fmt.Errorf("wg syncconf %s: %w: %s", name, err, out)
	}
	return nil
}

func (m *linuxNetManager) SaveConf(name string, confData string) error {
	confPath := filepath.Join("/etc/wireguard", name+".conf")
	if err := os.WriteFile(confPath, []byte(confData), 0600); err != nil {
		return fmt.Errorf("writing config %s: %w", confPath, err)
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
