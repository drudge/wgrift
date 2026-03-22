package wg

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/drudge/wgrift/internal/confgen"
	"github.com/drudge/wgrift/internal/crypto"
	"github.com/drudge/wgrift/internal/models"
	"github.com/drudge/wgrift/internal/store"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// Manager orchestrates WireGuard interface and peer management.
type Manager struct {
	store      store.Store
	enc        *crypto.Encryptor
	net        NetManager
	wg         *wgctrl.Client
	externalIP string
}

// NewManager creates a new WireGuard manager.
func NewManager(s store.Store, enc *crypto.Encryptor, nm NetManager, externalIP string) (*Manager, error) {
	wgClient, err := wgctrl.New()
	if err != nil {
		return nil, fmt.Errorf("creating wgctrl client: %w", err)
	}

	return &Manager{
		store:      s,
		enc:        enc,
		net:        nm,
		wg:         wgClient,
		externalIP: externalIP,
	}, nil
}

// Close closes the wgctrl client.
func (m *Manager) Close() error {
	return m.wg.Close()
}

// CreateInterface creates a new WireGuard interface in the database and optionally on the system.
func (m *Manager) CreateInterface(iface *models.Interface) error {
	// Generate keys
	privKey, pubKey, err := crypto.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generating keys: %w", err)
	}
	_ = pubKey // Public key is derived when needed

	// Encrypt private key
	encrypted, err := m.enc.Encrypt(privKey)
	if err != nil {
		return fmt.Errorf("encrypting private key: %w", err)
	}
	iface.PrivateKeyEncrypted = encrypted
	iface.Enabled = true

	if err := m.store.CreateInterface(iface); err != nil {
		return fmt.Errorf("storing interface: %w", err)
	}

	return nil
}

// DeleteInterface removes an interface from the database and system.
func (m *Manager) DeleteInterface(id string, force bool) error {
	iface, err := m.store.GetInterface(id)
	if err != nil {
		return fmt.Errorf("getting interface: %w", err)
	}
	if iface == nil {
		return fmt.Errorf("interface %q not found", id)
	}

	if !force {
		peers, err := m.store.ListPeers(id)
		if err != nil {
			return fmt.Errorf("listing peers: %w", err)
		}
		if len(peers) > 0 {
			return fmt.Errorf("interface %q has %d peers; use --force to delete", id, len(peers))
		}
	}

	// Try to remove from system (ignore errors on non-Linux)
	_ = m.net.Delete(id)

	if err := m.store.DeleteInterface(id); err != nil {
		return fmt.Errorf("deleting interface from store: %w", err)
	}

	return nil
}

// SyncInterface syncs the database state to the kernel for a single interface.
func (m *Manager) SyncInterface(id string) error {
	iface, err := m.store.GetInterface(id)
	if err != nil {
		return fmt.Errorf("getting interface: %w", err)
	}
	if iface == nil {
		return fmt.Errorf("interface %q not found", id)
	}

	if !iface.Enabled {
		// If disabled, try to bring down
		_ = m.net.SetDown(id)
		return nil
	}

	// Ensure interface exists
	exists, err := m.net.Exists(id)
	if err != nil {
		return fmt.Errorf("checking interface: %w", err)
	}
	if !exists {
		if err := m.net.Create(id); err != nil {
			return fmt.Errorf("creating interface: %w", err)
		}
	}

	// Set address and MTU
	if err := m.net.SetAddress(id, iface.Address); err != nil {
		return fmt.Errorf("setting address: %w", err)
	}
	if err := m.net.SetMTU(id, iface.MTU); err != nil {
		return fmt.Errorf("setting MTU: %w", err)
	}

	// Decrypt private key
	privKey, err := m.enc.Decrypt(iface.PrivateKeyEncrypted)
	if err != nil {
		return fmt.Errorf("decrypting private key: %w", err)
	}

	key, err := wgtypes.ParseKey(privKey)
	if err != nil {
		return fmt.Errorf("parsing private key: %w", err)
	}

	// Build peer configs
	peers, err := m.store.ListPeers(id)
	if err != nil {
		return fmt.Errorf("listing peers: %w", err)
	}

	var peerConfigs []wgtypes.PeerConfig
	for _, p := range peers {
		if !p.Enabled {
			continue
		}

		pubKey, err := wgtypes.ParseKey(p.PublicKey)
		if err != nil {
			return fmt.Errorf("parsing peer public key: %w", err)
		}

		pc := wgtypes.PeerConfig{
			PublicKey:  pubKey,
			AllowedIPs: parseAllowedIPs(p.AllowedIPs),
		}

		if p.PresharedKeyEncrypted != "" {
			pskPlain, err := m.enc.Decrypt(p.PresharedKeyEncrypted)
			if err != nil {
				return fmt.Errorf("decrypting preshared key: %w", err)
			}
			psk, err := wgtypes.ParseKey(pskPlain)
			if err != nil {
				return fmt.Errorf("parsing preshared key: %w", err)
			}
			pc.PresharedKey = &psk
		}

		if p.Endpoint != "" {
			addr, err := net.ResolveUDPAddr("udp", p.Endpoint)
			if err != nil {
				return fmt.Errorf("resolving endpoint %s: %w", p.Endpoint, err)
			}
			pc.Endpoint = addr
		}

		if p.PersistentKeepalive > 0 {
			d := time.Duration(p.PersistentKeepalive) * time.Second
			pc.PersistentKeepaliveInterval = &d
		}

		peerConfigs = append(peerConfigs, pc)
	}

	listenPort := iface.ListenPort
	cfg := wgtypes.Config{
		PrivateKey:   &key,
		ListenPort:   &listenPort,
		ReplacePeers: true,
		Peers:        peerConfigs,
	}

	if err := m.wg.ConfigureDevice(id, cfg); err != nil {
		return fmt.Errorf("configuring device: %w", err)
	}

	if err := m.net.SetUp(id); err != nil {
		return fmt.Errorf("bringing up interface: %w", err)
	}

	return nil
}

