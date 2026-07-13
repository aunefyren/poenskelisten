package database

import (
	"testing"
)

func TestMFAEnrollmentLifecycle(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)

	// Fresh user: MFA disabled, treated as a local account.
	enabled, isLocal, err := GetUserMFAEnrollmentState(user.ID)
	if err != nil {
		t.Fatalf("GetUserMFAEnrollmentState error: %v", err)
	}
	if enabled {
		t.Error("new user reported MFA enabled")
	}
	if !isLocal {
		t.Error("new user reported as non-local account")
	}

	// Store a pending secret.
	if err := SetUserPendingMFASecret(user.ID, "encrypted-secret"); err != nil {
		t.Fatalf("SetUserPendingMFASecret error: %v", err)
	}
	stored, err := GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatalf("GetAllUserInformation error: %v", err)
	}
	if stored.MFASecret == nil || *stored.MFASecret != "encrypted-secret" {
		t.Errorf("MFASecret = %v, want 'encrypted-secret'", stored.MFASecret)
	}
	if stored.IsMFAEnabled() {
		t.Error("MFA reported enabled while still pending")
	}

	// Activate MFA.
	if err := ActivateUserMFA(user.ID); err != nil {
		t.Fatalf("ActivateUserMFA error: %v", err)
	}
	stored, err = GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatalf("GetAllUserInformation error: %v", err)
	}
	if !stored.IsMFAEnabled() {
		t.Error("MFA not enabled after activation")
	}
	if stored.MFAEnrolledAt == nil {
		t.Error("MFAEnrolledAt not set after activation")
	}

	enabled, _, err = GetUserMFAEnrollmentState(user.ID)
	if err != nil {
		t.Fatalf("GetUserMFAEnrollmentState error: %v", err)
	}
	if !enabled {
		t.Error("GetUserMFAEnrollmentState reports disabled after activation")
	}
}

func TestRecoveryCodeStoreConsumeAndClear(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)

	hashes := []string{"hash-a", "hash-b", "hash-c"}
	if err := StoreRecoveryCodes(user.ID, hashes); err != nil {
		t.Fatalf("StoreRecoveryCodes error: %v", err)
	}

	active, err := GetActiveRecoveryCodes(user.ID)
	if err != nil {
		t.Fatalf("GetActiveRecoveryCodes error: %v", err)
	}
	if len(active) != len(hashes) {
		t.Fatalf("got %d active codes, want %d", len(active), len(hashes))
	}

	// Consume one code.
	if err := MarkRecoveryCodeUsed(active[0].ID); err != nil {
		t.Fatalf("MarkRecoveryCodeUsed error: %v", err)
	}
	active, err = GetActiveRecoveryCodes(user.ID)
	if err != nil {
		t.Fatalf("GetActiveRecoveryCodes error: %v", err)
	}
	if len(active) != len(hashes)-1 {
		t.Errorf("got %d active codes after consume, want %d", len(active), len(hashes)-1)
	}
}

func TestDisableUserMFAClearsState(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)

	if err := SetUserPendingMFASecret(user.ID, "encrypted-secret"); err != nil {
		t.Fatalf("SetUserPendingMFASecret error: %v", err)
	}
	if err := ActivateUserMFA(user.ID); err != nil {
		t.Fatalf("ActivateUserMFA error: %v", err)
	}
	if err := StoreRecoveryCodes(user.ID, []string{"hash-a", "hash-b"}); err != nil {
		t.Fatalf("StoreRecoveryCodes error: %v", err)
	}

	if err := DisableUserMFA(user.ID); err != nil {
		t.Fatalf("DisableUserMFA error: %v", err)
	}

	stored, err := GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatalf("GetAllUserInformation error: %v", err)
	}
	if stored.IsMFAEnabled() {
		t.Error("MFA still enabled after disable")
	}
	if stored.MFASecret != nil {
		t.Errorf("MFASecret = %v, want nil after disable", *stored.MFASecret)
	}
	if stored.MFAEnrolledAt != nil {
		t.Error("MFAEnrolledAt still set after disable")
	}

	active, err := GetActiveRecoveryCodes(user.ID)
	if err != nil {
		t.Fatalf("GetActiveRecoveryCodes error: %v", err)
	}
	if len(active) != 0 {
		t.Errorf("got %d active recovery codes after disable, want 0", len(active))
	}
}

func TestDisableUserMFAIdempotent(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)

	// Disabling MFA for a user that never enrolled must not error.
	if err := DisableUserMFA(user.ID); err != nil {
		t.Errorf("DisableUserMFA on non-enrolled user returned error: %v", err)
	}
}
