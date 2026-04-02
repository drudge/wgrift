package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/drudge/wgrift/internal/models"
	"github.com/google/uuid"
)

// --- Settings ---

func (s *SQLiteStore) GetSetting(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("querying setting %s: %w", key, err)
	}
	return value, nil
}

func (s *SQLiteStore) SetSetting(key, value string) error {
	_, err := s.db.Exec(`
		INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("setting %s: %w", key, err)
	}
	return nil
}

// --- OIDC Providers ---

func (s *SQLiteStore) CreateOIDCProvider(provider *models.OIDCProvider) error {
	if provider.ID == "" {
		provider.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	provider.CreatedAt = now
	provider.UpdatedAt = now

	_, err := s.db.Exec(`
		INSERT INTO oidc_providers (id, name, issuer, client_id, client_secret_encrypted, scopes, auto_discover, admin_claim, admin_value, default_role, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		provider.ID, provider.Name, provider.Issuer, provider.ClientID,
		provider.ClientSecretEncrypted, provider.Scopes, provider.AutoDiscover,
		provider.AdminClaim, provider.AdminValue, provider.DefaultRole, provider.Enabled,
		provider.CreatedAt, provider.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting OIDC provider: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetOIDCProvider(id string) (*models.OIDCProvider, error) {
	p := &models.OIDCProvider{}
	err := s.db.QueryRow(`
		SELECT id, name, issuer, client_id, client_secret_encrypted, scopes, auto_discover, admin_claim, admin_value, default_role, enabled, created_at, updated_at
		FROM oidc_providers WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &p.Issuer, &p.ClientID,
		&p.ClientSecretEncrypted, &p.Scopes, &p.AutoDiscover,
		&p.AdminClaim, &p.AdminValue, &p.DefaultRole, &p.Enabled,
		&p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying OIDC provider: %w", err)
	}
	return p, nil
}

func (s *SQLiteStore) GetOIDCProviderByName(name string) (*models.OIDCProvider, error) {
	p := &models.OIDCProvider{}
	err := s.db.QueryRow(`
		SELECT id, name, issuer, client_id, client_secret_encrypted, scopes, auto_discover, admin_claim, admin_value, default_role, enabled, created_at, updated_at
		FROM oidc_providers WHERE name = ?`, name,
	).Scan(&p.ID, &p.Name, &p.Issuer, &p.ClientID,
		&p.ClientSecretEncrypted, &p.Scopes, &p.AutoDiscover,
		&p.AdminClaim, &p.AdminValue, &p.DefaultRole, &p.Enabled,
		&p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying OIDC provider by name: %w", err)
	}
	return p, nil
}

func (s *SQLiteStore) ListOIDCProviders() ([]models.OIDCProvider, error) {
	rows, err := s.db.Query(`
		SELECT id, name, issuer, client_id, client_secret_encrypted, scopes, auto_discover, admin_claim, admin_value, default_role, enabled, created_at, updated_at
		FROM oidc_providers ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("querying OIDC providers: %w", err)
	}
	defer rows.Close()

	var result []models.OIDCProvider
	for rows.Next() {
		var p models.OIDCProvider
		if err := rows.Scan(&p.ID, &p.Name, &p.Issuer, &p.ClientID,
			&p.ClientSecretEncrypted, &p.Scopes, &p.AutoDiscover,
			&p.AdminClaim, &p.AdminValue, &p.DefaultRole, &p.Enabled,
			&p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning OIDC provider: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (s *SQLiteStore) UpdateOIDCProvider(provider *models.OIDCProvider) error {
	provider.UpdatedAt = time.Now().UTC()
	_, err := s.db.Exec(`
		UPDATE oidc_providers SET name=?, issuer=?, client_id=?, client_secret_encrypted=?, scopes=?, auto_discover=?, admin_claim=?, admin_value=?, default_role=?, enabled=?, updated_at=?
		WHERE id=?`,
		provider.Name, provider.Issuer, provider.ClientID,
		provider.ClientSecretEncrypted, provider.Scopes, provider.AutoDiscover,
		provider.AdminClaim, provider.AdminValue, provider.DefaultRole, provider.Enabled,
		provider.UpdatedAt, provider.ID,
	)
	if err != nil {
		return fmt.Errorf("updating OIDC provider: %w", err)
	}
	return nil
}

func (s *SQLiteStore) DeleteOIDCProvider(id string) error {
	_, err := s.db.Exec("DELETE FROM oidc_providers WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting OIDC provider: %w", err)
	}
	return nil
}

// --- OIDC States ---

func (s *SQLiteStore) CreateOIDCState(state *models.OIDCState) error {
	state.CreatedAt = time.Now().UTC()
	_, err := s.db.Exec(`
		INSERT INTO oidc_states (state, provider_id, nonce, created_at)
		VALUES (?, ?, ?, ?)`,
		state.State, state.ProviderID, state.Nonce, state.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting OIDC state: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetOIDCState(state string) (*models.OIDCState, error) {
	s2 := &models.OIDCState{}
	err := s.db.QueryRow(`
		SELECT state, provider_id, nonce, created_at
		FROM oidc_states WHERE state = ?`, state,
	).Scan(&s2.State, &s2.ProviderID, &s2.Nonce, &s2.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying OIDC state: %w", err)
	}
	return s2, nil
}

func (s *SQLiteStore) DeleteOIDCState(state string) error {
	_, err := s.db.Exec("DELETE FROM oidc_states WHERE state = ?", state)
	if err != nil {
		return fmt.Errorf("deleting OIDC state: %w", err)
	}
	return nil
}

func (s *SQLiteStore) DeleteExpiredOIDCStates() error {
	_, err := s.db.Exec("DELETE FROM oidc_states WHERE created_at < ?", time.Now().UTC().Add(-10*time.Minute))
	if err != nil {
		return fmt.Errorf("deleting expired OIDC states: %w", err)
	}
	return nil
}
