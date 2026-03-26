package server

import (
	"context"
	"log"
	"time"

	"github.com/drudge/wgrift/internal/config"
	"github.com/drudge/wgrift/internal/models"
	"github.com/drudge/wgrift/internal/store"
	"github.com/drudge/wgrift/internal/wg"
)

// Poller polls WireGuard interface status and logs connection events.
type Poller struct {
	manager  *wg.Manager
	store    store.Store
	interval time.Duration
	retention time.Duration
}

type peerSnapshot struct {
	Connected  bool
	TransferRx int64
	TransferTx int64
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
	}
}

// Run starts the polling loop. Blocks until ctx is cancelled.
func (p *Poller) Run(ctx context.Context) {
	lastState := make(map[string]peerSnapshot)
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	cleanupTicker := time.NewTicker(1 * time.Hour)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.poll(lastState)
		case <-cleanupTicker.C:
			cutoff := time.Now().UTC().Add(-p.retention)
			if err := p.store.DeleteOldConnectionLogs(cutoff); err != nil {
				log.Printf("poller: cleanup error: %v", err)
			}
		}
	}
}

func (p *Poller) poll(lastState map[string]peerSnapshot) {
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
			prev, exists := lastState[key]

			current := peerSnapshot{
				Connected:  ps.Connected,
				TransferRx: ps.TransferRx,
				TransferTx: ps.TransferTx,
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
					TransferRx:  current.TransferRx,
					TransferTx:  current.TransferTx,
				}

				if err := p.store.CreateConnectionLog(logEntry); err != nil {
					log.Printf("poller: log connection event: %v", err)
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

			lastState[key] = current
		}
	}
}
