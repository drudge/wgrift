package confgen

import (
	"testing"
)

const fullConfig = `[Interface]
PrivateKey = yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk=
Address = 10.100.0.1/24
ListenPort = 51820
DNS = 1.1.1.1, 8.8.8.8
MTU = 1420

# Alice's laptop
[Peer]
PublicKey = xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=
AllowedIPs = 10.100.0.2/32
PersistentKeepalive = 25

# Bob's phone
[Peer]
PublicKey = TrMvSoP4jYQlY6RIzBgbssQqY3vxI2piVFBs2LTlA6s=
PresharedKey = AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=
AllowedIPs = 10.100.0.3/32
Endpoint = 192.168.1.100:51820
PersistentKeepalive = 15
`

func TestParseConfig_Full(t *testing.T) {
	cfg, err := ParseConfig(fullConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Interface checks
	if cfg.Interface.PrivateKey != "yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk=" {
		t.Errorf("PrivateKey = %q", cfg.Interface.PrivateKey)
	}
	if cfg.Interface.Address != "10.100.0.1/24" {
		t.Errorf("Address = %q", cfg.Interface.Address)
	}
	if cfg.Interface.ListenPort != 51820 {
		t.Errorf("ListenPort = %d", cfg.Interface.ListenPort)
	}
	if cfg.Interface.DNS != "1.1.1.1, 8.8.8.8" {
		t.Errorf("DNS = %q", cfg.Interface.DNS)
	}
	if cfg.Interface.MTU != 1420 {
		t.Errorf("MTU = %d", cfg.Interface.MTU)
	}

	// Peer checks
	if len(cfg.Peers) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(cfg.Peers))
	}

	// Peer 0 - Alice
	p0 := cfg.Peers[0]
	if p0.Name != "Alice's laptop" {
		t.Errorf("peer[0].Name = %q", p0.Name)
	}
	if p0.PublicKey != "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=" {
		t.Errorf("peer[0].PublicKey = %q", p0.PublicKey)
	}
	if p0.AllowedIPs != "10.100.0.2/32" {
		t.Errorf("peer[0].AllowedIPs = %q", p0.AllowedIPs)
	}
	if p0.PersistentKeepalive != 25 {
		t.Errorf("peer[0].PersistentKeepalive = %d", p0.PersistentKeepalive)
	}
	if p0.PresharedKey != "" {
		t.Errorf("peer[0].PresharedKey should be empty, got %q", p0.PresharedKey)
	}

	// Peer 1 - Bob
	p1 := cfg.Peers[1]
	if p1.Name != "Bob's phone" {
		t.Errorf("peer[1].Name = %q", p1.Name)
	}
	if p1.PublicKey != "TrMvSoP4jYQlY6RIzBgbssQqY3vxI2piVFBs2LTlA6s=" {
		t.Errorf("peer[1].PublicKey = %q", p1.PublicKey)
	}
	if p1.PresharedKey != "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=" {
		t.Errorf("peer[1].PresharedKey = %q", p1.PresharedKey)
	}
	if p1.AllowedIPs != "10.100.0.3/32" {
		t.Errorf("peer[1].AllowedIPs = %q", p1.AllowedIPs)
	}
	if p1.Endpoint != "192.168.1.100:51820" {
		t.Errorf("peer[1].Endpoint = %q", p1.Endpoint)
	}
	if p1.PersistentKeepalive != 15 {
		t.Errorf("peer[1].PersistentKeepalive = %d", p1.PersistentKeepalive)
	}
}

