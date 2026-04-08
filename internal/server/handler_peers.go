package server

import (
	"fmt"
	"net/http"
	"net/mail"
	"strings"

	"github.com/drudge/wgrift/internal/models"
	"github.com/drudge/wgrift/internal/qr"
	"github.com/drudge/wgrift/internal/store"
)

type addPeerRequest struct {
	Type                string `json:"type"`
	Name                string `json:"name"`
	Address             string `json:"address"`
	AllowedIPs          string `json:"allowed_ips"`
	ClientAllowedIPs    string `json:"client_allowed_ips"`
	DNS                 string `json:"dns"`
	Endpoint            string `json:"endpoint"`
	PersistentKeepalive int    `json:"persistent_keepalive"`
	PSK                 bool   `json:"psk"`
	AlertOnConnect      bool   `json:"alert_on_connect"`
	AlertOnDisconnect   bool   `json:"alert_on_disconnect"`
	AlertEmails         string `json:"alert_emails"`
}

func (s *Server) handleListPeers(w http.ResponseWriter, r *http.Request) {
	ifaceID := r.PathValue("ifaceId")
	peers, err := s.store.ListPeers(ifaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, peers)
}

func (s *Server) handleAddPeer(w http.ResponseWriter, r *http.Request) {
	ifaceID := r.PathValue("ifaceId")

	var req addPeerRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Name == "" || req.AllowedIPs == "" {
		writeError(w, http.StatusBadRequest, "name and allowed_ips are required")
		return
	}

	peerType := models.PeerType(req.Type)
	if peerType != models.PeerTypeClient && peerType != models.PeerTypeSite {
		peerType = models.PeerTypeClient
	}

	if req.AlertEmails != "" {
		if err := validateEmails(req.AlertEmails); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	if req.Address != "" {
		inUse, err := s.store.IsTunnelIPInUse(req.Address, "", "")
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if inUse {
			writeValidationError(w, "tunnel IP address is already in use by another peer or interface")
			return
		}
	}

	peer := &models.Peer{
		InterfaceID:         ifaceID,
		Type:                peerType,
		Name:                req.Name,
		Address:             req.Address,
		AllowedIPs:          req.AllowedIPs,
		ClientAllowedIPs:    req.ClientAllowedIPs,
		DNS:                 req.DNS,
		Endpoint:            req.Endpoint,
		PersistentKeepalive: req.PersistentKeepalive,
		AlertOnConnect:      req.AlertOnConnect,
		AlertOnDisconnect:   req.AlertOnDisconnect,
		AlertEmails:         req.AlertEmails,
	}

	if err := s.manager.AddPeer(peer, req.PSK); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, peer)
}

type updatePeerRequest struct {
	Type                string  `json:"type"`
	Name                string  `json:"name"`
	Address             string  `json:"address"`
	AllowedIPs          string  `json:"allowed_ips"`
	ClientAllowedIPs    *string `json:"client_allowed_ips"`
	DNS                 *string `json:"dns"`
	Endpoint            string  `json:"endpoint"`
	PersistentKeepalive int     `json:"persistent_keepalive"`
	AlertOnConnect      *bool   `json:"alert_on_connect"`
	AlertOnDisconnect   *bool   `json:"alert_on_disconnect"`
	AlertEmails         *string `json:"alert_emails"`
}

func (s *Server) handleUpdatePeer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	peer, err := s.store.GetPeer(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if peer == nil {
		writeError(w, http.StatusNotFound, "peer not found")
		return
	}

	var req updatePeerRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Type != "" {
		peerType := models.PeerType(req.Type)
		if peerType == models.PeerTypeClient || peerType == models.PeerTypeSite {
			peer.Type = peerType
		}
	}
	if req.Name != "" {
		peer.Name = req.Name
	}
	if req.Address != "" {
		if store.ExtractHostIP(req.Address) != store.ExtractHostIP(peer.Address) {
			inUse, err := s.store.IsTunnelIPInUse(req.Address, peer.ID, "")
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if inUse {
				writeValidationError(w, "tunnel IP address is already in use by another peer or interface")
				return
			}
		}
		peer.Address = req.Address
	}
	if req.AllowedIPs != "" {
		peer.AllowedIPs = req.AllowedIPs
	}
	if req.ClientAllowedIPs != nil {
		peer.ClientAllowedIPs = *req.ClientAllowedIPs
	}
	if req.DNS != nil {
		peer.DNS = *req.DNS
	}
	peer.Endpoint = req.Endpoint
	peer.PersistentKeepalive = req.PersistentKeepalive
	if req.AlertOnConnect != nil {
		peer.AlertOnConnect = *req.AlertOnConnect
	}
	if req.AlertOnDisconnect != nil {
		peer.AlertOnDisconnect = *req.AlertOnDisconnect
	}
	if req.AlertEmails != nil {
		if *req.AlertEmails != "" {
			if err := validateEmails(*req.AlertEmails); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
		}
		peer.AlertEmails = *req.AlertEmails
	}

	if err := s.store.UpdatePeer(peer); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Sync the running interface so changes take effect immediately
	if err := s.manager.SyncInterface(peer.InterfaceID); err != nil {
		writeError(w, http.StatusInternalServerError, "peer updated but sync failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, peer)
}

func (s *Server) handleDeletePeer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.manager.RemovePeer(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleEnablePeer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.manager.EnablePeer(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "enabled"})
}

func (s *Server) handleDisablePeer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.manager.DisablePeer(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "disabled"})
}

func (s *Server) handleSetPeerPrivateKey(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	peer, err := s.store.GetPeer(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if peer == nil {
		writeError(w, http.StatusNotFound, "peer not found")
		return
	}

	var req struct {
		PrivateKey string `json:"private_key"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.PrivateKey == "" {
		writeError(w, http.StatusBadRequest, "private_key is required")
		return
	}

	encrypted, err := s.enc.Encrypt(req.PrivateKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "encrypting key: "+err.Error())
		return
	}

	peer.PrivateKeyEncrypted = encrypted
	if err := s.store.UpdatePeer(peer); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handlePeerConfig(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", "attachment; filename=wgrift.conf")
	if peer.Name != "" {
		w.Header().Set("X-Peer-Name", peer.Name)
	}
	w.Write([]byte(conf))
}

func (s *Server) handlePeerQR(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	conf, err := s.manager.GenerateConfig(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	png, err := qr.GeneratePNG(conf, 512)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(png)
}

func validateEmails(emails string) error {
	for _, addr := range strings.Split(emails, ",") {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}
		if _, err := mail.ParseAddress(addr); err != nil {
			return fmt.Errorf("invalid email address: %s", addr)
		}
	}
	return nil
}
