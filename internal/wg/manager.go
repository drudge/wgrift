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

	if externalIP == "" {
		externalIP = detectExternalIP()
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

// SyncInterface syncs the database state to the running WireGuard interface.
// It uses "wg syncconf" to apply changes without disrupting routes or iptables rules,
// and updates /etc/wireguard/<name>.conf so wg-quick stays in sync.
func (m *Manager) SyncInterface(id string) error {
	iface, err := m.store.GetInterface(id)
	if err != nil {
		return fmt.Errorf("getting interface: %w", err)
	}
	if iface == nil {
		return fmt.Errorf("interface %q not found", id)
	}

	if !iface.Enabled {
		_ = m.net.SetDown(id)
		return nil
	}

	// Decrypt private key
	privKey, err := m.enc.Decrypt(iface.PrivateKeyEncrypted)
	if err != nil {
		return fmt.Errorf("decrypting private key: %w", err)
	}

	// Build peer blocks for config generation
	peers, err := m.store.ListPeers(id)
	if err != nil {
		return fmt.Errorf("listing peers: %w", err)
	}

	var peerBlocks []confgen.ServerPeerBlock
	for _, p := range peers {
		if !p.Enabled {
			continue
		}

		// Build AllowedIPs: always include the peer's tunnel address
		allowedIPs := p.AllowedIPs
		if p.Address != "" {
			tunnelIP := ensureHost32(p.Address)
			if tunnelIP != "" {
				allowedIPs = mergeAllowedIPStrings(tunnelIP, allowedIPs)
			}
		}

		pb := confgen.ServerPeerBlock{
			Name:                p.Name,
			PublicKey:           p.PublicKey,
			AllowedIPs:          allowedIPs,
			Endpoint:            p.Endpoint,
			PersistentKeepalive: p.PersistentKeepalive,
		}

		if p.PresharedKeyEncrypted != "" {
			psk, err := m.enc.Decrypt(p.PresharedKeyEncrypted)
			if err != nil {
				return fmt.Errorf("decrypting preshared key: %w", err)
			}
			pb.PresharedKey = psk
		}

		peerBlocks = append(peerBlocks, pb)
	}

	params := confgen.ServerConfParams{
		PrivateKey: privKey,
		Address:    iface.Address,
		ListenPort: iface.ListenPort,
		MTU:        iface.MTU,
		DNS:        iface.DNS,
		Peers:      peerBlocks,
	}

	// Check if interface is running — if so, use wg syncconf for non-disruptive update
	exists, err := m.net.Exists(id)
	if err != nil {
		return fmt.Errorf("checking interface: %w", err)
	}

	if exists {
		// Generate stripped config (no Address/DNS/MTU) for wg syncconf
		strippedConf := confgen.GenerateStrippedConf(params)
		if err := m.net.SyncConf(id, strippedConf); err != nil {
			return fmt.Errorf("syncing config: %w", err)
		}
	} else {
		// Interface doesn't exist — save conf and use wg-quick to bring it up
		fullConf := confgen.GenerateServerConf(params)
		if err := m.net.SaveConf(id, fullConf); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		if err := m.net.QuickUp(id); err != nil {
			return fmt.Errorf("bringing up interface: %w", err)
		}
		return nil
	}

	// Also save the full conf so wg-quick stays in sync
	fullConf := confgen.GenerateServerConf(params)
	_ = m.net.SaveConf(id, fullConf)

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

	// Sync the running interface so the new peer is active immediately
	if err := m.SyncInterface(peer.InterfaceID); err != nil {
		return fmt.Errorf("syncing interface after adding peer: %w", err)
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

	ifaceID := peer.InterfaceID

	if err := m.store.DeletePeer(id); err != nil {
		return fmt.Errorf("deleting peer: %w", err)
	}

	// Sync the running interface so the peer is removed immediately
	if err := m.SyncInterface(ifaceID); err != nil {
		return fmt.Errorf("syncing interface after removing peer: %w", err)
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

	// Sync the running interface so the change takes effect immediately
	if err := m.SyncInterface(peer.InterfaceID); err != nil {
		return fmt.Errorf("syncing interface after updating peer: %w", err)
	}

	return nil
}

// StartInterface brings up an interface and enables it.
func (m *Manager) StartInterface(id string) error {
	iface, err := m.store.GetInterface(id)
	if err != nil {
		return fmt.Errorf("getting interface: %w", err)
	}
	if iface == nil {
		return fmt.Errorf("interface %q not found", id)
	}

	iface.Enabled = true
	if err := m.store.UpdateInterface(iface); err != nil {
		return fmt.Errorf("updating interface: %w", err)
	}

	// Use wg-quick to properly run PostUp scripts
	if err := m.net.QuickUp(id); err != nil {
		// Fall back to netlink + wgctrl if wg-quick fails
		return m.SyncInterface(id)
	}
	return nil
}

// StopInterface brings down an interface and disables it.
func (m *Manager) StopInterface(id string) error {
	iface, err := m.store.GetInterface(id)
	if err != nil {
		return fmt.Errorf("getting interface: %w", err)
	}
	if iface == nil {
		return fmt.Errorf("interface %q not found", id)
	}

	// Use wg-quick to properly run PostDown scripts
	if err := m.net.QuickDown(id); err != nil {
		// Fall back to netlink if wg-quick fails (e.g. no conf file)
		if err2 := m.net.SetDown(id); err2 != nil {
			return fmt.Errorf("bringing down interface: %w", err2)
		}
	}

	iface.Enabled = false
	if err := m.store.UpdateInterface(iface); err != nil {
		return fmt.Errorf("updating interface: %w", err)
	}

	return nil
}

// RestartInterface brings down and then back up an interface.
func (m *Manager) RestartInterface(id string) error {
	// Use wg-quick for proper teardown/setup (PostDown/PostUp, routes, socket binding)
	_ = m.net.QuickDown(id)
	if err := m.net.QuickUp(id); err != nil {
		// Fall back to netlink + wgctrl if wg-quick fails
		return m.SyncInterface(id)
	}
	return nil
}

// InterfaceStatus holds live status for an interface.
type InterfaceStatus struct {
	Interface models.Interface `json:"interface"`
	PublicKey string           `json:"public_key"`
	Running   bool             `json:"running"`
	Peers     []PeerStatus     `json:"peers"`
}

// PeerStatus holds live status for a peer.
type PeerStatus struct {
	Peer          models.Peer `json:"peer"`
	HasPrivateKey bool        `json:"has_private_key"`
	LastHandshake time.Time   `json:"last_handshake"`
	TransferRx    int64       `json:"transfer_rx"`
	TransferTx    int64       `json:"transfer_tx"`
	Connected     bool        `json:"connected"`
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

	// Check if interface is running in the kernel
	exists, _ := m.net.Exists(interfaceID)
	status.Running = exists

	// Try to get live data from wgctrl
	device, err := m.wg.Device(interfaceID)
	if err != nil {
		// Interface might not be synced yet; return DB data only
		for _, p := range dbPeers {
			status.Peers = append(status.Peers, PeerStatus{Peer: p, HasPrivateKey: p.PrivateKeyEncrypted != ""})
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
		ps := PeerStatus{Peer: p, HasPrivateKey: p.PrivateKeyEncrypted != ""}
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

	// Imported peers don't have private keys — can't generate client config
	if peer.PrivateKeyEncrypted == "" {
		return "", fmt.Errorf("no private key available for this peer (imported peers don't have client configs)")
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

	// Determine the server endpoint for the client config
	serverHost := m.externalIP
	if iface.Endpoint != "" {
		serverHost = iface.Endpoint
	}

	// Use ClientAllowedIPs for client config; fall back to full tunnel
	clientIPs := peer.ClientAllowedIPs
	if clientIPs == "" {
		clientIPs = "0.0.0.0/0, ::/0"
	}

	// Peer DNS overrides interface DNS if set
	dns := iface.DNS
	if peer.DNS != "" {
		dns = peer.DNS
	}

	params := confgen.PeerConfParams{
		Name:            peer.Name,
		PrivateKey:      privKey,
		Address:         peer.Address,
		DNS:             dns,
		ServerPublicKey: serverPubKey,
		ServerEndpoint:  fmt.Sprintf("%s:%d", serverHost, iface.ListenPort),
		AllowedIPs:      clientIPs,
		MTU:             iface.MTU,
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

// ImportInterface creates an interface and its peers from a parsed WireGuard config.
// Peers from import get empty PrivateKeyEncrypted since we only have their public keys.
func (m *Manager) ImportInterface(parsed *confgen.ParsedConfig, id string, ifaceType string) (*models.Interface, error) {
	// Encrypt the private key from the config
	encrypted, err := m.enc.Encrypt(parsed.Interface.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("encrypting private key: %w", err)
	}

	mtu := parsed.Interface.MTU
	if mtu == 0 {
		mtu = 1420
	}

	iface := &models.Interface{
		ID:                  id,
		Type:                models.InterfaceType(ifaceType),
		ListenPort:          parsed.Interface.ListenPort,
		PrivateKeyEncrypted: encrypted,
		Address:             parsed.Interface.Address,
		DNS:                 parsed.Interface.DNS,
		MTU:                 mtu,
		Enabled:             true,
	}

	if err := m.store.CreateInterface(iface); err != nil {
		return nil, fmt.Errorf("storing interface: %w", err)
	}

	// Create each peer
	for _, pp := range parsed.Peers {
		// For imported peers, extract the first CIDR as the tunnel address
		peerAddr := pp.AllowedIPs
		if parts := strings.SplitN(pp.AllowedIPs, ",", 2); len(parts) > 1 {
			peerAddr = strings.TrimSpace(parts[0])
		}

		peer := &models.Peer{
			InterfaceID:         id,
			Name:                pp.Name,
			PublicKey:           pp.PublicKey,
			PrivateKeyEncrypted: "", // imported peers don't have private keys
			Address:             peerAddr,
			AllowedIPs:          pp.AllowedIPs,
			Endpoint:            pp.Endpoint,
			PersistentKeepalive: pp.PersistentKeepalive,
			Enabled:             true,
		}

		// Encrypt preshared key if present
		if pp.PresharedKey != "" {
			encPSK, err := m.enc.Encrypt(pp.PresharedKey)
			if err != nil {
				return nil, fmt.Errorf("encrypting preshared key: %w", err)
			}
			peer.PresharedKeyEncrypted = encPSK
		}

		if err := m.store.CreatePeer(peer); err != nil {
			return nil, fmt.Errorf("storing peer %q: %w", pp.PublicKey, err)
		}
	}

	return iface, nil
}

// detectExternalIP tries to find the machine's external IP by dialing a UDP socket.
func detectExternalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	addr := conn.LocalAddr().(*net.UDPAddr)
	return addr.IP.String()
}

// ensureHost32 converts a peer address like "10.200.0.2/24" to "10.200.0.2/32"
// so it can be used as an AllowedIP entry for that specific host.
func ensureHost32(address string) string {
	ip, _, err := net.ParseCIDR(address)
	if err != nil {
		ip = net.ParseIP(address)
		if ip == nil {
			return ""
		}
	}
	if ip.To4() != nil {
		return ip.String() + "/32"
	}
	return ip.String() + "/128"
}

// mergeAllowedIPs combines two AllowedIP lists, deduplicating by network string.
func mergeAllowedIPs(a, b []net.IPNet) []net.IPNet {
	seen := make(map[string]bool)
	var result []net.IPNet
	for _, n := range a {
		key := n.String()
		if !seen[key] {
			seen[key] = true
			result = append(result, n)
		}
	}
	for _, n := range b {
		key := n.String()
		if !seen[key] {
			seen[key] = true
			result = append(result, n)
		}
	}
	return result
}

// mergeAllowedIPStrings merges a tunnel IP into an AllowedIPs string, deduplicating.
func mergeAllowedIPStrings(tunnelIP, allowedIPs string) string {
	seen := make(map[string]bool)
	var parts []string

	// Add tunnel IP first
	seen[strings.TrimSpace(tunnelIP)] = true
	parts = append(parts, strings.TrimSpace(tunnelIP))

	// Add existing AllowedIPs
	for _, cidr := range strings.Split(allowedIPs, ",") {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" || seen[cidr] {
			continue
		}
		seen[cidr] = true
		parts = append(parts, cidr)
	}

	return strings.Join(parts, ", ")
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