func TestParseConfig_MinimalInterface(t *testing.T) {
	input := `[Interface]
PrivateKey = somekey123=
Address = 10.0.0.1/24
`
	cfg, err := ParseConfig(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Interface.PrivateKey != "somekey123=" {
		t.Errorf("PrivateKey = %q", cfg.Interface.PrivateKey)
	}
	if cfg.Interface.Address != "10.0.0.1/24" {
		t.Errorf("Address = %q", cfg.Interface.Address)
	}
	if cfg.Interface.ListenPort != 0 {
		t.Errorf("ListenPort should be 0, got %d", cfg.Interface.ListenPort)
	}
	if cfg.Interface.MTU != 0 {
		t.Errorf("MTU should be 0, got %d", cfg.Interface.MTU)
	}
	if len(cfg.Peers) != 0 {
		t.Errorf("expected 0 peers, got %d", len(cfg.Peers))
	}
}

func TestParseConfig_MissingPrivateKey(t *testing.T) {
	input := `[Interface]
Address = 10.0.0.1/24

[Peer]
PublicKey = abc123=
AllowedIPs = 10.0.0.2/32
`
	_, err := ParseConfig(input)
	if err == nil {
		t.Fatal("expected error for missing PrivateKey")
	}
}

func TestParseConfig_PeerWithoutComment(t *testing.T) {
	input := `[Interface]
PrivateKey = key123=

[Peer]
PublicKey = peer1=
AllowedIPs = 10.0.0.2/32
`
	cfg, err := ParseConfig(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(cfg.Peers))
	}
	if cfg.Peers[0].Name != "" {
		t.Errorf("peer name should be empty, got %q", cfg.Peers[0].Name)
	}
}

func TestParseConfig_CaseInsensitiveHeaders(t *testing.T) {
	input := `[interface]
PrivateKey = key123=
Address = 10.0.0.1/24

[peer]
PublicKey = peer1=
AllowedIPs = 10.0.0.2/32
`
	cfg, err := ParseConfig(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Interface.PrivateKey != "key123=" {
		t.Errorf("PrivateKey = %q", cfg.Interface.PrivateKey)
	}
	if len(cfg.Peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(cfg.Peers))
	}
}

func TestParseConfig_InvalidListenPort(t *testing.T) {
	input := `[Interface]
PrivateKey = key123=
ListenPort = notanumber
`
	_, err := ParseConfig(input)
	if err == nil {
		t.Fatal("expected error for invalid ListenPort")
	}
}

func TestParseConfig_InvalidMTU(t *testing.T) {
	input := `[Interface]
PrivateKey = key123=
MTU = abc
`
	_, err := ParseConfig(input)
	if err == nil {
		t.Fatal("expected error for invalid MTU")
	}
}

func TestParseConfig_InvalidPersistentKeepalive(t *testing.T) {
	input := `[Interface]
PrivateKey = key123=

[Peer]
PublicKey = peer1=
PersistentKeepalive = xyz
`
	_, err := ParseConfig(input)
	if err == nil {
		t.Fatal("expected error for invalid PersistentKeepalive")
	}
}

func TestParseConfig_MultiplePeers(t *testing.T) {
	input := `[Interface]
PrivateKey = key123=
Address = 10.0.0.1/24
ListenPort = 51820

# Peer Alpha
[Peer]
PublicKey = alpha=
AllowedIPs = 10.0.0.2/32

# Peer Beta
[Peer]
PublicKey = beta=
AllowedIPs = 10.0.0.3/32

[Peer]
PublicKey = gamma=
AllowedIPs = 10.0.0.4/32
`
	cfg, err := ParseConfig(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Peers) != 3 {
		t.Fatalf("expected 3 peers, got %d", len(cfg.Peers))
	}

	if cfg.Peers[0].Name != "Peer Alpha" {
		t.Errorf("peer[0].Name = %q", cfg.Peers[0].Name)
	}
	if cfg.Peers[1].Name != "Peer Beta" {
		t.Errorf("peer[1].Name = %q", cfg.Peers[1].Name)
	}
	if cfg.Peers[2].Name != "" {
		t.Errorf("peer[2].Name should be empty, got %q", cfg.Peers[2].Name)
	}
	if cfg.Peers[0].PublicKey != "alpha=" {
		t.Errorf("peer[0].PublicKey = %q", cfg.Peers[0].PublicKey)
	}
	if cfg.Peers[1].PublicKey != "beta=" {
		t.Errorf("peer[1].PublicKey = %q", cfg.Peers[1].PublicKey)
	}
	if cfg.Peers[2].PublicKey != "gamma=" {
		t.Errorf("peer[2].PublicKey = %q", cfg.Peers[2].PublicKey)
	}
}

