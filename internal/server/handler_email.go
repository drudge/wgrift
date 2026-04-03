package server

import (
	"net/http"
	gomail "net/mail"

	"github.com/drudge/wgrift/internal/mail"
	"github.com/drudge/wgrift/internal/qr"
)

type emailPeerConfigRequest struct {
	To   string `json:"to"`
	Note string `json:"note"`
}

func (s *Server) handleEmailPeerConfig(w http.ResponseWriter, r *http.Request) {
	smtp, err := s.smtpSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load SMTP settings")
		return
	}
	if smtp == nil {
		writeError(w, http.StatusBadRequest, "SMTP is not configured")
		return
	}

	var req emailPeerConfigRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.To == "" {
		writeError(w, http.StatusBadRequest, "email address is required")
		return
	}
	if _, err := gomail.ParseAddress(req.To); err != nil {
		writeError(w, http.StatusBadRequest, "invalid email address")
		return
	}

	id := r.PathValue("id")

	peer, err := s.store.GetPeer(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "peer not found")
		return
	}

	conf, err := s.manager.GenerateConfig(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	qrPNG, err := qr.GeneratePNG(conf, 512)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate QR code")
		return
	}

	serverName := s.externalURL(r)

	if err := mail.SendPeerConfig(*smtp, mail.PeerConfigEmail{
		To:         req.To,
		PeerName:   peer.Name,
		Note:       req.Note,
		ConfData:   []byte(conf),
		QRCodePNG:  qrPNG,
		ServerName: serverName,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to send email: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "sent",
		"to":     req.To,
	})
}
