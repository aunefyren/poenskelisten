package models

import (
	"time"

	"github.com/google/uuid"
)

// Session Kind values distinguish first-party web refresh tokens from OAuth
// client refresh tokens (the SSO login-state is a stateless HS256 cookie, not a
// Session row).
const (
	SessionKindWebRefresh   = "web_refresh"
	SessionKindOAuthRefresh = "oauth_refresh"
)

// Session is a server-side refresh-token record. The opaque refresh token itself
// is never stored; only its SHA-256 hash is kept, so a database leak can't be
// used to mint sessions. Rotation revokes the old row and links it to its
// replacement (ReplacedByID) so a reused token can be detected.
type Session struct {
	GormModel
	UserID           uuid.UUID  `json:"user_id" gorm:"not null; index"`
	RefreshTokenHash string     `json:"-" gorm:"not null; index"`
	IssuedAt         time.Time  `json:"issued_at"`
	ExpiresAt        time.Time  `json:"expires_at"`
	RevokedAt        *time.Time `json:"revoked_at"`
	RotatedAt        *time.Time `json:"rotated_at"`
	ReplacedByID     *uuid.UUID `json:"replaced_by_id"`
	LastUsedAt       *time.Time `json:"last_used_at"`
	UserAgent        string     `json:"user_agent"`
	IP               string     `json:"ip"`

	// OAuth fields. Kind defaults to a first-party web refresh token; OAuth client
	// grants also carry the client, granted scope, and bound resource (audience).
	Kind     string `json:"kind"`
	ClientID string `json:"client_id"`
	Scope    string `json:"scope"`
	Resource string `json:"resource"`
}
