package controllers

import (
	"aunefyren/poenskelisten/database"
	"errors"
	"testing"
)

func TestDeriveNames(t *testing.T) {
	cases := []struct {
		given, family, name, email string
		wantFirst, wantLast        string
	}{
		{"Ada", "Lovelace", "ignored", "ada@example.com", "Ada", "Lovelace"},
		{"", "", "Grace Hopper", "grace@example.com", "Grace", "Hopper"},
		{"", "", "Cher", "cher@example.com", "Cher", ""},
		{"", "", "", "alan@example.com", "alan", ""},
		{"", "", "", "", "User", ""},
		{"  ", "  ", "  Ada  Lovelace ", "x@y.com", "Ada", "Lovelace"},
		{"", "", "Mary Jane Watson", "mj@example.com", "Mary", "Jane Watson"},
	}

	for _, c := range cases {
		gotFirst, gotLast := deriveNames(c.given, c.family, c.name, c.email)
		if gotFirst != c.wantFirst || gotLast != c.wantLast {
			t.Errorf("deriveNames(%q,%q,%q,%q) = (%q,%q), want (%q,%q)",
				c.given, c.family, c.name, c.email, gotFirst, gotLast, c.wantFirst, c.wantLast)
		}
	}
}

func TestOIDCResolveErrorMessage(t *testing.T) {
	// Sentinel errors map to specific messages; anything else is generic.
	if msg := oidcResolveErrorMessage(database.ErrOIDCEmailNotVerified); msg == "Single sign-on failed." {
		t.Error("expected a specific message for ErrOIDCEmailNotVerified")
	}
	if msg := oidcResolveErrorMessage(database.ErrOIDCUserNotFound); msg == "Single sign-on failed." {
		t.Error("expected a specific message for ErrOIDCUserNotFound")
	}
	if msg := oidcResolveErrorMessage(database.ErrOIDCNoEmail); msg == "Single sign-on failed." {
		t.Error("expected a specific message for ErrOIDCNoEmail")
	}
	if msg := oidcResolveErrorMessage(errors.New("boom")); msg != "Single sign-on failed." {
		t.Errorf("unknown error mapped to %q, want generic message", msg)
	}
}