func TestParseConfig_CommentAfterPeerHeader(t *testing.T) {
	input := `[Interface]
PrivateKey = key123=
Address = 10.200.0.1/30
ListenPort = 51820

[Peer]
# UDM Pro
PublicKey = PFcPPyG9mVwkXSAxY9l+BDI195VaXnfkQwqX2gaOACU=
AllowedIPs = 10.200.0.2/32, 10.0.7.0/24, 10.0.6.0/24, 10.0.69.0/29

[Peer]
# Phone
PublicKey = abc123=
AllowedIPs = 10.200.0.3/32
`
	cfg, err := ParseConfig(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Peers) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(cfg.Peers))
	}
	if cfg.Peers[0].Name != "UDM Pro" {
		t.Errorf("peer[0].Name = %q, want %q", cfg.Peers[0].Name, "UDM Pro")
	}
	if cfg.Peers[1].Name != "Phone" {
		t.Errorf("peer[1].Name = %q, want %q", cfg.Peers[1].Name, "Phone")
	}
}

func TestParseConfig_PostUpDown(t *testing.T) {
	input := `[Interface]
PrivateKey = key123=
Address = 10.0.0.1/24
PostUp = iptables -A FORWARD -i %i -j ACCEPT
PostUp = iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i %i -j ACCEPT
PostDown = iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE
`
	cfg, err := ParseConfig(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Interface.PostUp) != 2 {
		t.Fatalf("expected 2 PostUp commands, got %d", len(cfg.Interface.PostUp))
	}
	if cfg.Interface.PostUp[0] != "iptables -A FORWARD -i %i -j ACCEPT" {
		t.Errorf("PostUp[0] = %q", cfg.Interface.PostUp[0])
	}
	if cfg.Interface.PostUp[1] != "iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE" {
		t.Errorf("PostUp[1] = %q", cfg.Interface.PostUp[1])
	}

	if len(cfg.Interface.PostDown) != 2 {
		t.Fatalf("expected 2 PostDown commands, got %d", len(cfg.Interface.PostDown))
	}
	if cfg.Interface.PostDown[0] != "iptables -D FORWARD -i %i -j ACCEPT" {
		t.Errorf("PostDown[0] = %q", cfg.Interface.PostDown[0])
	}
	if cfg.Interface.PostDown[1] != "iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE" {
		t.Errorf("PostDown[1] = %q", cfg.Interface.PostDown[1])
	}
}

func TestParseConfig_NoPostUpDown(t *testing.T) {
	input := `[Interface]
PrivateKey = key123=
Address = 10.0.0.1/24
`
	cfg, err := ParseConfig(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Interface.PostUp) != 0 {
		t.Errorf("expected 0 PostUp commands, got %d", len(cfg.Interface.PostUp))
	}
	if len(cfg.Interface.PostDown) != 0 {
		t.Errorf("expected 0 PostDown commands, got %d", len(cfg.Interface.PostDown))
	}
}

func TestParseConfig_CommentNotDirectlyBeforePeer(t *testing.T) {
	input := `[Interface]
PrivateKey = key123=
# This is just a comment in the interface section

[Peer]
PublicKey = peer1=
AllowedIPs = 10.0.0.2/32
`
	cfg, err := ParseConfig(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The comment is before [Peer] with no non-empty/non-comment lines in between,
	// so it should be captured as the name.
	if len(cfg.Peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(cfg.Peers))
	}
}
