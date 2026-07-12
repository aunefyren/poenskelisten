package database

import (
	"testing"

	"github.com/google/uuid"
)

func TestGetUserInformationRedactsAndFiltersEnabled(t *testing.T) {
	setupTestDB(t)

	user := createTestUser(t)

	got, err := GetUserInformation(user.ID)
	if err != nil {
		t.Fatalf("GetUserInformation returned error: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("expected user %v, got %v", user.ID, got.ID)
	}

	// GetUserInformation must never leak sensitive fields.
	if got.Password != nil || got.VerificationCode != nil || got.Verified != nil ||
		got.ResetCode != nil || got.ResetExpiration != nil {
		t.Fatalf("expected sensitive fields to be redacted, got %+v", got)
	}

	// A disabled user must not be found by the enabled-only lookup.
	user.Enabled = boolPtr(false)
	if _, err := UpdateUserInDB(user); err != nil {
		t.Fatalf("failed to disable user: %v", err)
	}
	if _, err := GetUserInformation(user.ID); err == nil {
		t.Fatalf("expected error looking up disabled user, got nil")
	}
}

func TestGetAllUserInformationKeepsSensitiveFields(t *testing.T) {
	setupTestDB(t)

	user := createTestUser(t)

	got, err := GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatalf("GetAllUserInformation returned error: %v", err)
	}
	if got.Password == nil || *got.Password != "hashed-password" {
		t.Fatalf("expected password to be retained, got %v", got.Password)
	}
}

func TestAnyStateLookupsFindDisabledUsers(t *testing.T) {
	setupTestDB(t)

	user := createTestUser(t)
	user.Enabled = boolPtr(false)
	if _, err := UpdateUserInDB(user); err != nil {
		t.Fatalf("failed to disable user: %v", err)
	}

	// The enabled-only lookup must not find the disabled user...
	if _, err := GetUserInformation(user.ID); err == nil {
		t.Fatalf("expected enabled-only lookup to fail for disabled user")
	}

	// ...but the AnyState variants must, redacted and non-redacted respectively.
	redacted, err := GetUserInformationAnyState(user.ID)
	if err != nil {
		t.Fatalf("GetUserInformationAnyState returned error: %v", err)
	}
	if redacted.ID != user.ID || redacted.Password != nil {
		t.Fatalf("expected redacted disabled user, got %+v", redacted)
	}

	full, err := GetAllUserInformationAnyState(user.ID)
	if err != nil {
		t.Fatalf("GetAllUserInformationAnyState returned error: %v", err)
	}
	if full.Password == nil || *full.Password != "hashed-password" {
		t.Fatalf("expected password retained in AnyState full lookup")
	}
}

func TestGetUserInformationByEmail(t *testing.T) {
	setupTestDB(t)

	user := createTestUser(t)

	got, err := GetUserInformationByEmail(*user.Email)
	if err != nil {
		t.Fatalf("GetUserInformationByEmail returned error: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("expected user %v, got %v", user.ID, got.ID)
	}
	// Redacted variant must not leak the password.
	if got.Password != nil {
		t.Fatalf("expected redacted user from GetUserInformationByEmail")
	}

	// The full-information variant keeps sensitive fields.
	full, err := GetAllUserInformationByEmail(*user.Email)
	if err != nil {
		t.Fatalf("GetAllUserInformationByEmail returned error: %v", err)
	}
	if full.Password == nil {
		t.Fatalf("expected password retained in GetAllUserInformationByEmail")
	}

	if _, err := GetUserInformationByEmail("missing@example.com"); err == nil {
		t.Fatalf("expected error for unknown e-mail, got nil")
	}
}

func TestVerifyUniqueUserEmail(t *testing.T) {
	setupTestDB(t)

	user := createTestUser(t)

	unique, err := VerifyUniqueUserEmail("fresh@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !unique {
		t.Fatalf("expected unused e-mail to be unique")
	}

	taken, err := VerifyUniqueUserEmail(*user.Email)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if taken {
		t.Fatalf("expected existing e-mail to be reported as not unique")
	}
}

func TestGetAmountOfEnabledUsersAndListings(t *testing.T) {
	setupTestDB(t)

	createTestUser(t)
	createTestUser(t)
	disabled := createTestUser(t)
	disabled.Enabled = boolPtr(false)
	if _, err := UpdateUserInDB(disabled); err != nil {
		t.Fatalf("failed to disable user: %v", err)
	}

	count, err := GetAmountOfEnabledUsers()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 enabled users, got %d", count)
	}

	enabled, err := GetEnabledUsers()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(enabled) != 2 {
		t.Fatalf("expected 2 enabled users listed, got %d", len(enabled))
	}
	// Listings must be redacted.
	for _, u := range enabled {
		if u.Password != nil {
			t.Fatalf("expected redacted password in listing")
		}
	}

	all, err := GetAllUsers()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 users including disabled, got %d", len(all))
	}
}

func TestGenerateAndLookupResetCode(t *testing.T) {
	setupTestDB(t)

	user := createTestUser(t)

	code, err := GenerateRandomResetCodeForUser(user.ID, true)
	if err != nil {
		t.Fatalf("GenerateRandomResetCodeForUser returned error: %v", err)
	}
	if code == "" {
		t.Fatalf("expected a non-empty reset code")
	}

	got, err := GetAllUserInformationByResetCode(code)
	if err != nil {
		t.Fatalf("GetAllUserInformationByResetCode returned error: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("expected user %v, got %v", user.ID, got.ID)
	}
	if got.ResetExpiration == nil {
		t.Fatalf("expected a reset expiration to be set")
	}
}

func TestGenerateResetCodeForMissingUser(t *testing.T) {
	setupTestDB(t)

	if _, err := GenerateRandomResetCodeForUser(uuid.New(), true); err == nil {
		t.Fatalf("expected error generating reset code for unknown user, got nil")
	}
}
