package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/drudge/wgrift/internal/confgen"
	"github.com/drudge/wgrift/internal/models"
)

type createInterfaceRequest struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	ListenPort int    `json:"listen_port"`
	Address    string `json:"address"`
	DNS        string `json:"dns"`
	MTU        int    `json:"mtu"`
	Endpoint   string `json:"endpoint"`
}

func (s *Server) handleListInterfaces(w http.ResponseWriter, r *http.Request) {
	ifaces, err := s.store.ListInterfaces()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ifaces)
}

func (s *Server) handleCreateInterface(w http.ResponseWriter, r *http.Request) {
	var req createInterfaceRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.ID == "" || req.Type == "" || req.ListenPort == 0 || req.Address == "" {
		writeError(w, http.StatusBadRequest, "id, type, listen_port, and address are required")
		return
	}

	if req.Type != "site-to-site" && req.Type != "client-access" {
		writeError(w, http.StatusBadRequest, "type must be 'site-to-site' or 'client-access'")
		return
	}

	if req.MTU == 0 {
		req.MTU = 1420
	}

	iface := &models.Interface{
		ID:         req.ID,
		Type:       models.InterfaceType(req.Type),
		ListenPort: req.ListenPort,
		Address:    req.Address,
		DNS:        req.DNS,
		MTU:        req.MTU,
		Endpoint:   req.Endpoint,
	}

	if err := s.manager.CreateInterface(iface); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, iface)
}

type updateInterfaceRequest struct {
	Address    string `json:"address"`
	ListenPort int    `json:"listen_port"`
	DNS        string `json:"dns"`
	MTU        int    `json:"mtu"`
	Endpoint   string `json:"endpoint"`
}

func (s *Server) handleUpdateInterface(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	iface, err := s.store.GetInterface(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if iface == nil {
		writeError(w, http.StatusNotFound, "interface not found")
		return
	}

	var req updateInterfaceRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Address != "" {
		iface.Address = req.Address
	}
	if req.ListenPort > 0 {
		iface.ListenPort = req.ListenPort
	}
	iface.DNS = req.DNS
	iface.Endpoint = req.Endpoint
	if req.MTU > 0 {
		iface.MTU = req.MTU
	}

	if err := s.store.UpdateInterface(iface); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, iface)
}

func (s *Server) handleGetInterface(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	iface, err := s.store.GetInterface(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if iface == nil {
		writeError(w, http.StatusNotFound, "interface not found")
		return
	}
	writeJSON(w, http.StatusOK, iface)
}

func (s *Server) handleDeleteInterface(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.manager.DeleteInterface(id, true); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleSyncInterface(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.manager.SyncInterface(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "synced"})
}

type importConfigRequest struct {
	ID     string `json:"id"`     // interface name e.g. "wg0"
	Type   string `json:"type"`   // "client-access" or "site-to-site"
	Config string `json:"config"` // raw wg config text
}

func (s *Server) handleImportInterface(w http.ResponseWriter, r *http.Request) {
	var req importConfigRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.ID == "" || req.Type == "" || req.Config == "" {
		writeError(w, http.StatusBadRequest, "id, type, and config are required")
		return
	}

	if req.Type != "site-to-site" && req.Type != "client-access" {
		writeError(w, http.StatusBadRequest, "type must be 'site-to-site' or 'client-access'")
		return
	}

	parsed, err := confgen.ParseConfig(req.Config)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid config: "+err.Error())
		return
	}

	iface, err := s.manager.ImportInterface(parsed, req.ID, req.Type)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, iface)
}

type adoptInterfaceRequest struct {
	ID   string `json:"id"`   // interface name e.g. "wg0"
	Type string `json:"type"` // "client-access" or "site-to-site"
}

func (s *Server) handleAdoptInterface(w http.ResponseWriter, r *http.Request) {
	var req adoptInterfaceRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	if req.Type == "" {
		req.Type = "client-access"
	}

	configPath := filepath.Join("/etc/wireguard", req.ID+".conf")
	data, err := os.ReadFile(configPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("cannot read %s: %v", configPath, err))
		return
	}

	parsed, err := confgen.ParseConfig(string(data))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid config: "+err.Error())
		return
	}

	iface, err := s.manager.ImportInterface(parsed, req.ID, req.Type)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"interface":  iface,
		"peer_count": len(parsed.Peers),
	})
}

func (s *Server) handleStartInterface(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.manager.StartInterface(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

func (s *Server) handleStopInterface(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.manager.StopInterface(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (s *Server) handleRestartInterface(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.manager.RestartInterface(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restarted"})
}

func (s *Server) handleInterfaceStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	status, err := s.manager.GetStatus(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}
