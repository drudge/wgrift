package wg

import (
	"fmt"
	"hash/fnv"
	"log"
	"math/rand"
	"time"

	"github.com/drudge/wgrift/internal/crypto"
	"github.com/drudge/wgrift/internal/models"
	"github.com/drudge/wgrift/internal/store"
	"golang.org/x/crypto/bcrypt"
)

// SeedDemoData populates the database with realistic fake data for demo mode.
// It only seeds if no interfaces exist yet, so it's safe to call on every startup.
func SeedDemoData(s store.Store, enc *crypto.Encryptor) error {
	ifaces, err := s.ListInterfaces()
	if err != nil {
		return fmt.Errorf("checking existing interfaces: %w", err)
	}
	if len(ifaces) > 0 {
		return nil
	}

	log.Println("demo: seeding database with sample data...")

	if err := seedDemoAdmin(s); err != nil {
		return fmt.Errorf("seeding demo admin: %w", err)
	}

	if err := seedVandelayVPN(s, enc); err != nil {
		return fmt.Errorf("seeding Vandelay VPN: %w", err)
	}

	if err := seedVandelaySites(s, enc); err != nil {
		return fmt.Errorf("seeding Vandelay sites: %w", err)
	}

	log.Println("demo: seed complete — login with admin / admin")
	return nil
}

func seedDemoAdmin(s store.Store) error {
	count, err := s.CountUsers()
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	user := &models.User{
		Username:     "admin",
		PasswordHash: string(hash),
		DisplayName:  "Admin",
		Role:         "admin",
		IsInitial:    true,
	}
	return s.CreateUser(user)
}

type demoPeerDef struct {
	name             string
	address          string
	clientAllowedIPs string
	enabled          bool
	psk              bool
	// simulation hints
	connected bool   // initial state the mock client should match
	endpoint  string // static endpoint for history
}

// seedVandelayVPN creates the client-access VPN with realistic usage:
//
//   - George is at the office on his MacBook (connected). His iPhone is in
//     his pocket — not on VPN because he's on the office Wi-Fi.
//   - Jerry is working from home on his MacBook (connected).
//   - Elaine is out at a meeting — her MacBook disconnected when she closed
//     the lid at the coffee shop.
//   - Kramer just popped online from his apartment (connected).
//   - Newman's account is disabled (terminated).
func seedVandelayVPN(s store.Store, enc *crypto.Encryptor) error {
	privKey, _, err := crypto.GenerateKeyPair()
	if err != nil {
		return err
	}
	encPriv, err := enc.Encrypt(privKey)
	if err != nil {
		return err
	}

	iface := &models.Interface{
		ID:                  "wg0",
		Type:                models.InterfaceTypeClientAccess,
		ListenPort:          51820,
		PrivateKeyEncrypted: encPriv,
		Address:             "10.200.0.1/24",
		DNS:                 "1.1.1.1, 1.0.0.1",
		MTU:                 1420,
		Endpoint:            "vpn.vandelay.io",
		Enabled:             true,
	}
	if err := s.CreateInterface(iface); err != nil {
		return fmt.Errorf("creating wg0: %w", err)
	}

	peers := []demoPeerDef{
		// George — at the office, MacBook on VPN, iPhone not needed
		{"george-macbook", "10.200.0.2/32", "0.0.0.0/0, ::/0", true, true, true, "73.42.118.205"},
		{"george-iphone", "10.200.0.3/32", "0.0.0.0/0, ::/0", true, true, false, "73.42.118.205"},
		// Jerry — working from home, MacBook connected
		{"jerry-macbook", "10.200.0.4/32", "0.0.0.0/0, ::/0", true, true, true, "98.207.45.12"},
		// Elaine — out at a meeting, laptop closed
		{"elaine-macbook", "10.200.0.5/32", "0.0.0.0/0, ::/0", true, true, false, "174.63.221.88"},
		// Kramer — just connected from home
		{"kramer-macbook", "10.200.0.6/32", "0.0.0.0/0, ::/0", true, false, true, "24.185.93.140"},
		// Newman — terminated, account disabled
		{"newman-desktop", "10.200.0.7/32", "0.0.0.0/0, ::/0", false, false, false, ""},
	}

	for _, p := range peers {
		peerID, err := createDemoPeer(s, enc, "wg0", models.PeerTypeClient, p)
		if err != nil {
			return fmt.Errorf("creating peer %s: %w", p.name, err)
		}
		if p.enabled {
			if err := seedPeerHistory(s, "wg0", peerID, p); err != nil {
				return fmt.Errorf("seeding history for %s: %w", p.name, err)
			}
		}
	}

	return nil
}