// SyncAll syncs all enabled interfaces.
func (m *Manager) SyncAll() error {
	ifaces, err := m.store.ListInterfaces()
	if err != nil {
		return fmt.Errorf("listing interfaces: %w", err)
	}

	var errs []string
	for _, iface := range ifaces {
		if err := m.SyncInterface(iface.ID); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", iface.ID, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("sync errors:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

// AddPeer creates a new peer with auto-generated keys.
func (m *Manager) AddPeer(peer *models.Peer, generatePSK bool) error {
	// Verify interface exists
	iface, err := m.store.GetInterface(peer.InterfaceID)
	if err != nil {
		return fmt.Errorf("getting interface: %w", err)
	}
	if iface == nil {
		return fmt.Errorf("interface %q not found", peer.InterfaceID)
	}

	// Generate key pair for the peer
	privKey, pubKey, err := crypto.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generating peer keys: %w", err)
	}

	peer.PublicKey = pubKey

	// Encrypt and store private key
	encPriv, err := m.enc.Encrypt(privKey)
	if err != nil {
		return fmt.Errorf("encrypting peer private key: %w", err)
	}
	peer.PrivateKeyEncrypted = encPriv

	if generatePSK {
		psk, err := crypto.GeneratePresharedKey()
		if err != nil {
			return fmt.Errorf("generating preshared key: %w", err)
		}
		encPSK, err := m.enc.Encrypt(psk)
		if err != nil {
			return fmt.Errorf("encrypting preshared key: %w", err)
		}
		peer.PresharedKeyEncrypted = encPSK
	}

	peer.Enabled = true

	if err := m.store.CreatePeer(peer); err != nil {
		return fmt.Errorf("storing peer: %w", err)
	}

	return nil
}

// RemovePeer deletes a peer from the database.
func (m *Manager) RemovePeer(id string) error {
	peer, err := m.store.GetPeer(id)
	if err != nil {
		return fmt.Errorf("getting peer: %w", err)
	}
	if peer == nil {
		return fmt.Errorf("peer %q not found", id)
	}

	if err := m.store.DeletePeer(id); err != nil {
		return fmt.Errorf("deleting peer: %w", err)
	}
	return nil
}

// EnablePeer enables a peer.
func (m *Manager) EnablePeer(id string) error {
	return m.setPeerEnabled(id, true)
}

// DisablePeer disables a peer without deleting it.
func (m *Manager) DisablePeer(id string) error {
	return m.setPeerEnabled(id, false)
}

func (m *Manager) setPeerEnabled(id string, enabled bool) error {
	peer, err := m.store.GetPeer(id)
	if err != nil {
		return fmt.Errorf("getting peer: %w", err)
	}
	if peer == nil {
		return fmt.Errorf("peer %q not found", id)
	}

	peer.Enabled = enabled
	if err := m.store.UpdatePeer(peer); err != nil {
		return fmt.Errorf("updating peer: %w", err)
	}
	return nil
}

// InterfaceStatus holds live status for an interface.
type InterfaceStatus struct {
	Interface models.Interface
	PublicKey string
	Peers     []PeerStatus
}

// PeerStatus holds live status for a peer.
type PeerStatus struct {
	Peer          models.Peer
	LastHandshake time.Time
	TransferRx    int64
	TransferTx    int64
	Connected     bool
}

// GetStatus returns live status for an interface from wgctrl.
func (m *Manager) GetStatus(interfaceID string) (*InterfaceStatus, error) {
	iface, err := m.store.GetInterface(interfaceID)
	if err != nil {
		return nil, fmt.Errorf("getting interface: %w", err)
	}
	if iface == nil {
		return nil, fmt.Errorf("interface %q not found", interfaceID)
	}

	// Get public key
	privKey, err := m.enc.Decrypt(iface.PrivateKeyEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypting private key: %w", err)
	}
	pubKey, err := crypto.PublicKeyFromPrivate(privKey)
	if err != nil {
		return nil, fmt.Errorf("deriving public key: %w", err)
	}

	status := &InterfaceStatus{
		Interface: *iface,
		PublicKey: pubKey,
	}

	dbPeers, err := m.store.ListPeers(interfaceID)
	if err != nil {
		return nil, fmt.Errorf("listing peers: %w", err)
	}

	// Try to get live data from wgctrl
	device, err := m.wg.Device(interfaceID)
	if err != nil {
		// Interface might not be synced yet; return DB data only
		for _, p := range dbPeers {
			status.Peers = append(status.Peers, PeerStatus{Peer: p})
		}
		return status, nil
	}

	// Build a map of live peer data by public key
	livePeers := make(map[string]wgtypes.Peer)
	for _, p := range device.Peers {
		livePeers[p.PublicKey.String()] = p
	}

	threshold := 180 * time.Second
	now := time.Now()

	for _, p := range dbPeers {
		ps := PeerStatus{Peer: p}
		if live, ok := livePeers[p.PublicKey]; ok {
			ps.LastHandshake = live.LastHandshakeTime
			ps.TransferRx = live.ReceiveBytes
			ps.TransferTx = live.TransmitBytes
			if !live.LastHandshakeTime.IsZero() && now.Sub(live.LastHandshakeTime) < threshold {
				ps.Connected = true
			}
		}
		status.Peers = append(status.Peers, ps)
	}

	return status, nil
}

// GenerateConfig generates a WireGuard .conf for a peer (client config).
func (m *Manager) GenerateConfig(peerID string) (string, error) {
	peer, err := m.store.GetPeer(peerID)
	if err != nil {
		return "", fmt.Errorf("getting peer: %w", err)
	}
	if peer == nil {
		return "", fmt.Errorf("peer %q not found", peerID)
	}

	iface, err := m.store.GetInterface(peer.InterfaceID)
	if err != nil {
		return "", fmt.Errorf("getting interface: %w", err)
	}
	if iface == nil {
		return "", fmt.Errorf("interface %q not found", peer.InterfaceID)
	}

	// Decrypt peer's private key
	privKey, err := m.enc.Decrypt(peer.PrivateKeyEncrypted)
	if err != nil {
		return "", fmt.Errorf("decrypting peer private key: %w", err)
	}

	// Get server public key
	serverPrivKey, err := m.enc.Decrypt(iface.PrivateKeyEncrypted)
	if err != nil {
		return "", fmt.Errorf("decrypting server private key: %w", err)
	}
	serverPubKey, err := crypto.PublicKeyFromPrivate(serverPrivKey)
	if err != nil {
		return "", fmt.Errorf("deriving server public key: %w", err)
	}

	params := confgen.PeerConfParams{
		PrivateKey:      privKey,
		Address:         peer.AllowedIPs,
		DNS:             iface.DNS,
		ServerPublicKey: serverPubKey,
		ServerEndpoint:  fmt.Sprintf("%s:%d", m.externalIP, iface.ListenPort),
		AllowedIPs:      "0.0.0.0/0, ::/0",
		MTU:             iface.MTU,
	}

	// For site-to-site, AllowedIPs should be more specific
	if iface.Type == models.InterfaceTypeSiteToSite {
		params.AllowedIPs = peer.AllowedIPs
	}

	// Decrypt preshared key if present
	if peer.PresharedKeyEncrypted != "" {
		psk, err := m.enc.Decrypt(peer.PresharedKeyEncrypted)
		if err != nil {
			return "", fmt.Errorf("decrypting preshared key: %w", err)
		}
		params.PresharedKey = psk
	}

	if peer.PersistentKeepalive > 0 {
		params.PersistentKeepalive = peer.PersistentKeepalive
	}

	return confgen.GeneratePeerConf(params), nil
}

func parseAllowedIPs(s string) []net.IPNet {
	var nets []net.IPNet
	for _, cidr := range strings.Split(s, ",") {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		nets = append(nets, *ipNet)
	}
	return nets
}
