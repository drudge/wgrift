package server

import (
	"fmt"
	"net/http"
	gomail "net/mail"

	"github.com/drudge/wgrift/internal/mail"
	"github.com/drudge/wgrift/internal/models"
)

type settingsResponse struct {
	ExternalURL      string                `json:"external_url"`
	OIDCProviders    []models.OIDCProvider `json:"oidc_providers"`
	LocalAuthEnabled bool                  `json:"local_auth_enabled"`
	SMTP             *smtpSettingsData     `json:"smtp,omitempty"`
}

type smtpSettingsData struct {
	Host        string `json:"host"`
	Port        string `json:"port"`
	Username    string `json:"username"`
	HasPassword bool   `json:"has_password"`
	From        string `json:"from"`
	TLS         string `json:"tls"`
}

type updateSMTPRequest struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	TLS      string `json:"tls"`
}

type testSMTPRequest struct {
	To string `json:"to"`
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

	resp := settingsResponse{
		ExternalURL:      externalURL,
		OIDCProviders:    providers,
		LocalAuthEnabled: s.cfg.Auth.Local.Enabled,
	}

	// Load SMTP settings
	resp.SMTP = s.loadSMTPSettingsData()

	writeJSON(w, http.StatusOK, resp)
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

// --- SMTP Settings ---

func (s *Server) loadSMTPSettingsData() *smtpSettingsData {
	host, _ := s.store.GetSetting("smtp_host")
	if host == "" {
		// Check config fallback
		if !s.cfg.SMTP.Enabled() {
			return nil
		}
		port := fmt.Sprintf("%d", s.cfg.SMTP.Port)
		if s.cfg.SMTP.Port == 0 {
			port = "587"
		}
		tlsMode := s.cfg.SMTP.TLS
		if tlsMode == "" {
			tlsMode = "starttls"
		}
		pw, _ := s.cfg.SMTP.Password()
		return &smtpSettingsData{
			Host:        s.cfg.SMTP.Host,
			Port:        port,
			Username:    s.cfg.SMTP.Username,
			HasPassword: pw != "",
			From:        s.cfg.SMTP.From,
			TLS:         tlsMode,
		}
	}

	port, _ := s.store.GetSetting("smtp_port")
	if port == "" {
		port = "587"
	}
	username, _ := s.store.GetSetting("smtp_username")
	passwordEnc, _ := s.store.GetSetting("smtp_password_encrypted")
	from, _ := s.store.GetSetting("smtp_from")
	tlsMode, _ := s.store.GetSetting("smtp_tls")
	if tlsMode == "" {
		tlsMode = "starttls"
	}

	return &smtpSettingsData{
		Host:        host,
		Port:        port,
		Username:    username,
		HasPassword: passwordEnc != "",
		From:        from,
		TLS:         tlsMode,
	}
}

func (s *Server) handleUpdateSMTP(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user.Role != "admin" {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}

	var req updateSMTPRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Host == "" {
		writeError(w, http.StatusBadRequest, "host is required")
		return
	}
	if req.From == "" {
		writeError(w, http.StatusBadRequest, "from address is required")
		return
	}

	if err := s.store.SetSetting("smtp_host", req.Host); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	port := req.Port
	if port == "" {
		port = "587"
	}
	if err := s.store.SetSetting("smtp_port", port); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := s.store.SetSetting("smtp_username", req.Username); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Only update password if provided (like OIDC client secret)
	if req.Password != "" {
		encrypted, err := s.enc.Encrypt(req.Password)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to encrypt password")
			return
		}
		if err := s.store.SetSetting("smtp_password_encrypted", encrypted); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if err := s.store.SetSetting("smtp_from", req.From); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tlsMode := req.TLS
	if tlsMode == "" {
		tlsMode = "starttls"
	}
	if err := s.store.SetSetting("smtp_tls", tlsMode); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleTestSMTP(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user.Role != "admin" {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}

	var req testSMTPRequest
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

	smtp, err := s.smtpSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load SMTP settings: "+err.Error())
		return
	}
	if smtp == nil {
		writeError(w, http.StatusBadRequest, "SMTP is not configured")
		return
	}

	if err := mail.SendTestEmail(*smtp, req.To, s.externalURL(r)); err != nil {
		writeError(w, http.StatusInternalServerError, "SMTP test failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "sent", "to": req.To})
}

func (s *Server) handleDeleteSMTP(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user.Role != "admin" {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}

	keys := []string{"smtp_host", "smtp_port", "smtp_username", "smtp_password_encrypted", "smtp_from", "smtp_tls"}
	for _, key := range keys {
		if err := s.store.SetSetting(key, ""); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
