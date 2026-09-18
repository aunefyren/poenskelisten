package models

import (
	"os"
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
