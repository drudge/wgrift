package server

import (
	"net/http"

	"github.com/drudge/wgrift/internal/models"
)

type createUserRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type changePasswordRequest struct {
	Password string `json:"password"`
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user.Role != "admin" {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}

	users, err := s.store.ListUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Enrich OIDC users with issuer URL
	providers, _ := s.store.ListOIDCProviders()
	issuerMap := make(map[string]string, len(providers))
	for _, p := range providers {
		issuerMap[p.Name] = p.Issuer
	}
	for i := range users {
		if users[i].OIDCProvider != "" {
			users[i].OIDCIssuer = issuerMap[users[i].OIDCProvider]
		}
	}

	writeJSON(w, http.StatusOK, users)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	caller := UserFromContext(r.Context())
	if caller.Role != "admin" {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}

	var req createUserRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password required")
		return
	}

	if req.Role == "" {
		req.Role = "viewer"
	}
	if req.Role != "admin" && req.Role != "viewer" {
		writeError(w, http.StatusBadRequest, "role must be 'admin' or 'viewer'")
		return
	}

	if err := s.auth.ValidatePasswordStrength(req.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := s.auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user := &models.User{
		Username:     req.Username,
		PasswordHash: hash,
		DisplayName:  req.DisplayName,
		Role:         req.Role,
	}
	if user.DisplayName == "" {
		user.DisplayName = user.Username
	}

	if err := s.store.CreateUser(user); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	caller := UserFromContext(r.Context())
	if caller.Role != "admin" {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}

	id := r.PathValue("id")
	if id == caller.ID {
		writeError(w, http.StatusBadRequest, "cannot delete yourself")
		return
	}

	if err := s.store.DeleteUser(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	caller := UserFromContext(r.Context())
	id := r.PathValue("id")

	// Users can change their own password; admins can change anyone's
	if id != caller.ID && caller.Role != "admin" {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	var req changePasswordRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.auth.ValidatePasswordStrength(req.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := s.store.GetUser(id)
	if err != nil || user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	if user.OIDCProvider != "" {
		writeError(w, http.StatusBadRequest, "cannot change password for SSO user")
		return
	}

	hash, err := s.auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user.PasswordHash = hash
	if err := s.store.UpdateUser(user); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
