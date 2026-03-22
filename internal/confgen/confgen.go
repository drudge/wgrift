package confgen

import (
	"fmt"
	"strings"
)

// PeerConfParams holds all parameters needed to generate a WireGuard client config.
type PeerConfParams struct {
	// Client (peer) side
	PrivateKey string
	Address    string
	DNS        string

	// Server side
	ServerPublicKey string
	ServerEndpoint  string
	AllowedIPs      string
	PresharedKey    string
	PersistentKeepalive int
	MTU             int
}

// GeneratePeerConf produces a standard WireGuard .conf file for a peer/client.
func GeneratePeerConf(p PeerConfParams) string {
	var b strings.Builder

	b.WriteString("[Interface]\n")
	fmt.Fprintf(&b, "PrivateKey = %s\n", p.PrivateKey)
	fmt.Fprintf(&b, "Address = %s\n", p.Address)

	if p.DNS != "" {
		fmt.Fprintf(&b, "DNS = %s\n", p.DNS)
	}
	if p.MTU > 0 {
		fmt.Fprintf(&b, "MTU = %d\n", p.MTU)
	}

	b.WriteString("\n[Peer]\n")
	fmt.Fprintf(&b, "PublicKey = %s\n", p.ServerPublicKey)

	if p.PresharedKey != "" {
		fmt.Fprintf(&b, "PresharedKey = %s\n", p.PresharedKey)
	}

	fmt.Fprintf(&b, "AllowedIPs = %s\n", p.AllowedIPs)

	if p.ServerEndpoint != "" {
		fmt.Fprintf(&b, "Endpoint = %s\n", p.ServerEndpoint)
	}

	if p.PersistentKeepalive > 0 {
		fmt.Fprintf(&b, "PersistentKeepalive = %d\n", p.PersistentKeepalive)
	}

	return b.String()
}

// ServerConfParams holds parameters for generating the server-side WireGuard config.
type ServerConfParams struct {
	PrivateKey string
	Address    string
	ListenPort int
	MTU        int
	DNS        string
	Peers      []ServerPeerBlock
}

// ServerPeerBlock represents one [Peer] block in a server config.
type ServerPeerBlock struct {
	PublicKey           string
	PresharedKey        string
	AllowedIPs          string
	Endpoint            string
	PersistentKeepalive int
}

// GenerateServerConf produces a WireGuard .conf for the server/interface.
func GenerateServerConf(p ServerConfParams) string {
	var b strings.Builder

	b.WriteString("[Interface]\n")
	fmt.Fprintf(&b, "PrivateKey = %s\n", p.PrivateKey)
	fmt.Fprintf(&b, "Address = %s\n", p.Address)
	fmt.Fprintf(&b, "ListenPort = %d\n", p.ListenPort)

	if p.MTU > 0 {
		fmt.Fprintf(&b, "MTU = %d\n", p.MTU)
	}
	if p.DNS != "" {
		fmt.Fprintf(&b, "DNS = %s\n", p.DNS)
	}

	for _, peer := range p.Peers {
		b.WriteString("\n[Peer]\n")
		fmt.Fprintf(&b, "PublicKey = %s\n", peer.PublicKey)

		if peer.PresharedKey != "" {
			fmt.Fprintf(&b, "PresharedKey = %s\n", peer.PresharedKey)
		}

		fmt.Fprintf(&b, "AllowedIPs = %s\n", peer.AllowedIPs)

		if peer.Endpoint != "" {
			fmt.Fprintf(&b, "Endpoint = %s\n", peer.Endpoint)
		}

		if peer.PersistentKeepalive > 0 {
			fmt.Fprintf(&b, "PersistentKeepalive = %d\n", peer.PersistentKeepalive)
		}
	}

	return b.String()
}
