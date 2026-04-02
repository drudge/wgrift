package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/drudge/wgrift/internal/models"
	"github.com/google/uuid"

	_ "modernc.org/sqlite"
)

// Store defines the data access interface.
type Store interface {
	// Interfaces
	CreateInterface(iface *models.Interface) error
	GetInterface(id string) (*models.Interface, error)
	ListInterfaces() ([]models.Interface, error)
	UpdateInterface(iface *models.Interface) error
	DeleteInterface(id string) error

	// Peers
	CreatePeer(peer *models.Peer) error
	GetPeer(id string) (*models.Peer, error)
	ListPeers(interfaceID string) ([]models.Peer, error)
	ListAllPeers() ([]models.Peer, error)
	UpdatePeer(peer *models.Peer) error
	DeletePeer(id string) error

	// Users
	CreateUser(user *models.User) error
	GetUser(id string) (*models.User, error)
	GetUserByUsername(username string) (*models.User, error)
	GetUserByOIDCIdentity(provider, subject string) (*models.User, error)
	ListUsers() ([]models.User, error)
	UpdateUser(user *models.User) error
	DeleteUser(id string) error
	CountUsers() (int, error)

	// Sessions
	CreateSession(session *models.Session) error
	GetSession(id string) (*models.Session, error)
	TouchSession(id string) error
	DeleteSession(id string) error
	DeleteExpiredSessions() error
	DeleteUserSessions(userID string) error

	// Settings
	GetSetting(key string) (string, error)
	SetSetting(key, value string) error

	// OIDC Providers
	CreateOIDCProvider(provider *models.OIDCProvider) error
	GetOIDCProvider(id string) (*models.OIDCProvider, error)
	GetOIDCProviderByName(name string) (*models.OIDCProvider, error)
	ListOIDCProviders() ([]models.OIDCProvider, error)
	UpdateOIDCProvider(provider *models.OIDCProvider) error
	DeleteOIDCProvider(id string) error

	// OIDC States
	CreateOIDCState(state *models.OIDCState) error
	GetOIDCState(state string) (*models.OIDCState, error)
	DeleteOIDCState(state string) error
	DeleteExpiredOIDCStates() error

	// Connection Logs
	CreateConnectionLog(log *models.ConnectionLog) error
	ListConnectionLogs(interfaceID string, limit, offset int) ([]models.ConnectionLog, int, error)
	ListPeerConnectionLogs(peerID string, limit, offset int) ([]models.ConnectionLog, int, error)
	DeleteOldConnectionLogs(before time.Time) error

	Close() error
}

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// New opens a SQLite database and runs migrations.
func New(dsn string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode and foreign keys
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("setting %s: %w", pragma, err)
		}
	}

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// --- Interfaces ---

func (s *SQLiteStore) CreateInterface(iface *models.Interface) error {
	now := time.Now().UTC()
	iface.CreatedAt = now
	iface.UpdatedAt = now

	_, err := s.db.Exec(`
		INSERT INTO interfaces (id, type, listen_port, private_key_encrypted, address, dns, mtu, endpoint, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		iface.ID, iface.Type, iface.ListenPort, iface.PrivateKeyEncrypted,
		iface.Address, iface.DNS, iface.MTU, iface.Endpoint, iface.Enabled,
		iface.CreatedAt, iface.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting interface: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetInterface(id string) (*models.Interface, error) {
	iface := &models.Interface{}
	err := s.db.QueryRow(`
		SELECT id, type, listen_port, private_key_encrypted, address, dns, mtu, endpoint, enabled, created_at, updated_at
		FROM interfaces WHERE id = ?`, id,
	).Scan(
		&iface.ID, &iface.Type, &iface.ListenPort, &iface.PrivateKeyEncrypted,
		&iface.Address, &iface.DNS, &iface.MTU, &iface.Endpoint, &iface.Enabled,
		&iface.CreatedAt, &iface.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying interface: %w", err)
	}
	return iface, nil
}

func (s *SQLiteStore) ListInterfaces() ([]models.Interface, error) {
	rows, err := s.db.Query(`
		SELECT id, type, listen_port, private_key_encrypted, address, dns, mtu, endpoint, enabled, created_at, updated_at
		FROM interfaces ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("querying interfaces: %w", err)
	}
	defer rows.Close()

	var result []models.Interface
	for rows.Next() {
		var iface models.Interface
		if err := rows.Scan(
			&iface.ID, &iface.Type, &iface.ListenPort, &iface.PrivateKeyEncrypted,
			&iface.Address, &iface.DNS, &iface.MTU, &iface.Endpoint, &iface.Enabled,
			&iface.CreatedAt, &iface.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning interface: %w", err)
		}
		result = append(result, iface)
	}
	return result, rows.Err()
}

