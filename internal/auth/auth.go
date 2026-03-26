package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/drudge/wgrift/internal/models"
	"github.com/drudge/wgrift/internal/store"
	"golang.org/x/crypto/bcrypt"
)

// Service handles authentication and session management.
type Service struct {
	store          store.Store
	sessionTimeout time.Duration
	maxSessionAge  time.Duration
	minPasswordLen int
}

// NewService creates an auth service.
func NewService(s store.Store, sessionTimeout, maxSessionAge time.Duration, minPwLen int) *Service {
	return &Service{
		store:          s,
		sessionTimeout: sessionTimeout,
		maxSessionAge:  maxSessionAge,
		minPasswordLen: minPwLen,
	}
}

// HashPassword hashes a password with bcrypt.
func (s *Service) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword verifies a password against a bcrypt hash.
func (s *Service) CheckPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// ValidatePasswordStrength checks minimum password requirements.
func (s *Service) ValidatePasswordStrength(password string) error {
	if len(password) < s.minPasswordLen {
		return fmt.Errorf("password must be at least %d characters", s.minPasswordLen)
	}
	return nil
}

// Authenticate validates credentials and returns the user.
func (s *Service) Authenticate(username, password string) (*models.User, error) {
	user, err := s.store.GetUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("looking up user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := s.CheckPassword(user.PasswordHash, password); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return user, nil
}

// CreateSession creates a new session for a user.
func (s *Service) CreateSession(userID, ip, userAgent string) (*models.Session, error) {
	sessionID, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generating session ID: %w", err)
	}

	csrfToken, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generating CSRF token: %w", err)
	}

	session := &models.Session{
		ID:        sessionID,
		UserID:    userID,
		CSRFToken: csrfToken,
		ExpiresAt: time.Now().UTC().Add(s.maxSessionAge),
		IPAddress: ip,
		UserAgent: userAgent,
	}

	if err := s.store.CreateSession(session); err != nil {
		return nil, fmt.Errorf("storing session: %w", err)
	}

	return session, nil
}

// ValidateSession checks if a session is valid, not expired, and touches it.
// Returns the session and user, or an error.
func (s *Service) ValidateSession(sessionID string) (*models.Session, *models.User, error) {
	session, err := s.store.GetSession(sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("getting session: %w", err)
	}
	if session == nil {
		return nil, nil, fmt.Errorf("session not found")
	}

	now := time.Now().UTC()

	// Check absolute expiry
	if now.After(session.ExpiresAt) {
		s.store.DeleteSession(sessionID)
		return nil, nil, fmt.Errorf("session expired")
	}

	// Check idle timeout
	if now.Sub(session.LastSeenAt) > s.sessionTimeout {
		s.store.DeleteSession(sessionID)
		return nil, nil, fmt.Errorf("session timed out")
	}

	// Touch session
	if err := s.store.TouchSession(sessionID); err != nil {
		return nil, nil, fmt.Errorf("touching session: %w", err)
	}

	user, err := s.store.GetUser(session.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("getting user: %w", err)
	}
	if user == nil {
		s.store.DeleteSession(sessionID)
		return nil, nil, fmt.Errorf("user not found")
	}

	return session, user, nil
}

// DestroySession removes a session.
func (s *Service) DestroySession(sessionID string) error {
	return s.store.DeleteSession(sessionID)
}

// CleanExpiredSessions removes all expired sessions.
func (s *Service) CleanExpiredSessions() error {
	return s.store.DeleteExpiredSessions()
}

// NeedsSetup returns true if no users exist yet.
func (s *Service) NeedsSetup() (bool, error) {
	count, err := s.store.CountUsers()
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// CreateInitialAdmin creates the first admin user.
func (s *Service) CreateInitialAdmin(username, password string) (*models.User, error) {
	needs, err := s.NeedsSetup()
	if err != nil {
		return nil, err
	}
	if !needs {
		return nil, fmt.Errorf("setup already complete")
	}

	if err := s.ValidatePasswordStrength(password); err != nil {
		return nil, err
	}

	hash, err := s.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username:     username,
		PasswordHash: hash,
		DisplayName:  username,
		Role:         "admin",
		IsInitial:    true,
	}

	if err := s.store.CreateUser(user); err != nil {
		return nil, fmt.Errorf("creating admin user: %w", err)
	}

	return user, nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
