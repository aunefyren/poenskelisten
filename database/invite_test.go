package database

import (
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"errors"
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

func TestCreateUserWithInviteCode(t *testing.T) {
	setupTestDB(t)
	code, err := GenerateRandomInvite()
	if err != nil {
		t.Fatalf("GenerateRandomInvite returned error: %v", err)
	}

	user := newInviteTestUser()
	created, err := CreateUserWithInviteCode(user, code)
	if err != nil {
		t.Fatalf("CreateUserWithInviteCode returned error: %v", err)
	}
	if _, err := GetUserInformation(created.ID); err != nil {
		t.Fatalf("created user not found: %v", err)
	}

	// Once claimed, the code must no longer verify as unused, and must name
	// the new user as its recipient.
	if ok, err := VerifyUnusedUserInviteCode(code); err != nil || ok {
		t.Fatalf("expected used invite to be invalid (ok=%v err=%v)", ok, err)
	}
	var invite models.Invite
	if err := Instance.Where(&models.Invite{Code: code}).First(&invite).Error; err != nil {
		t.Fatalf("failed to reload invite: %v", err)
	}
	if invite.RecipientID == nil || *invite.RecipientID != created.ID {
		t.Errorf("recipient = %v, want %v", invite.RecipientID, created.ID)
	}
}

// A code that's already claimed (e.g. by a registration that won a race) must
// fail with ErrInviteCodeUnavailable and leave no user behind.
func TestCreateUserWithInviteCodeAlreadyUsed(t *testing.T) {
	setupTestDB(t)
	code, err := GenerateRandomInvite()
	if err != nil {
		t.Fatalf("GenerateRandomInvite returned error: %v", err)
	}
	if _, err := CreateUserWithInviteCode(newInviteTestUser(), code); err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	second := newInviteTestUser()
	if _, err := CreateUserWithInviteCode(second, code); !errors.Is(err, ErrInviteCodeUnavailable) {
		t.Fatalf("err = %v, want ErrInviteCodeUnavailable", err)
	}
	if _, err := GetUserInformation(second.ID); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("second user lookup err = %v, want ErrUserNotFound (creation rolled back)", err)
	}
}

// If claiming the invite fails, the user insert must be rolled back too.
func TestCreateUserWithInviteCodeClaimFailureRollsBack(t *testing.T) {
	setupTestDB(t)
	code, err := GenerateRandomInvite()
	if err != nil {
		t.Fatalf("GenerateRandomInvite returned error: %v", err)
	}
	injectFault(t, "update", "invites", 0)

	user := newInviteTestUser()
	if _, err := CreateUserWithInviteCode(user, code); !errors.Is(err, errInjected) {
		t.Fatalf("err = %v, want the injected fault", err)
	}
	if _, err := GetUserInformation(user.ID); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("user lookup err = %v, want ErrUserNotFound (creation rolled back)", err)
	}
	if ok, err := VerifyUnusedUserInviteCode(code); err != nil || !ok {
		t.Errorf("invite should still be unused (ok=%v err=%v)", ok, err)
	}
}

func TestCreateUserWithInviteCodeInsertFailure(t *testing.T) {
	setupTestDB(t)
	code, err := GenerateRandomInvite()
	if err != nil {
		t.Fatalf("GenerateRandomInvite returned error: %v", err)
	}
	injectFault(t, "create", "users", 0)

	if _, err := CreateUserWithInviteCode(newInviteTestUser(), code); !errors.Is(err, errInjected) {
		t.Fatalf("err = %v, want the injected fault", err)
	}
	if ok, err := VerifyUnusedUserInviteCode(code); err != nil || !ok {
		t.Errorf("invite should still be unused (ok=%v err=%v)", ok, err)
	}
}

// newInviteTestUser builds (without inserting) an enabled user with a unique
// e-mail, for passing to CreateUserWithInviteCode.
func newInviteTestUser() models.User {
	email := uuid.NewString() + "@example.com"
	user := models.User{FirstName: "Invited", LastName: "User", Email: &email, Enabled: &utilities.DBTrue}
	user.ID = uuid.New()
	return user
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
	if _, err := GetInviteByID(invite.ID); !errors.Is(err, ErrInviteNotFound) {
		t.Fatalf("err = %v, want ErrInviteNotFound looking up disabled invite", err)
	}

	// Deleting an unknown invite fails (RowsAffected != 1).
	if err := DeleteInviteByID(uuid.New()); err == nil {
		t.Fatalf("expected error deleting unknown invite, got nil")
	}
}

func TestInviteQueriesFailOnClosedDB(t *testing.T) {
	runClosedDBCases(t, map[string]func() error{
		"GetAllEnabledInvites": func() error {
			_, err := GetAllEnabledInvites()
			return err
		},
		"GetInviteByID": func() error {
			_, err := GetInviteByID(uuid.New())
			return err
		},
		"DeleteInviteByID": func() error {
			return DeleteInviteByID(uuid.New())
		},
	})
}
