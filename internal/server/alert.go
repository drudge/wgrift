package server

import (
	"log"
	"strings"
	"time"

	"github.com/drudge/wgrift/internal/mail"
	"github.com/drudge/wgrift/internal/models"
)

var estLocation = func() *time.Location {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return time.FixedZone("EST", -5*60*60)
	}
	return loc
}()

func (s *Server) sendPeerAlert(peer models.Peer, iface models.Interface, event, endpoint string) {
	smtp, err := s.smtpSettings()
	if err != nil || smtp == nil {
		return
	}

	serverURL := s.cfg.Server.ExternalURL
	if u, err := s.store.GetSetting("external_url"); err == nil && u != "" {
		serverURL = u
	}

	timestamp := time.Now().In(estLocation).Format("Jan 2, 2006 3:04:05 PM MST")

	emails := strings.Split(peer.AlertEmails, ",")
	for _, to := range emails {
		to = strings.TrimSpace(to)
		if to == "" {
			continue
		}
		err := mail.SendAlertEmail(*smtp, mail.AlertEmail{
			To:            to,
			PeerName:      peer.Name,
			PublicKey:     peer.PublicKey,
			InterfaceName: iface.ID,
			Event:         event,
			Endpoint:      endpoint,
			TransferRx:    peer.TransferRx,
			TransferTx:    peer.TransferTx,
			Timestamp:     timestamp,
			ServerName:    serverURL,
		})
		if err != nil {
			log.Printf("alert: failed to send %s alert for peer %s to %s: %v", event, peer.Name, to, err)
		}
	}
}
