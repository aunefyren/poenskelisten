package controllers

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"strings"
	"testing"
)

func TestIssuePasswordResetLink(t *testing.T) {
	restoreConfig(t)
	setupControllersDB(t)
	originalURL := config.ConfigFile.PoenskelistenExternalURL
	t.Cleanup(func() { config.ConfigFile.PoenskelistenExternalURL = originalURL })
	config.ConfigFile.PoenskelistenExternalURL = "https://wish.example.com"

	user := createTestUser(t)

	// Operators may not know the capitalisation used at registration.
	got, link, err := IssuePasswordResetLink(strings.ToUpper(*user.Email))
	if err != nil {
		t.Fatalf("IssuePasswordResetLink failed: %v", err)
	}
	if got.ID != user.ID {
		t.Errorf("user = %v, want %v", got.ID, user.ID)
	}

	reloaded, err := database.GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if reloaded.ResetCode == nil || link != "https://wish.example.com/login?reset_code="+*reloaded.ResetCode {
		t.Errorf("link = %q, want it to carry the stored reset code", link)
	}

	// The link must be usable with the normal reset flow, which rejects expired codes.
	byCode, err := database.GetAllUserInformationByResetCode(*reloaded.ResetCode)
	if err != nil || byCode.ID != user.ID || byCode.ResetExpiration == nil || !byCode.ResetExpiration.After(reloaded.CreatedAt) {
		t.Errorf("reset code lookup = %v (expiration %v) err %v, want the user with a future expiry", byCode.ID, byCode.ResetExpiration, err)
	}
}

func TestIssuePasswordResetLinkUnknownUser(t *testing.T) {
	setupControllersDB(t)

	if _, _, err := IssuePasswordResetLink("nobody@example.com"); err == nil {
		t.Error("expected an error for an unknown e-mail")
	}
}

func TestResetUserMFA(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	user.MFAEnabled = boolPtr(true)
	user, err := database.UpdateUserInDB(user)
	if err != nil {
		t.Fatalf("failed to enable MFA: %v", err)
	}
	if err := database.StoreRecoveryCodes(user.ID, []string{"hash-1", "hash-2"}); err != nil {
		t.Fatalf("failed to store recovery codes: %v", err)
	}

	if _, err := ResetUserMFA(strings.ToUpper(*user.Email)); err != nil {
		t.Fatalf("ResetUserMFA failed: %v", err)
	}

	reloaded, err := database.GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if reloaded.IsMFAEnabled() {
		t.Error("expected MFA to be disabled")
	}
	codes, err := database.GetActiveRecoveryCodes(user.ID)
	if err != nil || len(codes) != 0 {
		t.Errorf("recovery codes = %d err %v, want none left", len(codes), err)
	}
}

func TestResetUserMFAUnknownUser(t *testing.T) {
	setupControllersDB(t)

	if _, err := ResetUserMFA("nobody@example.com"); err == nil {
		t.Error("expected an error for an unknown e-mail")
	}
}

func TestIssuePasswordResetLinkStoreFails(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	failDBOperation(t, "update", "users", 0)

	_, link, err := IssuePasswordResetLink(*user.Email)
	if err == nil || err.Error() != "Failed to generate reset code." || link != "" {
		t.Errorf("link = %q err = %v, want 'Failed to generate reset code.'", link, err)
	}
}

func TestResetUserMFAStoreFails(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	user.MFAEnabled = boolPtr(true)
	user, err := database.UpdateUserInDB(user)
	if err != nil {
		t.Fatalf("failed to enable MFA: %v", err)
	}
	failDBOperation(t, "update", "users", 0)

	if _, err := ResetUserMFA(*user.Email); err == nil || err.Error() != "Failed to disable MFA." {
		t.Errorf("err = %v, want 'Failed to disable MFA.'", err)
	}
	reloaded, err := database.GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if !reloaded.IsMFAEnabled() {
		t.Error("MFA should still be enabled after a failed reset")
	}
}
