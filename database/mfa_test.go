package database

import (
	"errors"
	"testing"

	"github.com/google/uuid"
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

func TestClearUserRecoveryCodes(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)

	if err := StoreRecoveryCodes(user.ID, []string{"hash-1", "hash-2"}); err != nil {
		t.Fatalf("StoreRecoveryCodes error: %v", err)
	}
	before, err := GetActiveRecoveryCodes(user.ID)
	if err != nil {
		t.Fatalf("GetActiveRecoveryCodes error: %v", err)
	}
	if len(before) != 2 {
		t.Fatalf("got %d recovery codes before clearing, want 2", len(before))
	}

	if err := ClearUserRecoveryCodes(user.ID); err != nil {
		t.Fatalf("ClearUserRecoveryCodes error: %v", err)
	}

	after, err := GetActiveRecoveryCodes(user.ID)
	if err != nil {
		t.Fatalf("GetActiveRecoveryCodes error: %v", err)
	}
	if len(after) != 0 {
		t.Errorf("got %d recovery codes after clearing, want 0", len(after))
	}
}

func TestClearUserRecoveryCodesNoneStored(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)

	if err := ClearUserRecoveryCodes(user.ID); err != nil {
		t.Errorf("ClearUserRecoveryCodes with nothing stored returned error: %v", err)
	}
}

func TestMfaQueriesFailOnClosedDB(t *testing.T) {
	runClosedDBCases(t, map[string]func() error{
		"SetUserPendingMFASecret": func() error {
			return SetUserPendingMFASecret(uuid.New(), "x")
		},
		"ActivateUserMFA": func() error {
			return ActivateUserMFA(uuid.New())
		},
		"DisableUserMFA": func() error {
			return DisableUserMFA(uuid.New())
		},
		"StoreRecoveryCodes": func() error {
			return StoreRecoveryCodes(uuid.New(), []string{"x"})
		},
		"GetActiveRecoveryCodes": func() error {
			_, err := GetActiveRecoveryCodes(uuid.New())
			return err
		},
		"MarkRecoveryCodeUsed": func() error {
			return MarkRecoveryCodeUsed(uuid.New())
		},
		"GetUserMFAEnrollmentState": func() error {
			_, _, err := GetUserMFAEnrollmentState(uuid.New())
			return err
		},
	})
}

func TestMFAWritesForUnknownUserFail(t *testing.T) {
	setupTestDB(t)
	missing := uuid.New()

	if err := SetUserPendingMFASecret(missing, "secret"); err == nil {
		t.Error("SetUserPendingMFASecret: expected error for an unknown user")
	}
	if err := ActivateUserMFA(missing); err == nil {
		t.Error("ActivateUserMFA: expected error for an unknown user")
	}
	if err := MarkRecoveryCodeUsed(missing); err == nil {
		t.Error("MarkRecoveryCodeUsed: expected error for an unknown code")
	}
	if _, _, err := GetUserMFAEnrollmentState(missing); err == nil {
		t.Error("GetUserMFAEnrollmentState: expected error for an unknown user")
	}
}

func TestDisableUserMFARecoveryCodeDeleteFails(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)
	injectFault(t, "delete", "mfa_recovery_codes", 0)

	if err := DisableUserMFA(user.ID); !errors.Is(err, errInjected) {
		t.Fatalf("DisableUserMFA error = %v, want the injected fault", err)
	}
}

func TestStoreRecoveryCodesEmptyIsNoOp(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)

	if err := StoreRecoveryCodes(user.ID, nil); err != nil {
		t.Fatalf("StoreRecoveryCodes(nil) error: %v", err)
	}
	codes, err := GetActiveRecoveryCodes(user.ID)
	if err != nil || len(codes) != 0 {
		t.Fatalf("expected no stored codes, got %d (err %v)", len(codes), err)
	}
}

func TestStoreRecoveryCodesPartialInsertFails(t *testing.T) {
	setupTestDB(t)
	forceRowsAffected(t, "create", "mfa_recovery_codes", 1)

	err := StoreRecoveryCodes(uuid.New(), []string{"a", "b"})
	if err == nil || err.Error() != "not all recovery codes were stored" {
		t.Fatalf("error = %v, want \"not all recovery codes were stored\"", err)
	}
}
