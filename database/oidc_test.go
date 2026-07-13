package database

import (
	"aunefyren/poenskelisten/models"
	"errors"
	"testing"
)

const testIssuer = "https://auth.example.com"

func TestResolveOIDCUserByExistingLink(t *testing.T) {
	setupTestDB(t)

	// Auto-create a user, then resolve the same subject again: it must return the
	// same account without creating a duplicate.
	created, err := ResolveOIDCUser(testIssuer, "sub-1", "sso@example.com", "Ada", "Lovelace", true, true)
	if err != nil {
		t.Fatalf("initial ResolveOIDCUser error: %v", err)
	}

	again, err := ResolveOIDCUser(testIssuer, "sub-1", "sso@example.com", "Ada", "Lovelace", true, true)
	if err != nil {
		t.Fatalf("second ResolveOIDCUser error: %v", err)
	}
	if again.ID != created.ID {
		t.Errorf("resolved a different user on repeat: %v vs %v", again.ID, created.ID)
	}

	users, err := GetAllUsers()
	if err != nil {
		t.Fatalf("GetAllUsers error: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("expected exactly 1 user, got %d", len(users))
	}
}

func TestResolveOIDCUserLinksVerifiedEmail(t *testing.T) {
	setupTestDB(t)

	local := createTestUser(t) // enabled, verified, has password

	resolved, err := ResolveOIDCUser(testIssuer, "sub-2", *local.Email, "Test", "User", true, false)
	if err != nil {
		t.Fatalf("ResolveOIDCUser error: %v", err)
	}
	if resolved.ID != local.ID {
		t.Errorf("linked to a different account: %v vs %v", resolved.ID, local.ID)
	}

	// The subject is now linked; the existing password must remain (still local).
	stored, err := GetAllUserInformation(local.ID)
	if err != nil {
		t.Fatalf("GetAllUserInformation error: %v", err)
	}
	if stored.OIDCSubject == nil || *stored.OIDCSubject != "sub-2" {
		t.Errorf("OIDCSubject = %v, want 'sub-2'", stored.OIDCSubject)
	}
	if !stored.HasPassword() {
		t.Error("linking cleared the local password, want it kept")
	}
	if !stored.IsLocalAuth() {
		t.Error("linked account should still be treated as local auth")
	}
}

func TestResolveOIDCUserRefusesUnverifiedEmailLink(t *testing.T) {
	setupTestDB(t)

	local := createTestUser(t)

	_, err := ResolveOIDCUser(testIssuer, "sub-3", *local.Email, "Test", "User", false, false)
	if !errors.Is(err, ErrOIDCEmailNotVerified) {
		t.Errorf("error = %v, want ErrOIDCEmailNotVerified", err)
	}
}

func TestResolveOIDCUserAutoCreateOff(t *testing.T) {
	setupTestDB(t)

	// Seed a user so this isn't the first-account case, then attempt an unknown
	// OIDC login with auto-create disabled.
	_ = createTestUser(t)

	_, err := ResolveOIDCUser(testIssuer, "sub-4", "new@example.com", "New", "Person", true, false)
	if !errors.Is(err, ErrOIDCUserNotFound) {
		t.Errorf("error = %v, want ErrOIDCUserNotFound", err)
	}
}

func TestResolveOIDCUserAutoCreateOn(t *testing.T) {
	setupTestDB(t)

	_ = createTestUser(t) // ensure the new account is not the first (not admin)

	created, err := ResolveOIDCUser(testIssuer, "sub-5", "new@example.com", "New", "Person", true, true)
	if err != nil {
		t.Fatalf("ResolveOIDCUser error: %v", err)
	}

	stored, err := GetAllUserInformation(created.ID)
	if err != nil {
		t.Fatalf("GetAllUserInformation error: %v", err)
	}
	if stored.AuthSource == nil || *stored.AuthSource != models.AuthSourceOIDC {
		t.Errorf("AuthSource = %v, want oidc", stored.AuthSource)
	}
	if stored.IsLocalAuth() {
		t.Error("auto-created OIDC user reported as local auth")
	}
	if stored.HasPassword() {
		t.Error("auto-created OIDC user should have no local password")
	}
	if stored.Verified == nil || !*stored.Verified {
		t.Error("auto-created OIDC user should be pre-verified")
	}
	if stored.Admin {
		t.Error("second account should not be admin")
	}
}

func TestResolveOIDCUserAutoCreateRequiresVerifiedEmail(t *testing.T) {
	setupTestDB(t)

	_, err := ResolveOIDCUser(testIssuer, "sub-6", "unverified@example.com", "N", "P", false, true)
	if !errors.Is(err, ErrOIDCEmailNotVerified) {
		t.Errorf("error = %v, want ErrOIDCEmailNotVerified", err)
	}
}

func TestResolveOIDCUserRequiresEmail(t *testing.T) {
	setupTestDB(t)

	_, err := ResolveOIDCUser(testIssuer, "sub-7", "   ", "N", "P", true, true)
	if !errors.Is(err, ErrOIDCNoEmail) {
		t.Errorf("error = %v, want ErrOIDCNoEmail", err)
	}
}

func TestResolveOIDCUserFirstAccountBecomesAdmin(t *testing.T) {
	setupTestDB(t)

	created, err := ResolveOIDCUser(testIssuer, "sub-8", "first@example.com", "First", "Admin", true, true)
	if err != nil {
		t.Fatalf("ResolveOIDCUser error: %v", err)
	}
	if !created.Admin {
		t.Error("first auto-created account should be admin")
	}
}