func (s *SQLiteStore) UpdateInterface(iface *models.Interface) error {
	iface.UpdatedAt = time.Now().UTC()
	_, err := s.db.Exec(`
		UPDATE interfaces SET type=?, listen_port=?, private_key_encrypted=?, address=?, dns=?, mtu=?, endpoint=?, enabled=?, updated_at=?
		WHERE id=?`,
		iface.Type, iface.ListenPort, iface.PrivateKeyEncrypted,
		iface.Address, iface.DNS, iface.MTU, iface.Endpoint, iface.Enabled,
		iface.UpdatedAt, iface.ID,
	)
	if err != nil {
		return fmt.Errorf("updating interface: %w", err)
	}
	return nil
}

func (s *SQLiteStore) DeleteInterface(id string) error {
	_, err := s.db.Exec("DELETE FROM interfaces WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting interface: %w", err)
	}
	return nil
}

// --- Peers ---

func (s *SQLiteStore) CreatePeer(peer *models.Peer) error {
	if peer.ID == "" {
		peer.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	peer.CreatedAt = now
	peer.UpdatedAt = now

	_, err := s.db.Exec(`
		INSERT INTO peers (id, interface_id, name, public_key, private_key_encrypted, preshared_key_encrypted, address, allowed_ips, client_allowed_ips, dns, endpoint, persistent_keepalive, enabled, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		peer.ID, peer.InterfaceID, peer.Name, peer.PublicKey,
		peer.PrivateKeyEncrypted, nullString(peer.PresharedKeyEncrypted),
		peer.Address, peer.AllowedIPs, peer.ClientAllowedIPs, peer.DNS, peer.Endpoint, peer.PersistentKeepalive,
		peer.Enabled, nullTime(peer.ExpiresAt),
		peer.CreatedAt, peer.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting peer: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetPeer(id string) (*models.Peer, error) {
	peer := &models.Peer{}
	var psk, endpoint sql.NullString
	var expiresAt, lastHandshake sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, interface_id, name, public_key, private_key_encrypted, preshared_key_encrypted, address, allowed_ips, client_allowed_ips, dns, endpoint, persistent_keepalive, enabled, expires_at, last_handshake, transfer_rx, transfer_tx, created_at, updated_at
		FROM peers WHERE id = ?`, id,
	).Scan(
		&peer.ID, &peer.InterfaceID, &peer.Name, &peer.PublicKey,
		&peer.PrivateKeyEncrypted, &psk, &peer.Address, &peer.AllowedIPs, &peer.ClientAllowedIPs, &peer.DNS, &endpoint,
		&peer.PersistentKeepalive, &peer.Enabled,
		&expiresAt, &lastHandshake,
		&peer.TransferRx, &peer.TransferTx,
		&peer.CreatedAt, &peer.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying peer: %w", err)
	}

	if psk.Valid {
		peer.PresharedKeyEncrypted = psk.String
	}
	if endpoint.Valid {
		peer.Endpoint = endpoint.String
	}
	if expiresAt.Valid {
		peer.ExpiresAt = &expiresAt.Time
	}
	if lastHandshake.Valid {
		peer.LastHandshake = &lastHandshake.Time
	}

	return peer, nil
}

func (s *SQLiteStore) ListPeers(interfaceID string) ([]models.Peer, error) {
	rows, err := s.db.Query(`
		SELECT id, interface_id, name, public_key, private_key_encrypted, preshared_key_encrypted, address, allowed_ips, client_allowed_ips, dns, endpoint, persistent_keepalive, enabled, expires_at, last_handshake, transfer_rx, transfer_tx, created_at, updated_at
		FROM peers WHERE interface_id = ? ORDER BY created_at`, interfaceID)
	if err != nil {
		return nil, fmt.Errorf("querying peers: %w", err)
	}
	defer rows.Close()

	return scanPeers(rows)
}

func (s *SQLiteStore) ListAllPeers() ([]models.Peer, error) {
	rows, err := s.db.Query(`
		SELECT id, interface_id, name, public_key, private_key_encrypted, preshared_key_encrypted, address, allowed_ips, client_allowed_ips, dns, endpoint, persistent_keepalive, enabled, expires_at, last_handshake, transfer_rx, transfer_tx, created_at, updated_at
		FROM peers ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("querying all peers: %w", err)
	}
	defer rows.Close()

	return scanPeers(rows)
}

func (s *SQLiteStore) UpdatePeer(peer *models.Peer) error {
	peer.UpdatedAt = time.Now().UTC()
	_, err := s.db.Exec(`
		UPDATE peers SET interface_id=?, name=?, public_key=?, private_key_encrypted=?, preshared_key_encrypted=?, address=?, allowed_ips=?, client_allowed_ips=?, dns=?, endpoint=?, persistent_keepalive=?, enabled=?, expires_at=?, last_handshake=?, transfer_rx=?, transfer_tx=?, updated_at=?
		WHERE id=?`,
		peer.InterfaceID, peer.Name, peer.PublicKey,
		peer.PrivateKeyEncrypted, nullString(peer.PresharedKeyEncrypted),
		peer.Address, peer.AllowedIPs, peer.ClientAllowedIPs, peer.DNS, peer.Endpoint, peer.PersistentKeepalive,
		peer.Enabled, nullTime(peer.ExpiresAt), nullTime(peer.LastHandshake),
		peer.TransferRx, peer.TransferTx, peer.UpdatedAt,
		peer.ID,
	)
	if err != nil {
		return fmt.Errorf("updating peer: %w", err)
	}
	return nil
}

func (s *SQLiteStore) DeletePeer(id string) error {
	_, err := s.db.Exec("DELETE FROM peers WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting peer: %w", err)
	}
	return nil
}

// --- Helpers ---

func scanPeers(rows *sql.Rows) ([]models.Peer, error) {
	var result []models.Peer
	for rows.Next() {
		var peer models.Peer
		var psk, endpoint sql.NullString
		var expiresAt, lastHandshake sql.NullTime

		if err := rows.Scan(
			&peer.ID, &peer.InterfaceID, &peer.Name, &peer.PublicKey,
			&peer.PrivateKeyEncrypted, &psk, &peer.Address, &peer.AllowedIPs, &peer.ClientAllowedIPs, &peer.DNS, &endpoint,
			&peer.PersistentKeepalive, &peer.Enabled,
			&expiresAt, &lastHandshake,
			&peer.TransferRx, &peer.TransferTx,
			&peer.CreatedAt, &peer.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning peer: %w", err)
		}

		if psk.Valid {
			peer.PresharedKeyEncrypted = psk.String
		}
		if endpoint.Valid {
			peer.Endpoint = endpoint.String
		}
		if expiresAt.Valid {
			peer.ExpiresAt = &expiresAt.Time
		}
		if lastHandshake.Valid {
			peer.LastHandshake = &lastHandshake.Time
		}

		result = append(result, peer)
	}
	return result, rows.Err()
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
