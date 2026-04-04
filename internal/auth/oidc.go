package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/drudge/wgrift/internal/crypto"
	"github.com/drudge/wgrift/internal/models"
	"github.com/drudge/wgrift/internal/store"
	"golang.org/x/oauth2"
)

type resolvedProvider struct {
	model    models.OIDCProvider
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
}

// OIDCService manages OIDC authentication providers.
type OIDCService struct {
	mu        sync.RWMutex
	providers map[string]*resolvedProvider // keyed by provider ID
	store     store.Store
	enc       *crypto.Encryptor
	ready     chan struct{}
	readyOnce sync.Once
}

// NewOIDCService creates an OIDC service and starts loading providers in the background.
func NewOIDCService(s store.Store, enc *crypto.Encryptor) *OIDCService {
	svc := &OIDCService{
		providers: make(map[string]*resolvedProvider),
		store:     s,
		enc:       enc,
		ready:     make(chan struct{}),
	}
	go svc.Reload() // Non-blocking: discover in background
	return svc
}

// WaitReady blocks until the initial OIDC discovery has completed or the
// context is cancelled. Returns true if ready, false if the context expired.
func (s *OIDCService) WaitReady(ctx context.Context) bool {
	select {
	case <-s.ready:
		return true
	case <-ctx.Done():
		return false
	}
}

// Reload loads all enabled OIDC providers from the database and performs discovery.
func (s *OIDCService) Reload() {
	providers, err := s.store.ListOIDCProviders()
	if err != nil {
		log.Printf("OIDC: failed to list providers: %v", err)
		return
	}

	resolved := make(map[string]*resolvedProvider)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, p := range providers {
		if !p.Enabled {
			continue
		}

		secret, err := s.enc.Decrypt(p.ClientSecretEncrypted)
		if err != nil {
			log.Printf("OIDC: failed to decrypt client secret for %s: %v", p.Name, err)
			continue
		}

		provider, err := oidc.NewProvider(ctx, p.Issuer)
		if err != nil {
			log.Printf("OIDC: discovery failed for %s (%s): %v", p.Name, p.Issuer, err)
			continue
		}

		scopes := strings.Fields(strings.ReplaceAll(p.Scopes, ",", " "))
		if len(scopes) == 0 {
			scopes = []string{oidc.ScopeOpenID, "profile", "email", "groups"}
		}

		resolved[p.ID] = &resolvedProvider{
			model: p,
			oauth: oauth2.Config{
				ClientID:     p.ClientID,
				ClientSecret: secret,
				Scopes:       scopes,
				Endpoint:     provider.Endpoint(),
			},
			verifier: provider.Verifier(&oidc.Config{ClientID: p.ClientID}),
		}

		log.Printf("OIDC: loaded provider %s (%s)", p.Name, p.Issuer)
	}

	s.mu.Lock()
	s.providers = resolved
	s.mu.Unlock()

	s.readyOnce.Do(func() { close(s.ready) })
}

// ListProviders returns info about enabled and resolved OIDC providers.
func (s *OIDCService) ListProviders() []models.OIDCProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]models.OIDCProvider, 0, len(s.providers))
	for _, p := range s.providers {
		result = append(result, p.model)
	}
	return result
}

// AuthorizationURL generates an OIDC authorization URL for the given provider.
func (s *OIDCService) AuthorizationURL(providerID, externalURL, redirectURL string) (string, error) {
	s.mu.RLock()
	p, ok := s.providers[providerID]
	s.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("OIDC provider not found")
	}

	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	nonceBytes := make([]byte, 32)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}
	nonce := hex.EncodeToString(nonceBytes)

	if err := s.store.CreateOIDCState(&models.OIDCState{
		State:       state,
		ProviderID:  providerID,
		Nonce:       nonce,
		RedirectURL: redirectURL,
	}); err != nil {
		return "", fmt.Errorf("storing OIDC state: %w", err)
	}

	cfg := p.oauth
	cfg.RedirectURL = externalURL + "/api/v1/auth/oidc/callback"

	url := cfg.AuthCodeURL(state, oidc.Nonce(nonce))
	return url, nil
}

