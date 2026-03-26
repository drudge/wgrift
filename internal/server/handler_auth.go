package server

import (
	"net/http"
	"time"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type sessionResponse struct {
	User      any    `json:"user"`
	CSRFToken string `json:"csrf_token"`
}

type setupCheckResponse struct {
	NeedsSetup bool `json:"needs_setup"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := s.auth.Authenticate(req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	session, err := s.auth.CreateSession(user.ID, r.RemoteAddr, r.UserAgent())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   s.cfg.Server.TLS.Mode != "none",
		MaxAge:   int(24 * time.Hour / time.Second),
	})

	writeJSON(w, http.StatusOK, sessionResponse{
		User:      user,
		CSRFToken: session.CSRFToken,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(cookieName)
	if err == nil {
		s.auth.DestroySession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	// Check if setup is needed first
	needs, err := s.auth.NeedsSetup()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "checking setup status")
		return
	}
	if needs {
		writeJSON(w, http.StatusOK, setupCheckResponse{NeedsSetup: true})
		return
	}

	cookie, err := r.Cookie(cookieName)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	session, user, err := s.auth.ValidateSession(cookie.Value)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid session")
		return
	}

	writeJSON(w, http.StatusOK, sessionResponse{
		User:      user,
		CSRFToken: session.CSRFToken,
	})
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password required")
		return
	}

	user, err := s.auth.CreateInitialAdmin(req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	session, err := s.auth.CreateSession(user.ID, r.RemoteAddr, r.UserAgent())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   s.cfg.Server.TLS.Mode != "none",
		MaxAge:   int(24 * time.Hour / time.Second),
	})

	writeJSON(w, http.StatusCreated, sessionResponse{
		User:      user,
		CSRFToken: session.CSRFToken,
	})
}
