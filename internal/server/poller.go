package server

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/drudge/wgrift/internal/config"
	"github.com/drudge/wgrift/internal/models"
	"github.com/drudge/wgrift/internal/store"
	"github.com/drudge/wgrift/internal/wg"
)

// Poller polls WireGuard interface status and logs connection events.
type Poller struct {
	manager   *wg.Manager
	store     store.Store
	interval  time.Duration
	retention time.Duration

	mu        sync.Mutex
	lastState map[string]peerSnapshot
	notify    chan struct{}

	alertFn func(peer models.Peer, iface models.Interface, event, endpoint string)
}

type peerSnapshot struct {
	Connected  bool
	TransferRx int64
	TransferTx int64
	Endpoint   string
}

// NewPoller creates a connection status poller.
func NewPoller(mgr *wg.Manager, s store.Store, cfg config.Config) *Poller {
	interval := 30 * time.Second
	if d, err := time.ParseDuration(cfg.Logging.ConnectionPollInterval); err == nil {
		interval = d
	}

	retention := time.Duration(cfg.Logging.RetentionDays) * 24 * time.Hour

	return &Poller{
		manager:   mgr,
		store:     s,
		interval:  interval,
		retention: retention,
		lastState: make(map[string]peerSnapshot),
		notify:    make(chan struct{}, 1),
	}
}

// Kick triggers an immediate poll cycle. Non-blocking.
func (p *Poller) Kick() {
	select {
	case p.notify <- struct{}{}:
	default:
	}
}

// Run starts the polling loop. Blocks until ctx is cancelled.
func (p *Poller) Run(ctx context.Context) {
	// Seed initial state so the first real transition is detected.
	p.poll()

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	cleanupTicker := time.NewTicker(1 * time.Hour)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.poll()
		case <-p.notify:
			p.poll()
		case <-cleanupTicker.C:
			cutoff := time.Now().UTC().Add(-p.retention)
			if err := p.store.DeleteOldConnectionLogs(cutoff); err != nil {
				log.Printf("poller: cleanup error: %v", err)
			}
		}
	}
}

func (p *Poller) poll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	ifaces, err := p.store.ListInterfaces()
	if err != nil {
		log.Printf("poller: list interfaces: %v", err)
		return
	}

	for _, iface := range ifaces {
		if !iface.Enabled {
			continue
		}

		status, err := p.manager.GetStatus(iface.ID)
		if err != nil {
			continue
		}

		for _, ps := range status.Peers {
			key := ps.Peer.PublicKey
			prev, exists := p.lastState[key]

			current := peerSnapshot{
				Connected:  ps.Connected,
				TransferRx: ps.TransferRx,
				TransferTx: ps.TransferTx,
				Endpoint:   ps.Endpoint,
			}

			// Log state transitions
			if exists && prev.Connected != current.Connected {
				event := "disconnected"
				if current.Connected {
					event = "connected"
				}

				logEntry := &models.ConnectionLog{
					PeerID:      ps.Peer.ID,
					InterfaceID: iface.ID,
					Event:       event,
					Endpoint:    current.Endpoint,
					TransferRx:  current.TransferRx,
					TransferTx:  current.TransferTx,
				}

				if err := p.store.CreateConnectionLog(logEntry); err != nil {
					log.Printf("poller: log connection event: %v", err)
				}

				if p.alertFn != nil {
					shouldAlert := (event == "connected" && ps.Peer.AlertOnConnect) ||
						(event == "disconnected" && ps.Peer.AlertOnDisconnect)
					if shouldAlert && ps.Peer.AlertEmails != "" {
						peer := ps.Peer
						peer.TransferRx = current.TransferRx
						peer.TransferTx = current.TransferTx
						ifaceCopy := iface
						go p.alertFn(peer, ifaceCopy, event, current.Endpoint)
					}
				}
			}

			// Update peer stats in database
			if current.TransferRx != prev.TransferRx || current.TransferTx != prev.TransferTx {
				peer := ps.Peer
				if !ps.LastHandshake.IsZero() {
					peer.LastHandshake = &ps.LastHandshake
				}
				peer.TransferRx = current.TransferRx
				peer.TransferTx = current.TransferTx
				p.store.UpdatePeer(&peer)
			}

			p.lastState[key] = current
		}
	}
}
