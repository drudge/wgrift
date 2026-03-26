package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/drudge/wgrift/internal/models"
	"github.com/google/uuid"
)

// --- Users ---

func (s *SQLiteStore) CreateUser(user *models.User) error {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := s.db.Exec(`
		INSERT INTO users (id, username, password_hash, display_name, role, is_initial, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Username, user.PasswordHash, user.DisplayName,
		user.Role, user.IsInitial, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting user: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetUser(id string) (*models.User, error) {
	user := &models.User{}
	err := s.db.QueryRow(`
		SELECT id, username, password_hash, display_name, role, is_initial, created_at, updated_at
		FROM users WHERE id = ?`, id,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.DisplayName,
		&user.Role, &user.IsInitial, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying user: %w", err)
	}
	return user, nil
}

func (s *SQLiteStore) GetUserByUsername(username string) (*models.User, error) {
	user := &models.User{}
	err := s.db.QueryRow(`
		SELECT id, username, password_hash, display_name, role, is_initial, created_at, updated_at
		FROM users WHERE username = ?`, username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.DisplayName,
		&user.Role, &user.IsInitial, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying user by username: %w", err)
	}
	return user, nil
}

func (s *SQLiteStore) ListUsers() ([]models.User, error) {
	rows, err := s.db.Query(`
		SELECT id, username, password_hash, display_name, role, is_initial, created_at, updated_at
		FROM users ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("querying users: %w", err)
	}
	defer rows.Close()

	var result []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.DisplayName,
			&user.Role, &user.IsInitial, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning user: %w", err)
		}
		result = append(result, user)
	}
	return result, rows.Err()
}

func (s *SQLiteStore) UpdateUser(user *models.User) error {
	user.UpdatedAt = time.Now().UTC()
	_, err := s.db.Exec(`
		UPDATE users SET username=?, password_hash=?, display_name=?, role=?, is_initial=?, updated_at=?
		WHERE id=?`,
		user.Username, user.PasswordHash, user.DisplayName,
		user.Role, user.IsInitial, user.UpdatedAt, user.ID,
	)
	if err != nil {
		return fmt.Errorf("updating user: %w", err)
	}
	return nil
}

func (s *SQLiteStore) DeleteUser(id string) error {
	_, err := s.db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return nil
}

func (s *SQLiteStore) CountUsers() (int, error) {
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return 0, fmt.Errorf("counting users: %w", err)
	}
	return count, nil
}

// --- Sessions ---

func (s *SQLiteStore) CreateSession(session *models.Session) error {
	now := time.Now().UTC()
	session.CreatedAt = now
	session.LastSeenAt = now

	_, err := s.db.Exec(`
		INSERT INTO sessions (id, user_id, csrf_token, expires_at, created_at, last_seen_at, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		session.ID, session.UserID, session.CSRFToken, session.ExpiresAt,
		session.CreatedAt, session.LastSeenAt, session.IPAddress, session.UserAgent,
	)
	if err != nil {
		return fmt.Errorf("inserting session: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetSession(id string) (*models.Session, error) {
	session := &models.Session{}
	err := s.db.QueryRow(`
		SELECT id, user_id, csrf_token, expires_at, created_at, last_seen_at, ip_address, user_agent
		FROM sessions WHERE id = ?`, id,
	).Scan(&session.ID, &session.UserID, &session.CSRFToken, &session.ExpiresAt,
		&session.CreatedAt, &session.LastSeenAt, &session.IPAddress, &session.UserAgent)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying session: %w", err)
	}
	return session, nil
}

func (s *SQLiteStore) TouchSession(id string) error {
	_, err := s.db.Exec("UPDATE sessions SET last_seen_at = ? WHERE id = ?", time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("touching session: %w", err)
	}
	return nil
}

func (s *SQLiteStore) DeleteSession(id string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}

func (s *SQLiteStore) DeleteExpiredSessions() error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now().UTC())
	if err != nil {
		return fmt.Errorf("deleting expired sessions: %w", err)
	}
	return nil
}

func (s *SQLiteStore) DeleteUserSessions(userID string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("deleting user sessions: %w", err)
	}
	return nil
}
