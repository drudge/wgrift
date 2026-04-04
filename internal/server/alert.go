package server

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/drudge/wgrift/internal/mail"
	"github.com/drudge/wgrift/internal/models"
)

func formatDuration(d time.Duration) string {
	totalSecs := int64(d.Seconds())
	if totalSecs <= 0 {
		return "0s"
	}
	days := totalSecs / 86400
	hours := (totalSecs % 86400) / 3600
	mins := (totalSecs % 3600) / 60
	secs := totalSecs % 60
	switch {
	case days > 0 && hours > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case days > 0:
		return fmt.Sprintf("%dd", days)
	case hours > 0 && mins > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	case hours > 0:
		return fmt.Sprintf("%dh", hours)
	case mins > 0 && secs > 0:
		return fmt.Sprintf("%dm %ds", mins, secs)
	case mins > 0:
		return fmt.Sprintf("%dm", mins)
	default:
		return fmt.Sprintf("%ds", secs)
	}
}

var estLocation = func() *time.Location {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return time.FixedZone("EST", -5*60*60)
	}
	return loc
}()

func (s *Server) sendPeerAlert(peer models.Peer, iface models.Interface, event, endpoint string, duration time.Duration) {
	smtp, err := s.smtpSettings()
	if err != nil || smtp == nil {
		return
	}

	serverURL := s.cfg.Server.ExternalURL
	if u, err := s.store.GetSetting("external_url"); err == nil && u != "" {
		serverURL = u
	}

	timestamp := time.Now().In(estLocation).Format("Jan 2, 2006 3:04:05 PM MST")

	var durationStr string
	if duration > 0 {
		durationStr = formatDuration(duration)
	}

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
			Duration:      durationStr,
			Timestamp:     timestamp,
			ServerName:    serverURL,
		})
		if err != nil {
			log.Printf("alert: failed to send %s alert for peer %s to %s: %v", event, peer.Name, to, err)
		}
	}
}
