package database

import (
	"aunefyren/poenskelisten/models"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
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

func TestUserQueriesFailOnClosedDB(t *testing.T) {
	runClosedDBCases(t, map[string]func() error{
		"GetUserInformation": func() error {
			_, err := GetUserInformation(uuid.New())
			return err
		},
		"GetUserInformationAnyState": func() error {
			_, err := GetUserInformationAnyState(uuid.New())
			return err
		},
		"GetAllUserInformation": func() error {
			_, err := GetAllUserInformation(uuid.New())
			return err
		},
		"GetAllUserInformationAnyState": func() error {
			_, err := GetAllUserInformationAnyState(uuid.New())
			return err
		},
		"GetUserInformationByEmail": func() error {
			_, err := GetUserInformationByEmail("x")
			return err
		},
		"GetAllUserInformationByEmail": func() error {
			_, err := GetAllUserInformationByEmail("x")
			return err
		},
		"GenerateRandomResetCodeForUser": func() error {
			_, err := GenerateRandomResetCodeForUser(uuid.New(), true)
			return err
		},
		"GetAllUserInformationByResetCode": func() error {
			_, err := GetAllUserInformationByResetCode("x")
			return err
		},
		"GetAmountOfEnabledUsers": func() error {
			_, err := GetAmountOfEnabledUsers()
			return err
		},
		"GetEnabledUsers": func() error {
			_, err := GetEnabledUsers()
			return err
		},
		"GetAllUsers": func() error {
			_, err := GetAllUsers()
			return err
		},
		"UpdateUserInDB": func() error {
			_, err := UpdateUserInDB(models.User{})
			return err
		},
		"CreateUserInDB": func() error {
			_, err := CreateUserInDB(models.User{})
			return err
		},
	})
}

func TestGetAllUserInformationByEmailCaseInsensitive(t *testing.T) {
	setupTestDB(t)

	user := createTestUser(t)
	mixedCase := "Mixed.Case." + *user.Email
	user.Email = &mixedCase
	if _, err := UpdateUserInDB(user); err != nil {
		t.Fatalf("failed to update user: %v", err)
	}

	got, err := GetAllUserInformationByEmailCaseInsensitive(strings.ToUpper(mixedCase))
	if err != nil || got.ID != user.ID {
		t.Fatalf("got user %v err %v, want %v", got.ID, err, user.ID)
	}

	if _, err := GetAllUserInformationByEmailCaseInsensitive("nobody@example.com"); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("err = %v, want ErrUserNotFound", err)
	}

	// Two accounts differing only in case must not resolve to either one.
	other := createTestUser(t)
	lowerCase := strings.ToLower(mixedCase)
	other.Email = &lowerCase
	if _, err := UpdateUserInDB(other); err != nil {
		t.Fatalf("failed to update second user: %v", err)
	}
	if _, err := GetAllUserInformationByEmailCaseInsensitive(mixedCase); err == nil {
		t.Error("expected an error when two users match ignoring case")
	}
}

func TestUserLookupsForUnknownUserReturnErrUserNotFound(t *testing.T) {
	setupTestDB(t)
	missing := uuid.New()

	lookups := map[string]func() error{
		"GetUserInformationAnyState": func() error {
			_, err := GetUserInformationAnyState(missing)
			return err
		},
		"GetAllUserInformation": func() error {
			_, err := GetAllUserInformation(missing)
			return err
		},
		"GetAllUserInformationAnyState": func() error {
			_, err := GetAllUserInformationAnyState(missing)
			return err
		},
		"GetAllUserInformationByEmail": func() error {
			_, err := GetAllUserInformationByEmail("nobody@example.com")
			return err
		},
		"GetAllUserInformationByResetCode": func() error {
			_, err := GetAllUserInformationByResetCode("NOSUCHCODE")
			return err
		},
	}
	for name, lookup := range lookups {
		if err := lookup(); !errors.Is(err, ErrUserNotFound) {
			t.Errorf("%s error = %v, want ErrUserNotFound", name, err)
		}
	}
}

func TestGetAllUserInformationByEmailCaseInsensitiveQueryFails(t *testing.T) {
	setupTestDB(t)
	injectFault(t, "query", "users", 0)

	if _, err := GetAllUserInformationByEmailCaseInsensitive("a@example.com"); !errors.Is(err, errInjected) {
		t.Fatalf("error = %v, want the injected fault", err)
	}
}

// GenerateRandomResetCodeForUser writes the code and its expiry separately;
// a failure of the expiry write must not be reported as success.
func TestGenerateRandomResetCodeForUserExpiryWriteFailures(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		setupTestDB(t)
		user := createTestUser(t)
		injectFault(t, "update", "users", 1)

		if code, err := GenerateRandomResetCodeForUser(user.ID, true); !errors.Is(err, errInjected) || code != "" {
			t.Fatalf("got (%q, %v), want no code and the injected fault", code, err)
		}
	})

	t.Run("no rows", func(t *testing.T) {
		setupTestDB(t)
		user := createTestUser(t)
		if err := registerCallback("update", true, func(db *gorm.DB) {
			if dest, ok := db.Statement.Dest.(map[string]interface{}); ok {
				if _, isExpiry := dest["reset_expiration"]; isExpiry {
					db.RowsAffected = 0
				}
			}
		}); err != nil {
			t.Fatalf("failed to register callback: %v", err)
		}

		code, err := GenerateRandomResetCodeForUser(user.ID, true)
		if code != "" || err == nil || err.Error() != "Reset code expiration not changed in database." {
			t.Fatalf("got (%q, %v), want the expiration error", code, err)
		}
	})
}
