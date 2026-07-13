package database

import (
	"aunefyren/poenskelisten/models"
	"errors"
	"time"

	"github.com/google/uuid"
)

// RefreshTokenValidDuration is how long a refresh token (and thus a session) is
// valid before the user must log in again.
const RefreshTokenValidDuration = 7 * 24 * time.Hour

// RotationGraceWindow tolerates a just-rotated refresh token being presented
// again for a short period, so two browser tabs racing to refresh don't trip
// reuse detection and log the user out everywhere.
const RotationGraceWindow = 10 * time.Second

// Sentinel errors from RotateSession.
var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
	ErrSessionRevoked  = errors.New("session revoked")
	// ErrSessionReused means a rotated token was presented after the grace window;
	// all of the user's sessions have been revoked as a precaution.
	ErrSessionReused = errors.New("refresh token reuse detected")
)

// RefreshResult is the outcome of a successful RotateSession call. Scope /
// Resource / ClientID are carried from the rotated session so the caller can mint
// a new access token with the same binding.
type RefreshResult struct {
	UserID uuid.UUID
	// Rotated is true when a new session was created and the caller should set the
	// new refresh cookie. It is false in the grace-window case, where the caller
	// should only issue a fresh access token and leave the refresh cookie as-is.
	Rotated  bool
	Scope    string
	Resource string
	ClientID string
}

// CreateSession persists a new first-party (web) refresh-token session.
func CreateSession(userID uuid.UUID, refreshHash string, userAgent string, ip string) (models.Session, error) {
	return createSession(models.Session{
		UserID:           userID,
		RefreshTokenHash: refreshHash,
		Kind:             models.SessionKindWebRefresh,
		UserAgent:        userAgent,
		IP:               ip,
	})
}

// CreateOAuthRefreshSession persists a refresh-token session bound to an OAuth
// client, granted scope, and target resource (audience).
func CreateOAuthRefreshSession(userID uuid.UUID, refreshHash string, clientID string, scope string, resource string, userAgent string, ip string) (models.Session, error) {
	return createSession(models.Session{
		UserID:           userID,
		RefreshTokenHash: refreshHash,
		Kind:             models.SessionKindOAuthRefresh,
		ClientID:         clientID,
		Scope:            scope,
		Resource:         resource,
		UserAgent:        userAgent,
		IP:               ip,
	})
}

// createSession fills in the timestamps and persists a session.
func createSession(session models.Session) (models.Session, error) {
	now := time.Now()
	session.IssuedAt = now
	session.ExpiresAt = now.Add(RefreshTokenValidDuration)
	session.LastUsedAt = &now
	session.ID = uuid.New()

	record := Instance.Create(&session)
	if record.Error != nil {
		return models.Session{}, record.Error
	}
	return session, nil
}

// getSessionByRefreshHash finds a session by its stored token hash.
func getSessionByRefreshHash(hash string) (models.Session, bool, error) {
	var session models.Session
	record := Instance.
		Where(&models.Session{RefreshTokenHash: hash}).
		Find(&session)
	if record.Error != nil {
		return models.Session{}, false, record.Error
	}
	if record.RowsAffected == 0 {
		return models.Session{}, false, nil
	}
	if record.RowsAffected != 1 {
		return models.Session{}, false, errors.New("multiple sessions share the same refresh hash")
	}
	return session, true, nil
}

// RotateSession validates a presented refresh token (by hash) and, for an active
// session, rotates it: a new session is created with newHash and the old one is
// revoked and linked to its replacement. It enforces expiry, the reuse grace
// window, and theft detection (revoking all of the user's sessions).
func RotateSession(oldHash string, newHash string, userAgent string, ip string) (RefreshResult, error) {
	session, found, err := getSessionByRefreshHash(oldHash)
	if err != nil {
		return RefreshResult{}, err
	}
	if !found {
		return RefreshResult{}, ErrSessionNotFound
	}

	now := time.Now()

	if session.ExpiresAt.Before(now) {
		return RefreshResult{}, ErrSessionExpired
	}

	if session.RevokedAt != nil {
		// A revoked-by-rotation token that reappears is either a benign multi-tab
		// race (within the grace window) or a replayed/stolen token (after it).
		if session.RotatedAt != nil {
			if now.Sub(*session.RotatedAt) <= RotationGraceWindow {
				return RefreshResult{UserID: session.UserID, Rotated: false, Scope: session.Scope, Resource: session.Resource, ClientID: session.ClientID}, nil
			}
			// Reuse after grace: assume theft and revoke everything.
			_ = RevokeAllUserSessions(session.UserID)
			return RefreshResult{}, ErrSessionReused
		}
		// Revoked by an explicit logout: simply reject.
		return RefreshResult{}, ErrSessionRevoked
	}

	// Active session: rotate, carrying the OAuth binding to the new session.
	newSession, err := createSession(models.Session{
		UserID:           session.UserID,
		RefreshTokenHash: newHash,
		Kind:             session.Kind,
		ClientID:         session.ClientID,
		Scope:            session.Scope,
		Resource:         session.Resource,
		UserAgent:        userAgent,
		IP:               ip,
	})
	if err != nil {
		return RefreshResult{}, err
	}

	session.RevokedAt = &now
	session.RotatedAt = &now
	session.ReplacedByID = &newSession.ID
	session.LastUsedAt = &now
	if err := Instance.Save(&session).Error; err != nil {
		return RefreshResult{}, err
	}

	return RefreshResult{UserID: session.UserID, Rotated: true, Scope: session.Scope, Resource: session.Resource, ClientID: session.ClientID}, nil
}

// RevokeSessionByRefreshHash revokes the single session identified by a refresh
// token hash (used on logout). It is idempotent.
func RevokeSessionByRefreshHash(hash string) error {
	now := time.Now()
	record := Instance.
		Model(&models.Session{}).
		Where(&models.Session{RefreshTokenHash: hash}).
		Where("revoked_at IS NULL").
		Update("revoked_at", now)
	return record.Error
}

// RevokeUserClientSessions revokes a user's active refresh sessions for a single
// OAuth client (used when the user disconnects one connected app). Idempotent.
func RevokeUserClientSessions(userID uuid.UUID, clientID string) error {
	now := time.Now()
	record := Instance.
		Model(&models.Session{}).
		Where(&models.Session{UserID: userID, ClientID: clientID}).
		Where("revoked_at IS NULL").
		Update("revoked_at", now)
	return record.Error
}

// RevokeAllUserSessions revokes every active session for a user (logout
// everywhere, admin action, or reuse detection). It is idempotent.
func RevokeAllUserSessions(userID uuid.UUID) error {
	now := time.Now()
	record := Instance.
		Model(&models.Session{}).
		Where(&models.Session{UserID: userID}).
		Where("revoked_at IS NULL").
		Update("revoked_at", now)
	return record.Error
}
