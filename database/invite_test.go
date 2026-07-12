package database

import (
	"testing"

	"github.com/google/uuid"
)

func TestGenerateAndVerifyInvite(t *testing.T) {
	setupTestDB(t)

	code, err := GenerateRandomInvite()
	if err != nil {
		t.Fatalf("GenerateRandomInvite returned error: %v", err)
	}
	if code == "" {
		t.Fatalf("expected a non-empty invite code")
	}

	if ok, err := VerifyUnusedUserInviteCode(code); err != nil || !ok {
		t.Fatalf("expected fresh invite to be valid and unused (ok=%v err=%v)", ok, err)
	}

	// Unknown codes are not valid.
	if ok, err := VerifyUnusedUserInviteCode("NOTACODE"); err != nil || ok {
		t.Fatalf("expected unknown code to be invalid (ok=%v err=%v)", ok, err)
	}
}

func TestSetUsedUserInviteCode(t *testing.T) {
	setupTestDB(t)

	claimer := createTestUser(t)
	code, err := GenerateRandomInvite()
	if err != nil {
		t.Fatalf("GenerateRandomInvite returned error: %v", err)
	}

	if err := SetUsedUserInviteCode(code, claimer.ID); err != nil {
		t.Fatalf("SetUsedUserInviteCode returned error: %v", err)
	}

	// Once used, it must no longer verify as unused.
	if ok, err := VerifyUnusedUserInviteCode(code); err != nil || ok {
		t.Fatalf("expected used invite to be invalid (ok=%v err=%v)", ok, err)
	}
}

func TestGetAndDeleteInvite(t *testing.T) {
	setupTestDB(t)

	if _, err := GenerateRandomInvite(); err != nil {
		t.Fatalf("GenerateRandomInvite returned error: %v", err)
	}

	invites, err := GetAllEnabledInvites()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(invites) != 1 {
		t.Fatalf("expected 1 enabled invite, got %d", len(invites))
	}

	invite := invites[0]
	got, err := GetInviteByID(invite.ID)
	if err != nil {
		t.Fatalf("GetInviteByID returned error: %v", err)
	}
	if got.ID != invite.ID {
		t.Fatalf("expected invite %v, got %v", invite.ID, got.ID)
	}

	if err := DeleteInviteByID(invite.ID); err != nil {
		t.Fatalf("DeleteInviteByID returned error: %v", err)
	}

	// Disabled invites disappear from the enabled listing and from lookup.
	remaining, err := GetAllEnabledInvites()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected no enabled invites after delete, got %d", len(remaining))
	}
	if _, err := GetInviteByID(invite.ID); err == nil {
		t.Fatalf("expected error looking up disabled invite, got nil")
	}

	// Deleting an unknown invite fails (RowsAffected != 1).
	if err := DeleteInviteByID(uuid.New()); err == nil {
		t.Fatalf("expected error deleting unknown invite, got nil")
	}
}
