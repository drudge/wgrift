package server

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"
)

func (s *Server) handleOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if s.oidc == nil {
		writeError(w, http.StatusNotFound, "OIDC not configured")
		return
	}

	// Wait briefly for OIDC discovery if it hasn't finished yet.
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if !s.oidc.WaitReady(ctx) {
		writeError(w, http.StatusServiceUnavailable, "OIDC providers are still loading, please try again")
		return
	}

	providerID := r.PathValue("id")
	if providerID == "" {
		writeError(w, http.StatusBadRequest, "missing provider ID")
		return
	}

	redirectURL := sanitizeRedirectURL(r.URL.Query().Get("redirect"))

	url, err := s.oidc.AuthorizationURL(providerID, s.externalURL(r), redirectURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	http.Redirect(w, r, url, http.StatusFound)
}

func (s *Server) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	if s.oidc == nil {
		http.Redirect(w, r, "/?error=oidc_not_configured", http.StatusFound)
		return
	}

	// Check for error from provider
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		desc := r.URL.Query().Get("error_description")
		log.Printf("OIDC callback error: %s - %s", errParam, desc)
		http.Redirect(w, r, "/?error=access_denied", http.StatusFound)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		http.Redirect(w, r, "/?error=invalid_callback", http.StatusFound)
		return
	}

	user, redirectURL, err := s.oidc.HandleCallback(r.Context(), code, state, s.externalURL(r))
	if err != nil {
		log.Printf("OIDC callback failed: %v", err)
		http.Redirect(w, r, "/?error=auth_failed", http.StatusFound)
		return
	}

	session, err := s.auth.CreateSession(user.ID, r.RemoteAddr, r.UserAgent())
	if err != nil {
		log.Printf("OIDC session creation failed: %v", err)
		http.Redirect(w, r, "/?error=session_failed", http.StatusFound)
		return
	}

	// Use SameSite=Lax because this is a cross-origin redirect from the OIDC provider.
	// SameSiteStrict would cause the cookie to be dropped on the redirected GET.
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.Server.TLS.Mode != "none",
		MaxAge:   int(24 * time.Hour / time.Second),
	})

	dest := sanitizeRedirectURL(redirectURL)
	if dest == "" {
		dest = "/"
	}
	http.Redirect(w, r, dest, http.StatusFound)
}

// sanitizeRedirectURL validates a redirect URL to prevent open redirects.
// Returns the URL if safe, or empty string if invalid.
func sanitizeRedirectURL(u string) string {
	if u == "" || !strings.HasPrefix(u, "/") || strings.HasPrefix(u, "//") {
		return ""
	}
	return u
}
