package wg

import (
	"hash/fnv"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/drudge/wgrift/internal/models"
	"github.com/drudge/wgrift/internal/store"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// demoWGClient simulates a WireGuard kernel device for demo/development mode.
// It maintains per-peer state that evolves over time to produce realistic
// connection activity visible in the dashboard and connection logs.
type demoWGClient struct {
	store store.Store

	mu         sync.Mutex
	peerStates map[string]*demoPeerState
}

type demoPeerState struct {
	connected     bool
	persistent    bool // site-to-site peers rarely disconnect
	lastHandshake time.Time
	rxBytes       int64
	txBytes       int64
	endpoint      net.UDPAddr
	nextFlipAfter time.Time
}

// NewDemoWGClient creates a simulated WireGuard client backed by store data.
func NewDemoWGClient(s store.Store) WGClient {
	return &demoWGClient{
		store:      s,
		peerStates: make(map[string]*demoPeerState),
	}
}

func (d *demoWGClient) Device(name string) (*wgtypes.Device, error) {
	peers, err := d.store.ListPeers(name)
	if err != nil {
		return nil, err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	var wgPeers []wgtypes.Peer

	for _, p := range peers {
		if !p.Enabled {
			continue
		}

		key, err := wgtypes.ParseKey(p.PublicKey)
		if err != nil {
			continue
		}

		state := d.getOrCreateState(p, now)
		d.evolveState(state, now)

		wp := wgtypes.Peer{
			PublicKey:         key,
			LastHandshakeTime: state.lastHandshake,
			ReceiveBytes:      state.rxBytes,
			TransmitBytes:     state.txBytes,
			Endpoint:          &state.endpoint,
		}

		wgPeers = append(wgPeers, wp)
	}

	return &wgtypes.Device{
		Name:  name,
		Peers: wgPeers,
	}, nil
}

func (d *demoWGClient) Close() error { return nil }

func (d *demoWGClient) getOrCreateState(peer models.Peer, now time.Time) *demoPeerState {
	if state, ok := d.peerStates[peer.PublicKey]; ok {
		return state
	}

	// Deterministic seed from public key for consistent state across restarts
	h := fnv.New64a()
	h.Write([]byte(peer.PublicKey))
	rng := rand.New(rand.NewSource(int64(h.Sum64())))

	// Determine initial connected state from the peer's seeded data.
	// If the peer has a recent LastHandshake in the DB, start connected.
	// Otherwise start disconnected. This aligns with the seed data.
	connected := false
	if peer.LastHandshake != nil && time.Since(*peer.LastHandshake) < 180*time.Second {
		connected = true
	}

	// Site-to-site peers are persistent infrastructure — very stable
	persistent := peer.Type == models.PeerTypeSite

	var lastHandshake time.Time
	if connected {
		lastHandshake = now.Add(-time.Duration(rng.Intn(60)) * time.Second)
	}

	// Use the peer's seeded endpoint if available, otherwise generate one
	ep := net.UDPAddr{Port: 1024 + rng.Intn(64000)}
	if peer.Endpoint != "" {
		ep.IP = net.ParseIP(peer.Endpoint)
	}
	if ep.IP == nil {
		ep.IP = demoEndpointIP(rng)
	}

	// Start with accumulated transfer data based on peer type
	var rxBytes, txBytes int64
	if persistent {
		// Site-to-site: high baseline (always-on links)
		rxBytes = int64(rng.Intn(2_000_000_000)) + 500_000_000 // 500MB - 2.5GB
		txBytes = int64(rng.Intn(500_000_000)) + 100_000_000   // 100MB - 600MB
	} else {
		// Client: moderate baseline
		rxBytes = int64(rng.Intn(500_000_000)) + 1_000_000 // 1MB - 500MB
		txBytes = int64(rng.Intn(100_000_000)) + 500_000   // 500KB - 100MB
	}

	// First state flip further out for site-to-site
	var flipDelay int
	if persistent {
		flipDelay = 300 + rng.Intn(600) // 5-15 min before first potential flip
	} else {
		flipDelay = 30 + rng.Intn(270) // 30s-5min
	}

	state := &demoPeerState{
		connected:     connected,
		persistent:    persistent,
		lastHandshake: lastHandshake,
		rxBytes:       rxBytes,
		txBytes:       txBytes,
		endpoint:      ep,
		nextFlipAfter: now.Add(time.Duration(flipDelay) * time.Second),
	}

	d.peerStates[peer.PublicKey] = state
	return state
}

func (d *demoWGClient) evolveState(state *demoPeerState, now time.Time) {
	rng := rand.New(rand.NewSource(now.UnixNano()))

	if now.After(state.nextFlipAfter) {
		if state.persistent {
			// Site-to-site: very rarely disconnects (~5%), reconnects quickly (~90%)
			if state.connected {
				if rng.Float64() < 0.05 {
					state.connected = false
				}
			} else {
				if rng.Float64() < 0.90 {
					state.connected = true
				}
			}
			// Longer intervals between checks for persistent peers
			state.nextFlipAfter = now.Add(time.Duration(120+rng.Intn(480)) * time.Second) // 2-10 min
		} else {
			// Client peers: moderate churn
			if state.connected {
				if rng.Float64() < 0.25 {
					state.connected = false
				}
			} else {
				if rng.Float64() < 0.40 {
					state.connected = true
				}
			}
			state.nextFlipAfter = now.Add(time.Duration(30+rng.Intn(270)) * time.Second) // 30s-5min
		}
	}

	if state.connected {
		if state.persistent {
			// Site-to-site links: steady, higher throughput
			state.rxBytes += int64(100_000 + rng.Intn(5_000_000)) // 100KB - 5MB
			state.txBytes += int64(50_000 + rng.Intn(2_000_000))  // 50KB - 2MB
		} else {
			// Client VPN: bursty, lower throughput
			state.rxBytes += int64(50_000 + rng.Intn(2_000_000)) // 50KB - 2MB
			state.txBytes += int64(10_000 + rng.Intn(500_000))   // 10KB - 500KB
		}
		state.lastHandshake = now.Add(-time.Duration(rng.Intn(120)) * time.Second)
	}
}

// demoEndpointIP generates a plausible-looking public IP address.
func demoEndpointIP(rng *rand.Rand) net.IP {
	prefixes := []struct {
		a, b byte
	}{
		{24, 0},  // cable/dsl
		{47, 0},  // various ISPs
		{68, 0},  // comcast-ish
		{73, 0},  // comcast
		{76, 0},  // time warner
		{98, 0},  // charter
		{108, 0}, // at&t
		{174, 0}, // cogent
		{184, 0}, // various
		{203, 0}, // apnic
	}

	prefix := prefixes[rng.Intn(len(prefixes))]
	return net.IPv4(prefix.a, byte(rng.Intn(256)), byte(1+rng.Intn(254)), byte(1+rng.Intn(254)))
}

// isPersistentPeer checks if a peer name suggests a persistent site-to-site connection.
// Used as a fallback when peer type isn't available.
func isPersistentPeer(name string) bool {
	persistent := []string{"warehouse", "office", "datacenter", "server", "router", "gateway"}
	lower := strings.ToLower(name)
	for _, p := range persistent {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}
