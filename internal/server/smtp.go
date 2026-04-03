package server

import (
	"strconv"

	"github.com/drudge/wgrift/internal/mail"
)

// smtpSettings resolves SMTP configuration from DB settings (primary) or config file (fallback).
func (s *Server) smtpSettings() (*mail.SMTPSettings, error) {
	// Check DB first
	host, _ := s.store.GetSetting("smtp_host")
	if host != "" {
		portStr, _ := s.store.GetSetting("smtp_port")
		port := 587
		if portStr != "" {
			if p, err := strconv.Atoi(portStr); err == nil {
				port = p
			}
		}

		username, _ := s.store.GetSetting("smtp_username")
		passwordEnc, _ := s.store.GetSetting("smtp_password_encrypted")
		from, _ := s.store.GetSetting("smtp_from")
		tlsMode, _ := s.store.GetSetting("smtp_tls")
		if tlsMode == "" {
			tlsMode = "starttls"
		}

		password := ""
		if passwordEnc != "" {
			decrypted, err := s.enc.Decrypt(passwordEnc)
			if err != nil {
				return nil, err
			}
			password = decrypted
		}

		return &mail.SMTPSettings{
			Host:     host,
			Port:     port,
			Username: username,
			Password: password,
			From:     from,
			TLS:      tlsMode,
		}, nil
	}

	// Fall back to config file
	if !s.cfg.SMTP.Enabled() {
		return nil, nil
	}

	password, err := s.cfg.SMTP.Password()
	if err != nil {
		return nil, err
	}

	port := s.cfg.SMTP.Port
	if port == 0 {
		port = 587
	}

	tlsMode := s.cfg.SMTP.TLS
	if tlsMode == "" {
		tlsMode = "starttls"
	}

	return &mail.SMTPSettings{
		Host:     s.cfg.SMTP.Host,
		Port:     port,
		Username: s.cfg.SMTP.Username,
		Password: password,
		From:     s.cfg.SMTP.From,
		TLS:      tlsMode,
	}, nil
}

// smtpEnabled returns true if SMTP is configured (either in DB or config file).
func (s *Server) smtpEnabled() bool {
	host, _ := s.store.GetSetting("smtp_host")
	if host != "" {
		return true
	}
	return s.cfg.SMTP.Enabled()
}
