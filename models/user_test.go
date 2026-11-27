package models

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// TestMain drops BcryptCost to bcrypt's minimum for the whole package's test
// run. At the production cost, hashing is deliberately slow, and under
// `go test -race` that cost is amplified roughly another order of magnitude
// (measured: ~1s per hash at the production cost without -race, ~12s with
// it) - real tests only need the round-trip to work, not production-strength
// hardness.
func TestMain(m *testing.M) {
	BcryptCost = bcrypt.MinCost
	os.Exit(m.Run())
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func TestHashAndCheckPasswordRoundTrip(t *testing.T) {
	placeholder := ""
	user := User{Password: &placeholder}

	if err := user.HashPassword("correct-horse-battery-staple"); err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if *user.Password == "correct-horse-battery-staple" {
		t.Error("HashPassword left the password in plaintext")
	}

	if err := user.CheckPassword("correct-horse-battery-staple"); err != nil {
		t.Errorf("CheckPassword rejected the correct password: %v", err)
	}
	if err := user.CheckPassword("wrong-password"); err == nil {
		t.Error("CheckPassword accepted the wrong password")
	}
}

func TestCheckPasswordNoPasswordSet(t *testing.T) {
	user := User{}
	if err := user.CheckPassword("anything"); err == nil {
		t.Error("expected an error when the user has no password set")
	}
}

func TestHasPassword(t *testing.T) {
	cases := []struct {
		name string
		user User
		want bool
	}{
		{"nil pointer", User{}, false},
		{"empty string", User{Password: strPtr("")}, false},
		{"set", User{Password: strPtr("hash")}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.user.HasPassword(); got != c.want {
				t.Errorf("HasPassword() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestIsMFAEnabled(t *testing.T) {
	cases := []struct {
		name string
		user User
		want bool
	}{
		{"nil pointer", User{}, false},
		{"false", User{MFAEnabled: boolPtr(false)}, false},
		{"true", User{MFAEnabled: boolPtr(true)}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.user.IsMFAEnabled(); got != c.want {
				t.Errorf("IsMFAEnabled() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestIsLocalAuth(t *testing.T) {
	cases := []struct {
		name string
		user User
		want bool
	}{
		{"nil AuthSource treated as local", User{}, true},
		{"empty AuthSource treated as local", User{AuthSource: strPtr("")}, true},
		{"explicit local", User{AuthSource: strPtr(AuthSourceLocal)}, true},
		{"oidc is not local", User{AuthSource: strPtr(AuthSourceOIDC)}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.user.IsLocalAuth(); got != c.want {
				t.Errorf("IsLocalAuth() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestHasOIDCIdentity(t *testing.T) {
	cases := []struct {
		name string
		user User
		want bool
	}{
		{"no subject", User{}, false},
		{"empty subject", User{OIDCSubject: strPtr("")}, false},
		{"subject set", User{OIDCSubject: strPtr("sub-1")}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.user.HasOIDCIdentity(); got != c.want {
				t.Errorf("HasOIDCIdentity() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestNewAdminUserSignInMethod(t *testing.T) {
	cases := []struct {
		name           string
		user           User
		wantAuthSource string
		wantLinked     bool
	}{
		{"legacy account without auth source", User{Password: strPtr("hash")}, AuthSourceLocal, false},
		{"local account", User{Password: strPtr("hash"), AuthSource: strPtr(AuthSourceLocal)}, AuthSourceLocal, false},
		{"local account linked to an identity provider", User{Password: strPtr("hash"), OIDCSubject: strPtr("sub-1"), OIDCIssuer: strPtr("https://idp.example.com")}, AuthSourceLocal, true},
		{"account provisioned through single sign-on", User{AuthSource: strPtr(AuthSourceOIDC), OIDCSubject: strPtr("sub-2"), OIDCIssuer: strPtr("https://idp.example.com")}, AuthSourceOIDC, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := NewAdminUser(c.user)
			if got.AuthSource != c.wantAuthSource || got.OIDCLinked != c.wantLinked {
				t.Errorf("auth_source = %q, oidc_linked = %v, want %q, %v", got.AuthSource, got.OIDCLinked, c.wantAuthSource, c.wantLinked)
			}
		})
	}
}

// The admin view is an allowlist: whatever else a User carries, only these
// keys may reach the response.
func TestNewAdminUserCarriesOnlyItsOwnFields(t *testing.T) {
	email := "ada@example.com"
	user := User{
		FirstName:        "Ada",
		LastName:         "Lovelace",
		Email:            &email,
		Password:         strPtr("hash"),
		Admin:            true,
		Enabled:          boolPtr(true),
		Verified:         boolPtr(true),
		VerificationCode: strPtr("VERIFYCODE"),
		ResetCode:        strPtr("RESETCODE"),
		MFAEnabled:       boolPtr(true),
		MFASecret:        strPtr("MFASECRET"),
		OIDCSubject:      strPtr("SUBJECT"),
		OIDCIssuer:       strPtr("https://idp.example.com"),
	}

	adminUser := NewAdminUser(user)
	if adminUser.FirstName != "Ada" || adminUser.LastName != "Lovelace" || adminUser.Email != email ||
		!adminUser.Admin || !adminUser.Enabled || !adminUser.MFAEnabled {
		t.Errorf("NewAdminUser lost account details: %+v", adminUser)
	}

	encoded, err := json.Marshal(adminUser)
	if err != nil {
		t.Fatal(err)
	}
	var keys map[string]interface{}
	if err := json.Unmarshal(encoded, &keys); err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{"id": true, "created_at": true, "first_name": true, "last_name": true, "email": true,
		"admin": true, "enabled": true, "mfa_enabled": true, "auth_source": true, "oidc_linked": true}
	for key := range keys {
		if !allowed[key] {
			t.Errorf("admin view carries %q", key)
		}
	}
	for _, secret := range []string{"hash", "VERIFYCODE", "RESETCODE", "MFASECRET", "SUBJECT", "idp.example.com"} {
		if strings.Contains(string(encoded), secret) {
			t.Errorf("admin view leaks %q: %s", secret, encoded)
		}
	}
}

func TestNewAdminUserDisabledAccount(t *testing.T) {
	if NewAdminUser(User{Enabled: boolPtr(false)}).Enabled {
		t.Error("a disabled account is shown as enabled")
	}
	if NewAdminUser(User{}).Enabled {
		t.Error("an account without an enabled flag is shown as enabled")
	}
}

// TestHashPasswordTooLongKeepsExistingHash: bcrypt refuses inputs over 72
// bytes, and the stored password must be left untouched when it does.
func TestHashPasswordTooLongKeepsExistingHash(t *testing.T) {
	existing := "existing-hash"
	user := User{Password: &existing}

	err := user.HashPassword(strings.Repeat("a", 73))
	if !errors.Is(err, bcrypt.ErrPasswordTooLong) {
		t.Errorf("HashPassword error = %v, want bcrypt.ErrPasswordTooLong", err)
	}
	if *user.Password != "existing-hash" {
		t.Errorf("Password = %q, want the existing hash left in place", *user.Password)
	}
}