// HandleCallback processes an OIDC callback, returning the authenticated user.
func (s *OIDCService) HandleCallback(ctx context.Context, code, stateParam, externalURL string) (*models.User, string, error) {
	// Look up and consume the state (single-use)
	oidcState, err := s.store.GetOIDCState(stateParam)
	if err != nil {
		return nil, "", fmt.Errorf("looking up OIDC state: %w", err)
	}
	if oidcState == nil {
		return nil, "", fmt.Errorf("invalid or expired OIDC state")
	}
	s.store.DeleteOIDCState(stateParam)

	// Check TTL (10 minutes)
	if time.Since(oidcState.CreatedAt) > 10*time.Minute {
		return nil, "", fmt.Errorf("OIDC state expired")
	}

	redirectURL := oidcState.RedirectURL

	s.mu.RLock()
	p, ok := s.providers[oidcState.ProviderID]
	s.mu.RUnlock()
	if !ok {
		return nil, "", fmt.Errorf("OIDC provider no longer available")
	}

	// Exchange code for tokens
	cfg := p.oauth
	cfg.RedirectURL = externalURL + "/api/v1/auth/oidc/callback"

	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, "", fmt.Errorf("exchanging code: %w", err)
	}

	// Extract and verify ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, "", fmt.Errorf("no id_token in response")
	}

	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, "", fmt.Errorf("verifying id_token: %w", err)
	}

	// Verify nonce
	if idToken.Nonce != oidcState.Nonce {
		return nil, "", fmt.Errorf("nonce mismatch")
	}

	// Extract claims
	var claims struct {
		Subject           string `json:"sub"`
		PreferredUsername string `json:"preferred_username"`
		Email             string `json:"email"`
		Name              string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, "", fmt.Errorf("extracting claims: %w", err)
	}

	// Check admin claim if configured
	role := p.model.DefaultRole
	if role == "" {
		role = "viewer"
	}
	if p.model.AdminClaim != "" && p.model.AdminValue != "" {
		var allClaims map[string]any
		if err := idToken.Claims(&allClaims); err == nil {
			if matchesClaim(allClaims, p.model.AdminClaim, p.model.AdminValue) {
				role = "admin"
			}
		}
	}

	// Find existing user by OIDC identity
	user, err := s.store.GetUserByOIDCIdentity(p.model.Name, claims.Subject)
	if err != nil {
		return nil, "", fmt.Errorf("looking up OIDC user: %w", err)
	}

	if user != nil {
		// Update display name and role if changed
		changed := false
		if claims.Name != "" && claims.Name != user.DisplayName {
			user.DisplayName = claims.Name
			changed = true
		}
		if role != user.Role {
			user.Role = role
			changed = true
		}
		if changed {
			s.store.UpdateUser(user)
		}
		return user, redirectURL, nil
	}

	// JIT provision new user
	username := claims.PreferredUsername
	if username == "" {
		username = claims.Email
	}
	if username == "" {
		username = claims.Subject
	}

	// Handle username collision with a clean suffix
	existing, _ := s.store.GetUserByUsername(username)
	if existing != nil {
		slug := strings.ToLower(strings.ReplaceAll(p.model.Name, " ", "-"))
		username = username + "_" + slug
	}

	displayName := claims.Name
	if displayName == "" {
		displayName = username
	}

	user = &models.User{
		Username:     username,
		PasswordHash: "",
		DisplayName:  displayName,
		Role:         role,
		OIDCProvider: p.model.Name,
		OIDCSubject:  claims.Subject,
	}

	if err := s.store.CreateUser(user); err != nil {
		return nil, "", fmt.Errorf("creating OIDC user: %w", err)
	}

	return user, redirectURL, nil
}

// CleanExpiredStates removes OIDC states older than 10 minutes.
func (s *OIDCService) CleanExpiredStates() error {
	return s.store.DeleteExpiredOIDCStates()
}

// matchesClaim checks if a claims map contains a matching value for the given key.
// Supports both string and array-of-string claim values.
func matchesClaim(claims map[string]any, key, value string) bool {
	v, ok := claims[key]
	if !ok {
		return false
	}

	switch val := v.(type) {
	case string:
		return val == value
	case []any:
		for _, item := range val {
			if s, ok := item.(string); ok && s == value {
				return true
			}
		}
	}
	return false
}
