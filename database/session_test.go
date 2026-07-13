package database

import (
	"errors"
	"testing"
	"time"
)

func TestCreateAndRotateSession(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)

	original, err := CreateSession(user.ID, "hash-1", "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}
	if original.RevokedAt != nil {
		t.Error("new session should not be revoked")
	}

	result, err := RotateSession("hash-1", "hash-2", "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("RotateSession error: %v", err)
	}
	if !result.Rotated {
		t.Error("expected Rotated=true for an active session")
	}
	if result.UserID != user.ID {
		t.Errorf("UserID = %v, want %v", result.UserID, user.ID)
	}

	// Old session is now revoked+rotated; new session is active.
	old, found, err := getSessionByRefreshHash("hash-1")
	if err != nil || !found {
		t.Fatalf("lookup old session: found=%v err=%v", found, err)
	}
	if old.RevokedAt == nil || old.RotatedAt == nil || old.ReplacedByID == nil {
		t.Error("old session should be revoked, rotated, and linked to its replacement")
	}

	newer, found, err := getSessionByRefreshHash("hash-2")
	if err != nil || !found {
		t.Fatalf("lookup new session: found=%v err=%v", found, err)
	}
	if newer.RevokedAt != nil {
		t.Error("replacement session should be active")
	}
}

func TestRotateSessionGraceWindow(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)

	if _, err := CreateSession(user.ID, "hash-1", "a", "ip"); err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}
	if _, err := RotateSession("hash-1", "hash-2", "a", "ip"); err != nil {
		t.Fatalf("first RotateSession error: %v", err)
	}

	// Immediately presenting the just-rotated token again (a multi-tab race) is
	// tolerated: no rotation, no revoke-all.
	result, err := RotateSession("hash-1", "hash-3", "a", "ip")
	if err != nil {
		t.Fatalf("grace RotateSession error: %v", err)
	}
	if result.Rotated {
		t.Error("grace-window refresh should not rotate")
	}
	if result.UserID != user.ID {
		t.Errorf("UserID = %v, want %v", result.UserID, user.ID)
	}

	// The replacement session must still be active (not revoked by the race).
	newer, found, err := getSessionByRefreshHash("hash-2")
	if err != nil || !found {
		t.Fatalf("lookup replacement: found=%v err=%v", found, err)
	}
	if newer.RevokedAt != nil {
		t.Error("grace-window refresh must not revoke the live session")
	}
}

func TestRotateSessionReuseAfterGraceRevokesAll(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)

	if _, err := CreateSession(user.ID, "hash-1", "a", "ip"); err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}
	if _, err := RotateSession("hash-1", "hash-2", "a", "ip"); err != nil {
		t.Fatalf("RotateSession error: %v", err)
	}

	// Age the rotation past the grace window.
	old, _, _ := getSessionByRefreshHash("hash-1")
	past := time.Now().Add(-1 * time.Minute)
	old.RotatedAt = &past
	if err := Instance.Save(&old).Error; err != nil {
		t.Fatalf("failed to age rotation: %v", err)
	}

	// Presenting the old token now is treated as theft.
	_, err := RotateSession("hash-1", "hash-9", "a", "ip")
	if !errors.Is(err, ErrSessionReused) {
		t.Errorf("error = %v, want ErrSessionReused", err)
	}

	// Every session for the user is revoked, including the previously-live one.
	newer, _, _ := getSessionByRefreshHash("hash-2")
	if newer.RevokedAt == nil {
		t.Error("reuse detection should have revoked the live session too")
	}
}

func TestRotateSessionExpired(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)

	session, err := CreateSession(user.ID, "hash-1", "a", "ip")
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}
	session.ExpiresAt = time.Now().Add(-1 * time.Hour)
	if err := Instance.Save(&session).Error; err != nil {
		t.Fatalf("failed to expire session: %v", err)
	}

	if _, err := RotateSession("hash-1", "hash-2", "a", "ip"); !errors.Is(err, ErrSessionExpired) {
		t.Errorf("error = %v, want ErrSessionExpired", err)
	}
}

func TestRotateSessionAfterLogout(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)

	if _, err := CreateSession(user.ID, "hash-1", "a", "ip"); err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}
	if err := RevokeSessionByRefreshHash("hash-1"); err != nil {
		t.Fatalf("RevokeSessionByRefreshHash error: %v", err)
	}

	// A logged-out (revoked, never rotated) token is simply rejected.
	if _, err := RotateSession("hash-1", "hash-2", "a", "ip"); !errors.Is(err, ErrSessionRevoked) {
		t.Errorf("error = %v, want ErrSessionRevoked", err)
	}
}

func TestRotateSessionNotFound(t *testing.T) {
	setupTestDB(t)

	if _, err := RotateSession("nope", "hash-2", "a", "ip"); !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("error = %v, want ErrSessionNotFound", err)
	}
}

func TestRevokeUserClientSessions(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)

	if _, err := CreateOAuthRefreshSession(user.ID, "h-a", "client-a", "openid", "res", "ua", "ip"); err != nil {
		t.Fatalf("CreateOAuthRefreshSession error: %v", err)
	}
	if _, err := CreateOAuthRefreshSession(user.ID, "h-b", "client-b", "openid", "res", "ua", "ip"); err != nil {
		t.Fatalf("CreateOAuthRefreshSession error: %v", err)
	}

	if err := RevokeUserClientSessions(user.ID, "client-a"); err != nil {
		t.Fatalf("RevokeUserClientSessions error: %v", err)
	}

	a, _, _ := getSessionByRefreshHash("h-a")
	if a.RevokedAt == nil {
		t.Error("client-a session should be revoked")
	}
	b, _, _ := getSessionByRefreshHash("h-b")
	if b.RevokedAt != nil {
		t.Error("client-b session should remain active")
	}
}

func TestRevokeAllUserSessions(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)

	if _, err := CreateSession(user.ID, "hash-1", "a", "ip"); err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}
	if _, err := CreateSession(user.ID, "hash-2", "a", "ip"); err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}

	if err := RevokeAllUserSessions(user.ID); err != nil {
		t.Fatalf("RevokeAllUserSessions error: %v", err)
	}

	for _, hash := range []string{"hash-1", "hash-2"} {
		s, _, _ := getSessionByRefreshHash(hash)
		if s.RevokedAt == nil {
			t.Errorf("session %s should be revoked", hash)
		}
	}
}
