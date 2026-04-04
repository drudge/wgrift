package server

import (
	"net/http"
	"time"
)

type interfaceSummary struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	Address        string `json:"address"`
	ListenPort     int    `json:"listen_port"`
	Enabled        bool   `json:"enabled"`
	Running        bool   `json:"running"`
	PublicKey      string `json:"public_key"`
	TotalPeers     int    `json:"total_peers"`
	ConnectedPeers int    `json:"connected_peers"`
	TotalRx        int64  `json:"total_rx"`
	TotalTx        int64  `json:"total_tx"`
}

type activeConnection struct {
	InterfaceID    string `json:"interface_id"`
	PeerID         string `json:"peer_id"`
	PeerName       string `json:"peer_name"`
	PeerType       string `json:"peer_type"`
	Address        string `json:"address"`
	Endpoint       string `json:"endpoint,omitempty"`
	LastHandshake  string `json:"last_handshake"`
	ConnectedSince string `json:"connected_since,omitempty"`
	TransferRx     int64  `json:"transfer_rx"`
	TransferTx     int64  `json:"transfer_tx"`
}

type dashboardResponse struct {
	Interfaces        []interfaceSummary `json:"interfaces"`
	ActiveConnections []activeConnection `json:"active_connections"`
	TotalPeers        int                `json:"total_peers"`
	ActivePeers       int                `json:"active_peers"`
	TotalRx           int64              `json:"total_rx"`
	TotalTx           int64              `json:"total_tx"`
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	ifaces, err := s.store.ListInterfaces()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := dashboardResponse{}

	for _, iface := range ifaces {
		summary := interfaceSummary{
			ID:         iface.ID,
			Type:       string(iface.Type),
			Address:    iface.Address,
			ListenPort: iface.ListenPort,
			Enabled:    iface.Enabled,
		}

		status, err := s.manager.GetStatus(iface.ID)
		if err == nil {
			summary.PublicKey = status.PublicKey
			summary.Running = status.Running
			for _, p := range status.Peers {
				summary.TotalPeers++
				if p.Connected {
					summary.ConnectedPeers++
					conn := activeConnection{
						InterfaceID: iface.ID,
						PeerID:      p.Peer.ID,
						PeerName:    p.Peer.Name,
						PeerType:    string(p.Peer.Type),
						Address:     p.Peer.Address,
						Endpoint:    p.Endpoint,
						TransferRx:  p.TransferRx,
						TransferTx:  p.TransferTx,
					}
					if !p.LastHandshake.IsZero() {
						conn.LastHandshake = p.LastHandshake.Format(time.RFC3339)
					}
					if t := s.poller.GetConnectedSince(p.Peer.PublicKey); !t.IsZero() {
						conn.ConnectedSince = t.Format(time.RFC3339)
					} else if !p.LastHandshake.IsZero() {
						conn.ConnectedSince = p.LastHandshake.Format(time.RFC3339)
					}
					resp.ActiveConnections = append(resp.ActiveConnections, conn)
				}
				summary.TotalRx += p.TransferRx
				summary.TotalTx += p.TransferTx
			}
		}

		resp.Interfaces = append(resp.Interfaces, summary)
		resp.TotalPeers += summary.TotalPeers
		resp.ActivePeers += summary.ConnectedPeers
		resp.TotalRx += summary.TotalRx
		resp.TotalTx += summary.TotalTx
	}

	writeJSON(w, http.StatusOK, resp)
}
