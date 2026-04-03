package wg

import (
	"hash/fnv"
	"math/rand"
	"net"
	"sync"
	"time"

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

		state := d.getOrCreateState(p.PublicKey, now)
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

func (d *demoWGClient) getOrCreateState(pubKey string, now time.Time) *demoPeerState {
	if state, ok := d.peerStates[pubKey]; ok {
		return state
	}

	// Deterministic seed from public key for consistent initial state
	h := fnv.New64a()
	h.Write([]byte(pubKey))
	rng := rand.New(rand.NewSource(int64(h.Sum64())))

	// ~70% chance of starting connected
	connected := rng.Float64() < 0.70

	var lastHandshake time.Time
	if connected {
		lastHandshake = now.Add(-time.Duration(rng.Intn(120)) * time.Second)
	}

	// Generate a plausible public IP endpoint
	ep := net.UDPAddr{
		IP:   demoEndpointIP(rng),
		Port: 1024 + rng.Intn(64000),
	}

	// Start with some accumulated transfer data
	rxBytes := int64(rng.Intn(500_000_000)) + 1_000_000 // 1MB - 500MB
	txBytes := int64(rng.Intn(100_000_000)) + 500_000   // 500KB - 100MB

	state := &demoPeerState{
		connected:     connected,
		lastHandshake: lastHandshake,
		rxBytes:       rxBytes,
		txBytes:       txBytes,
		endpoint:      ep,
		nextFlipAfter: now.Add(time.Duration(30+rng.Intn(270)) * time.Second),
	}

	d.peerStates[pubKey] = state
	return state
}

func (d *demoWGClient) evolveState(state *demoPeerState, now time.Time) {
	rng := rand.New(rand.NewSource(now.UnixNano()))

	// Check for state transition
	if now.After(state.nextFlipAfter) {
		if state.connected {
			// ~30% chance of disconnecting when the flip timer fires
			if rng.Float64() < 0.30 {
				state.connected = false
			}
		} else {
			// ~60% chance of reconnecting when the flip timer fires
			if rng.Float64() < 0.60 {
				state.connected = true
			}
		}
		// Schedule next potential flip: 30s to 5min
		state.nextFlipAfter = now.Add(time.Duration(30+rng.Intn(270)) * time.Second)
	}

	if state.connected {
		// Increment transfer data
		state.rxBytes += int64(50_000 + rng.Intn(2_000_000)) // 50KB - 2MB
		state.txBytes += int64(10_000 + rng.Intn(500_000))   // 10KB - 500KB
		// Recent handshake (within 180s threshold)
		state.lastHandshake = now.Add(-time.Duration(rng.Intn(120)) * time.Second)
	}
	// Disconnected peers keep their stale lastHandshake (will exceed 180s threshold naturally)
}

// demoEndpointIP generates a plausible-looking public IP address.
func demoEndpointIP(rng *rand.Rand) net.IP {
	// Use various public IP ranges that look realistic
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
