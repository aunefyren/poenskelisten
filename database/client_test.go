package database

import (
	"testing"
)

func TestUserVerificationFlow(t *testing.T) {
	setupTestDB(t)

	user := createTestUser(t)

	// A freshly created user has no verification code.
	if has, err := VerifyUserHasVerificationCode(user.ID); err != nil || has {
		t.Fatalf("expected no verification code initially (has=%v err=%v)", has, err)
	}

	code, err := GenerateRandomVerificationCodeForUser(user.ID)
	if err != nil {
		t.Fatalf("GenerateRandomVerificationCodeForUser returned error: %v", err)
	}
	if code == "" {
		t.Fatalf("expected a non-empty verification code")
	}

	if has, err := VerifyUserHasVerificationCode(user.ID); err != nil || !has {
		t.Fatalf("expected a verification code to be set (has=%v err=%v)", has, err)
	}

	// The right code matches; a wrong one does not.
	if ok, err := VerifyUserVerificationCodeMatches(user.ID, code); err != nil || !ok {
		t.Fatalf("expected matching code to verify (ok=%v err=%v)", ok, err)
	}
	if ok, err := VerifyUserVerificationCodeMatches(user.ID, "WRONG"); err != nil || ok {
		t.Fatalf("expected wrong code to fail (ok=%v err=%v)", ok, err)
	}
}

func TestSetUserVerification(t *testing.T) {
	setupTestDB(t)

	user := createTestUser(t)

	// createTestUser marks the user verified; flip it off and confirm.
	if err := SetUserVerification(user.ID, false); err != nil {
		t.Fatalf("SetUserVerification returned error: %v", err)
	}
	if verified, err := VerifyUserIsVerified(user.ID); err != nil || verified {
		t.Fatalf("expected user to be unverified (verified=%v err=%v)", verified, err)
	}

	if err := SetUserVerification(user.ID, true); err != nil {
		t.Fatalf("SetUserVerification returned error: %v", err)
	}
	if verified, err := VerifyUserIsVerified(user.ID); err != nil || !verified {
		t.Fatalf("expected user to be verified (verified=%v err=%v)", verified, err)
	}
}
