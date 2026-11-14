package database

import (
	"errors"
	"testing"

	"aunefyren/poenskelisten/models"

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

	// A code already used can't be consumed again.
	var codes []models.MFARecoveryCode
	if err := Instance.Where("user_id = ? AND used_at IS NOT NULL", user.ID).Find(&codes).Error; err != nil || len(codes) != 1 {
		t.Fatalf("used codes = %d (err %v), want 1", len(codes), err)
	}
	if err := MarkRecoveryCodeUsed(codes[0].ID); !errors.Is(err, ErrRecoveryCodeUsed) {
		t.Errorf("second consume err = %v, want ErrRecoveryCodeUsed", err)
	}
}

// A TOTP step is accepted once, and never after a later one; a new secret
// or turning MFA off starts over.
func TestUseTOTPStep(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)

	for _, c := range []struct {
		step int64
		want bool
	}{{100, true}, {100, false}, {99, false}, {101, true}} {
		if used, err := UseTOTPStep(user.ID, c.step); err != nil || used != c.want {
			t.Errorf("step %d: used = %v (err %v), want %v", c.step, used, err, c.want)
		}
	}
	if err := SetUserPendingMFASecret(user.ID, "new-secret"); err != nil {
		t.Fatal(err)
	}
	if used, err := UseTOTPStep(user.ID, 50); err != nil || !used {
		t.Errorf("after a new secret: used = %v (err %v), want true", used, err)
	}
	if err := DisableUserMFA(user.ID); err != nil {
		t.Fatal(err)
	}
	if used, err := UseTOTPStep(user.ID, 10); err != nil || !used {
		t.Errorf("after disabling: used = %v (err %v), want true", used, err)
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

// pendingChallenge reads the user's stored challenge ID back.
func pendingChallenge(t *testing.T, userID uuid.UUID) *uuid.UUID {
	t.Helper()
	user, err := GetAllUserInformation(userID)
	if err != nil {
		t.Fatalf("GetAllUserInformation error: %v", err)
	}
	return user.MFAChallengeID
}

// A newer challenge replaces the older one, and each completes once.
func TestMFAChallengeSetAndConsume(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)
	older, newer := uuid.New(), uuid.New()

	if err := SetMFAChallenge(user.ID, older); err != nil {
		t.Fatalf("SetMFAChallenge error: %v", err)
	}
	if err := SetMFAChallenge(user.ID, newer); err != nil {
		t.Fatalf("SetMFAChallenge error: %v", err)
	}
	if got := pendingChallenge(t, user.ID); got == nil || *got != newer {
		t.Fatalf("stored challenge = %v, want %v", got, newer)
	}
	for _, c := range []struct {
		name string
		id   uuid.UUID
		want bool
	}{{"replaced", older, false}, {"latest", newer, true}, {"completed", newer, false}} {
		if consumed, err := ConsumeMFAChallenge(user.ID, c.id); err != nil || consumed != c.want {
			t.Errorf("%s: consumed = %v (err %v), want %v", c.name, consumed, err, c.want)
		}
	}
	if got := pendingChallenge(t, user.ID); got != nil {
		t.Errorf("stored challenge = %v after completion, want nil", *got)
	}
}

func TestSetMFAChallengeUnknownUser(t *testing.T) {
	setupTestDB(t)
	if err := SetMFAChallenge(uuid.New(), uuid.New()); err == nil {
		t.Error("SetMFAChallenge for an unknown user: want an error")
	}
}

// The challenge and the TOTP step are taken together or not at all: a
// sign-in that loses either leaves the other as it was.
func TestCompleteMFAChallengeWithTOTP(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)
	challenge := uuid.New()
	if err := SetMFAChallenge(user.ID, challenge); err != nil {
		t.Fatal(err)
	}

	// Wrong challenge: the step stays unused.
	if ok, err := CompleteMFAChallengeWithTOTP(user.ID, uuid.New(), 100); err != nil || ok {
		t.Errorf("wrong challenge: ok = %v (err %v), want false", ok, err)
	}
	if used, err := UseTOTPStep(user.ID, 99); err != nil || !used {
		t.Fatalf("UseTOTPStep(99) = %v (err %v), want true", used, err)
	}
	// Used step: the challenge stays pending.
	if ok, err := CompleteMFAChallengeWithTOTP(user.ID, challenge, 99); err != nil || ok {
		t.Errorf("used step: ok = %v (err %v), want false", ok, err)
	}
	if got := pendingChallenge(t, user.ID); got == nil || *got != challenge {
		t.Fatalf("challenge = %v after a used step, want it still pending", got)
	}
	// Both fresh: both taken.
	if ok, err := CompleteMFAChallengeWithTOTP(user.ID, challenge, 100); err != nil || !ok {
		t.Errorf("fresh: ok = %v (err %v), want true", ok, err)
	}
	if got := pendingChallenge(t, user.ID); got != nil {
		t.Errorf("challenge = %v after completion, want nil", *got)
	}
	if used, err := UseTOTPStep(user.ID, 100); err != nil || used {
		t.Errorf("step 100 after completion: used = %v (err %v), want false", used, err)
	}
}

func TestClearMFAChallenge(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)
	if err := SetMFAChallenge(user.ID, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if err := ClearMFAChallenge(user.ID); err != nil {
		t.Fatalf("ClearMFAChallenge error: %v", err)
	}
	if got := pendingChallenge(t, user.ID); got != nil {
		t.Errorf("challenge = %v, want nil", *got)
	}
}

// A user record read before a code or challenge was used, saved afterwards,
// doesn't bring either back.
func TestUpdateUserInDBKeepsOneTimeMFAState(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)
	challenge := uuid.New()
	if err := SetMFAChallenge(user.ID, challenge); err != nil {
		t.Fatal(err)
	}
	stale, err := GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := CompleteMFAChallengeWithTOTP(user.ID, challenge, 100); err != nil || !ok {
		t.Fatalf("CompleteMFAChallengeWithTOTP = %v (err %v)", ok, err)
	}

	stale.FirstName = "Renamed"
	if _, err := UpdateUserInDB(stale); err != nil {
		t.Fatalf("UpdateUserInDB error: %v", err)
	}

	saved, err := GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.FirstName != "Renamed" {
		t.Errorf("FirstName = %q, the rest of the record must still save", saved.FirstName)
	}
	if saved.MFAChallengeID != nil {
		t.Errorf("challenge = %v, the completed challenge came back", *saved.MFAChallengeID)
	}
	if saved.MFALastStep == nil || *saved.MFALastStep != 100 {
		t.Errorf("last step = %v, want 100", saved.MFALastStep)
	}
}

func TestMFAOneTimeStateDatabaseFailures(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)
	if err := Instance.Migrator().DropTable(&models.User{}); err != nil {
		t.Fatal(err)
	}
	if _, err := UseTOTPStep(user.ID, 1); err == nil {
		t.Error("UseTOTPStep: want an error")
	}
	if err := SetMFAChallenge(user.ID, uuid.New()); err == nil {
		t.Error("SetMFAChallenge: want an error")
	}
	if _, err := ConsumeMFAChallenge(user.ID, uuid.New()); err == nil {
		t.Error("ConsumeMFAChallenge: want an error")
	}
	if _, err := CompleteMFAChallengeWithTOTP(user.ID, uuid.New(), 1); err == nil {
		t.Error("CompleteMFAChallengeWithTOTP: want an error")
	}
	if err := ClearMFAChallenge(user.ID); err == nil {
		t.Error("ClearMFAChallenge: want an error")
	}
}
