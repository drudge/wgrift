package server

import (
	"net/http"

	"github.com/drudge/wgrift/internal/models"
)

type settingsResponse struct {
	ExternalURL      string                `json:"external_url"`
	OIDCProviders    []models.OIDCProvider `json:"oidc_providers"`
	LocalAuthEnabled bool                  `json:"local_auth_enabled"`
}

type updateSettingsRequest struct {
	ExternalURL string `json:"external_url"`
}

type oidcProviderRequest struct {
	Name         string `json:"name"`
	Issuer       string `json:"issuer"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Scopes       string `json:"scopes"`
	AutoDiscover bool   `json:"auto_discover"`
	AdminClaim   string `json:"admin_claim"`
	AdminValue   string `json:"admin_value"`
	DefaultRole  string `json:"default_role"`
	Enabled      bool   `json:"enabled"`
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user.Role != "admin" {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}

	externalURL, _ := s.store.GetSetting("external_url")

	providers, err := s.store.ListOIDCProviders()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if providers == nil {
		providers = []models.OIDCProvider{}
	}

	writeJSON(w, http.StatusOK, settingsResponse{
		ExternalURL:      externalURL,
		OIDCProviders:    providers,
		LocalAuthEnabled: s.cfg.Auth.Local.Enabled,
	})
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user.Role != "admin" {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}

	var req updateSettingsRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.store.SetSetting("external_url", req.ExternalURL); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleCreateOIDCProvider(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user.Role != "admin" {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}

	var req oidcProviderRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Name == "" || req.Issuer == "" || req.ClientID == "" || req.ClientSecret == "" {
		writeError(w, http.StatusBadRequest, "name, issuer, client_id, and client_secret are required")
		return
	}

	if req.DefaultRole == "" {
		req.DefaultRole = "viewer"
	}
	if req.DefaultRole != "admin" && req.DefaultRole != "viewer" {
		writeError(w, http.StatusBadRequest, "default_role must be 'admin' or 'viewer'")
		return
	}

	if req.Scopes == "" {
		req.Scopes = "openid profile email groups"
	}

	encrypted, err := s.enc.Encrypt(req.ClientSecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encrypt client secret")
		return
	}

	provider := &models.OIDCProvider{
		Name:                  req.Name,
		Issuer:                req.Issuer,
		ClientID:              req.ClientID,
		ClientSecretEncrypted: encrypted,
		Scopes:                req.Scopes,
		AutoDiscover:          req.AutoDiscover,
		AdminClaim:            req.AdminClaim,
		AdminValue:            req.AdminValue,
		DefaultRole:           req.DefaultRole,
		Enabled:               req.Enabled,
	}

	if err := s.store.CreateOIDCProvider(provider); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if s.oidc != nil {
		s.oidc.Reload()
	}

	writeJSON(w, http.StatusCreated, provider)
}

func (s *Server) handleUpdateOIDCProvider(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user.Role != "admin" {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}

	id := r.PathValue("id")
	existing, err := s.store.GetOIDCProvider(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "OIDC provider not found")
		return
	}

	var req oidcProviderRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Name == "" || req.Issuer == "" || req.ClientID == "" {
		writeError(w, http.StatusBadRequest, "name, issuer, and client_id are required")
		return
	}

	if req.DefaultRole == "" {
		req.DefaultRole = "viewer"
	}
	if req.Scopes == "" {
		req.Scopes = "openid profile email groups"
	}

	existing.Name = req.Name
	existing.Issuer = req.Issuer
	existing.ClientID = req.ClientID
	existing.Scopes = req.Scopes
	existing.AutoDiscover = req.AutoDiscover
	existing.AdminClaim = req.AdminClaim
	existing.AdminValue = req.AdminValue
	existing.DefaultRole = req.DefaultRole
	existing.Enabled = req.Enabled

	// Only re-encrypt if client secret was provided (non-empty means update)
	if req.ClientSecret != "" {
		encrypted, err := s.enc.Encrypt(req.ClientSecret)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to encrypt client secret")
			return
		}
		existing.ClientSecretEncrypted = encrypted
	}

	if err := s.store.UpdateOIDCProvider(existing); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if s.oidc != nil {
		s.oidc.Reload()
	}

	writeJSON(w, http.StatusOK, existing)
}

func (s *Server) handleDeleteOIDCProvider(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user.Role != "admin" {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}

	id := r.PathValue("id")
	if err := s.store.DeleteOIDCProvider(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if s.oidc != nil {
		s.oidc.Reload()
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