// seedVandelaySites creates site-to-site tunnels for Vandelay locations.
// These are persistent connections — dedicated hardware that stays online.
//
//   - warehouse-nj and office-manhattan are always connected (dedicated hardware).
//   - george-home is his home office router — connected and stable.
//   - kramer-home drops occasionally (Kramer's router is unreliable).
func seedVandelaySites(s store.Store, enc *crypto.Encryptor) error {
	privKey, _, err := crypto.GenerateKeyPair()
	if err != nil {
		return err
	}
	encPriv, err := enc.Encrypt(privKey)
	if err != nil {
		return err
	}

	iface := &models.Interface{
		ID:                  "wg1",
		Type:                models.InterfaceTypeSiteToSite,
		ListenPort:          51821,
		PrivateKeyEncrypted: encPriv,
		Address:             "10.100.0.1/24",
		DNS:                 "",
		MTU:                 1420,
		Endpoint:            "hq.vandelay.io",
		Enabled:             true,
	}
	if err := s.CreateInterface(iface); err != nil {
		return fmt.Errorf("creating wg1: %w", err)
	}

	peers := []demoPeerDef{
		// Permanent sites — always connected
		{"warehouse-nj", "10.100.0.2/32", "10.100.0.2/32, 192.168.10.0/24", true, true, true, "203.45.167.22"},
		{"office-manhattan", "10.100.0.3/32", "10.100.0.3/32, 192.168.20.0/24", true, true, true, "68.132.91.44"},
		// Home offices
		{"george-home", "10.100.0.4/32", "10.100.0.4/32, 192.168.30.0/24", true, true, true, "73.42.118.205"},
		{"kramer-home", "10.100.0.5/32", "10.100.0.5/32, 192.168.40.0/24", true, true, false, "24.185.93.140"},
	}

	for _, p := range peers {
		peerID, err := createDemoPeer(s, enc, "wg1", models.PeerTypeSite, p)
		if err != nil {
			return fmt.Errorf("creating peer %s: %w", p.name, err)
		}
		if err := seedPeerHistory(s, "wg1", peerID, p); err != nil {
			return fmt.Errorf("seeding history for %s: %w", p.name, err)
		}
	}

	return nil
}

func createDemoPeer(s store.Store, enc *crypto.Encryptor, ifaceID string, peerType models.PeerType, def demoPeerDef) (string, error) {
	privKey, pubKey, err := crypto.GenerateKeyPair()
	if err != nil {
		return "", err
	}

	encPriv, err := enc.Encrypt(privKey)
	if err != nil {
		return "", err
	}

	peer := &models.Peer{
		InterfaceID:         ifaceID,
		Type:                peerType,
		Name:                def.name,
		PublicKey:           pubKey,
		PrivateKeyEncrypted: encPriv,
		Address:             def.address,
		AllowedIPs:          def.address,
		ClientAllowedIPs:    def.clientAllowedIPs,
		Enabled:             def.enabled,
	}

	if def.psk {
		pskKey, err := crypto.GeneratePresharedKey()
		if err != nil {
			return "", err
		}
		encPSK, err := enc.Encrypt(pskKey)
		if err != nil {
			return "", err
		}
		peer.PresharedKeyEncrypted = encPSK
	}

	if err := s.CreatePeer(peer); err != nil {
		return "", err
	}
	return peer.ID, nil
}

// seedPeerHistory creates realistic connection log entries going back ~24 hours.
// The final event matches the peer's current connected state so history aligns
// with what the simulation engine shows live.
func seedPeerHistory(s store.Store, ifaceID, peerID string, def demoPeerDef) error {
	h := fnv.New64a()
	h.Write([]byte(peerID))
	rng := rand.New(rand.NewSource(int64(h.Sum64())))

	now := time.Now().UTC()
	t := now.Add(-24 * time.Hour)

	connected := true
	var rxTotal, txTotal int64

	for t.Before(now) {
		event := "connected"
		if !connected {
			event = "disconnected"
		}

		if connected {
			rxTotal += int64(1_000_000 + rng.Intn(50_000_000))
			txTotal += int64(500_000 + rng.Intn(10_000_000))
		}

		entry := &models.ConnectionLog{
			PeerID:      peerID,
			InterfaceID: ifaceID,
			Event:       event,
			Endpoint:    def.endpoint,
			TransferRx:  rxTotal,
			TransferTx:  txTotal,
			RecordedAt:  t,
		}
		if err := s.CreateConnectionLog(entry); err != nil {
			return err
		}

		connected = !connected

		if connected {
			// Disconnected periods: 5-45 min
			t = t.Add(time.Duration(5+rng.Intn(40)) * time.Minute)
		} else {
			// Connected periods: 30 min to 4 hours
			t = t.Add(time.Duration(30+rng.Intn(210)) * time.Minute)
		}
	}

	// Ensure the final logged state matches the simulation's initial state
	if connected != def.connected {
		event := "connected"
		if !def.connected {
			event = "disconnected"
		}
		if def.connected {
			rxTotal += int64(1_000_000 + rng.Intn(10_000_000))
			txTotal += int64(500_000 + rng.Intn(5_000_000))
		}
		entry := &models.ConnectionLog{
			PeerID:      peerID,
			InterfaceID: ifaceID,
			Event:       event,
			Endpoint:    def.endpoint,
			TransferRx:  rxTotal,
			TransferTx:  txTotal,
			RecordedAt:  now.Add(-time.Duration(1+rng.Intn(10)) * time.Minute),
		}
		if err := s.CreateConnectionLog(entry); err != nil {
			return err
		}
	}

	return nil
}
