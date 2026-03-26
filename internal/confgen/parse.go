package confgen

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

// ParsedConfig holds the result of parsing a WireGuard .conf file.
type ParsedConfig struct {
	Interface ParsedInterface
	Peers     []ParsedPeer
}

// ParsedInterface holds the [Interface] section fields.
type ParsedInterface struct {
	PrivateKey string
	Address    string
	ListenPort int
	DNS        string
	MTU        int
}

// ParsedPeer holds the fields from a [Peer] section.
type ParsedPeer struct {
	Name                string // from comment above [Peer]
	PublicKey           string
	PresharedKey        string
	AllowedIPs          string
	Endpoint            string
	PersistentKeepalive int
}

// ParseConfig parses a standard WireGuard .conf file and returns structured data.
// Comment lines (# ...) immediately before a [Peer] section are captured as the peer's name.
func ParseConfig(input string) (*ParsedConfig, error) {
	cfg := &ParsedConfig{}

	scanner := bufio.NewScanner(strings.NewReader(input))

	type section int
	const (
		sectionNone section = iota
		sectionInterface
		sectionPeer
	)

	current := sectionNone
	var lastComment string
	var peerIndex int = -1

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Track comment lines for peer naming
		if strings.HasPrefix(line, "#") {
			comment := strings.TrimSpace(strings.TrimPrefix(line, "#"))
			lastComment = comment
			// If we're inside a [Peer] section and the peer has no name yet, use this comment
			if current == sectionPeer && peerIndex >= 0 && cfg.Peers[peerIndex].Name == "" {
				cfg.Peers[peerIndex].Name = comment
			}
			continue
		}

		// Empty lines reset the comment tracker unless we're about to hit a section header
		if line == "" {
			// Don't reset lastComment - it might precede the next [Peer]
			continue
		}

		// Section headers
		if strings.EqualFold(line, "[interface]") {
			current = sectionInterface
			lastComment = ""
			continue
		}
		if strings.EqualFold(line, "[peer]") {
			current = sectionPeer
			peer := ParsedPeer{}
			if lastComment != "" {
				peer.Name = lastComment
			}
			cfg.Peers = append(cfg.Peers, peer)
			peerIndex = len(cfg.Peers) - 1
			lastComment = ""
			continue
		}

		// Reset comment if we encounter a non-comment, non-empty, non-section line
		lastComment = ""

		// Parse key = value
		key, value, ok := parseKeyValue(line)
		if !ok {
			continue
		}

		switch current {
		case sectionInterface:
			switch strings.ToLower(key) {
			case "privatekey":
				cfg.Interface.PrivateKey = value
			case "address":
				cfg.Interface.Address = value
			case "listenport":
				port, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("invalid ListenPort %q: %w", value, err)
				}
				cfg.Interface.ListenPort = port
			case "dns":
				cfg.Interface.DNS = value
			case "mtu":
				mtu, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("invalid MTU %q: %w", value, err)
				}
				cfg.Interface.MTU = mtu
			}
		case sectionPeer:
			if peerIndex < 0 {
				continue
			}
			switch strings.ToLower(key) {
			case "publickey":
				cfg.Peers[peerIndex].PublicKey = value
			case "presharedkey":
				cfg.Peers[peerIndex].PresharedKey = value
			case "allowedips":
				cfg.Peers[peerIndex].AllowedIPs = value
			case "endpoint":
				cfg.Peers[peerIndex].Endpoint = value
			case "persistentkeepalive":
				ka, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("invalid PersistentKeepalive %q: %w", value, err)
				}
				cfg.Peers[peerIndex].PersistentKeepalive = ka
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if cfg.Interface.PrivateKey == "" {
		return nil, fmt.Errorf("missing PrivateKey in [Interface] section")
	}

	return cfg, nil
}

// parseKeyValue splits "Key = Value" into (key, value, true) or returns ("", "", false).
func parseKeyValue(line string) (string, string, bool) {
	idx := strings.Index(line, "=")
	if idx < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])
	if key == "" {
		return "", "", false
	}
	return key, value, true
}
