package store

import (
	"testing"

	"github.com/drudge/wgrift/internal/models"
)

func testStore(t *testing.T) *SQLiteStore {
	t.Helper()
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestInterfaceCRUD(t *testing.T) {
	s := testStore(t)

	iface := &models.Interface{
		ID:                  "wg0",
		Type:                models.InterfaceTypeClientAccess,
		ListenPort:          51820,
		PrivateKeyEncrypted: "encrypted-key-data",
		Address:             "10.100.0.1/24",
		DNS:                 "1.1.1.1",
		MTU:                 1420,
		Enabled:             true,
	}

	// Create
	if err := s.CreateInterface(iface); err != nil {
		t.Fatalf("CreateInterface: %v", err)
	}

	// Get
	got, err := s.GetInterface("wg0")
	if err != nil {
		t.Fatalf("GetInterface: %v", err)
	}
	if got == nil {
		t.Fatal("GetInterface returned nil")
	}
	if got.ID != "wg0" || got.ListenPort != 51820 || got.Address != "10.100.0.1/24" {
		t.Fatalf("unexpected interface: %+v", got)
	}

	// List
	ifaces, err := s.ListInterfaces()
	if err != nil {
		t.Fatalf("ListInterfaces: %v", err)
	}
	if len(ifaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(ifaces))
	}

	// Update
	got.DNS = "8.8.8.8"
	if err := s.UpdateInterface(got); err != nil {
		t.Fatalf("UpdateInterface: %v", err)
	}
	updated, _ := s.GetInterface("wg0")
	if updated.DNS != "8.8.8.8" {
		t.Fatalf("DNS not updated: %q", updated.DNS)
	}

	// Get nonexistent
	missing, err := s.GetInterface("wg99")
	if err != nil {
		t.Fatalf("GetInterface for missing: %v", err)
	}
	if missing != nil {
		t.Fatal("expected nil for missing interface")
	}

	// Delete
	if err := s.DeleteInterface("wg0"); err != nil {
		t.Fatalf("DeleteInterface: %v", err)
	}
	deleted, _ := s.GetInterface("wg0")
	if deleted != nil {
		t.Fatal("interface should be deleted")
	}
}

func TestPeerCRUD(t *testing.T) {
	s := testStore(t)

	// Create interface first
	iface := &models.Interface{
		ID:                  "wg0",
		Type:                models.InterfaceTypeClientAccess,
		ListenPort:          51820,
		PrivateKeyEncrypted: "encrypted-key",
		Address:             "10.100.0.1/24",
		MTU:                 1420,
		Enabled:             true,
	}
	if err := s.CreateInterface(iface); err != nil {
		t.Fatalf("CreateInterface: %v", err)
	}

	peer := &models.Peer{
		ID:                  "test-peer-id",
		InterfaceID:         "wg0",
		Type:                models.PeerTypeClient,
		Name:                "Test Peer",
		PublicKey:           "aGVsbG8gd29ybGQgdGhpcyBpcyBhIHRlc3Qga2V5",
		PrivateKeyEncrypted: "encrypted-peer-key",
		AllowedIPs:          "10.100.0.2/32",
		PersistentKeepalive: 25,
		Enabled:             true,
	}

	// Create
	if err := s.CreatePeer(peer); err != nil {
		t.Fatalf("CreatePeer: %v", err)
	}

	// Get
	got, err := s.GetPeer("test-peer-id")
	if err != nil {
		t.Fatalf("GetPeer: %v", err)
	}
	if got == nil {
		t.Fatal("GetPeer returned nil")
	}
	if got.Name != "Test Peer" || got.AllowedIPs != "10.100.0.2/32" {
		t.Fatalf("unexpected peer: %+v", got)
	}

	// List by interface
	peers, err := s.ListPeers("wg0")
	if err != nil {
		t.Fatalf("ListPeers: %v", err)
	}
	if len(peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(peers))
	}

	// List all
	allPeers, err := s.ListAllPeers()
	if err != nil {
		t.Fatalf("ListAllPeers: %v", err)
	}
	if len(allPeers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(allPeers))
	}

	// Update
	got.Name = "Updated Peer"
	got.Enabled = false
	if err := s.UpdatePeer(got); err != nil {
		t.Fatalf("UpdatePeer: %v", err)
	}
	updated, _ := s.GetPeer("test-peer-id")
	if updated.Name != "Updated Peer" || updated.Enabled {
		t.Fatalf("peer not updated: %+v", updated)
	}

	// Delete
	if err := s.DeletePeer("test-peer-id"); err != nil {
		t.Fatalf("DeletePeer: %v", err)
	}
	deleted, _ := s.GetPeer("test-peer-id")
	if deleted != nil {
		t.Fatal("peer should be deleted")
	}
}

func TestCascadeDelete(t *testing.T) {
	s := testStore(t)

	iface := &models.Interface{
		ID:                  "wg0",
		Type:                models.InterfaceTypeClientAccess,
		ListenPort:          51820,
		PrivateKeyEncrypted: "key",
		Address:             "10.0.0.1/24",
		MTU:                 1420,
		Enabled:             true,
	}
	s.CreateInterface(iface)

	peer := &models.Peer{
		ID:                  "peer-1",
		InterfaceID:         "wg0",
		Name:                "Peer 1",
		PublicKey:           "dW5pcXVlLXB1YmxpYy1rZXktZm9yLXRlc3Q=",
		PrivateKeyEncrypted: "key",
		AllowedIPs:          "10.0.0.2/32",
		Enabled:             true,
	}
	s.CreatePeer(peer)

	// Delete interface should cascade to peers
	s.DeleteInterface("wg0")

	got, _ := s.GetPeer("peer-1")
	if got != nil {
		t.Fatal("peer should be cascade-deleted with interface")
	}
}
