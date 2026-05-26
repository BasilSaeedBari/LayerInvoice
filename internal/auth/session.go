package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// User represents the authenticated operator.
type User struct {
	ID           string
	TenantID     string
	Name         string
	Email        string
	Role         string // owner | employee | viewer
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Tenant represents the multi-company scope.
type Tenant struct {
	ID          string
	Name        string
	Slug        string
	CompanyLogo string // Base64 logo for global UI headers
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SessionStore manages active user sessions.
type SessionStore struct {
	db *sql.DB
}

// NewSessionStore creates a new SessionStore.
func NewSessionStore(db *sql.DB) *SessionStore {
	return &SessionStore{db: db}
}

// CreateSession generates a new session token, persists it in SQLite, and returns the token ID.
func (s *SessionStore) CreateSession(ctx context.Context, userID string, duration time.Duration) (string, error) {
	// Generate ULID for session
	entropy := rand.New(rand.NewSource(time.Now().UnixNano()))
	id, err := ulid.New(ulid.Timestamp(time.Now()), entropy)
	if err != nil {
		id = ulid.Make()
	}
	sessionID := id.String()

	createdAt := time.Now().Format(time.RFC3339)
	expiresAt := time.Now().Add(duration).Format(time.RFC3339)

	query := `INSERT INTO sessions (id, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`
	_, err = s.db.ExecContext(ctx, query, sessionID, userID, expiresAt, createdAt)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return sessionID, nil
}

// ValidateSession reads session token, confirms expiration status, and retrieves associated User & Tenant.
func (s *SessionStore) ValidateSession(ctx context.Context, sessionID string) (*User, *Tenant, error) {
	query := `
		SELECT 
			s.expires_at,
			u.id, u.tenant_id, u.name, u.email, u.role, u.is_active,
			t.id, t.name, t.slug
		FROM sessions s
		JOIN users u ON s.user_id = u.id
		JOIN tenants t ON u.tenant_id = t.id
		WHERE s.id = ?
	`

	var expiresAtStr string
	var u User
	var t Tenant
	var isActiveInt int

	err := s.db.QueryRowContext(ctx, query, sessionID).Scan(
		&expiresAtStr,
		&u.ID, &u.TenantID, &u.Name, &u.Email, &u.Role, &isActiveInt,
		&t.ID, &t.Name, &t.Slug,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, errors.New("session not found")
		}
		return nil, nil, fmt.Errorf("failed to query session: %w", err)
	}

	u.IsActive = isActiveInt == 1

	// Query logo setting from db for header render
	_ = s.db.QueryRowContext(ctx, "SELECT value FROM settings WHERE tenant_id = ? AND key = 'company_logo'", t.ID).Scan(&t.CompanyLogo)

	// Parse expiration timestamp
	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse session expiry: %w", err)
	}

	// Check if session has expired
	if time.Now().After(expiresAt) {
		// Clean up expired session
		_ = s.DeleteSession(ctx, sessionID)
		return nil, nil, errors.New("session expired")
	}

	if !u.IsActive {
		return nil, nil, errors.New("user account is inactive")
	}

	return &u, &t, nil
}

// DeleteSession removes a session from SQLite.
func (s *SessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	query := `DELETE FROM sessions WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}
