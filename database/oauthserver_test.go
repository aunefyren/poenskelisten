package database

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/models"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSeedFirstPartyClientIdempotent(t *testing.T) {
	setupTestDB(t)
	config.ConfigFile.PoenskelistenExternalURL = "https://wishlist.example.com"
	config.ConfigFile.PoenskelistenName = "Test"

	if err := SeedFirstPartyClient(); err != nil {
		t.Fatalf("SeedFirstPartyClient error: %v", err)
	}
	if err := SeedFirstPartyClient(); err != nil {
		t.Fatalf("second SeedFirstPartyClient error: %v", err)
	}

	client, found, err := GetOAuthClient(models.FirstPartyClientID)
	if err != nil || !found {
		t.Fatalf("first-party client not found: found=%v err=%v", found, err)
	}
	if !client.IsFirstParty || !client.IsPublic || !client.IsEnabled() {
		t.Error("first-party client should be public, first-party, enabled")
	}
	if !client.HasRedirectURI("https://wishlist.example.com/oauth/callback") {
		t.Errorf("redirect URI not set correctly: %v", client.RedirectURIs)
	}
	if !client.AllowsScopes([]string{"openid", "email"}) {
		t.Errorf("expected identity scopes, got %v", client.Scopes)
	}

	// Exactly one row.
	var count int64
	Instance.Model(&models.OAuthClient{}).Where(&models.OAuthClient{ClientID: models.FirstPartyClientID}).Count(&count)
	if count != 1 {
		t.Errorf("got %d first-party clients, want 1", count)
	}
}

func TestCreateListDisableOAuthClient(t *testing.T) {
	setupTestDB(t)

	secret := "should-not-leak"
	enabled := true
	created, err := CreateOAuthClient(models.OAuthClient{
		ClientID:         "dyn-client-1",
		ClientName:       "Dynamic",
		ClientSecretHash: &secret,
		RedirectURIs:     []string{"https://client.example/callback"},
		Scopes:           []string{"openid", "mcp:wishlists.read"},
		IsPublic:         true,
		Registered:       true,
		Enabled:          &enabled,
	})
	if err != nil {
		t.Fatalf("CreateOAuthClient error: %v", err)
	}
	if created.ClientID != "dyn-client-1" {
		t.Errorf("client id = %q", created.ClientID)
	}

	clients, err := GetAllOAuthClients()
	if err != nil {
		t.Fatalf("GetAllOAuthClients error: %v", err)
	}
	found := false
	for _, c := range clients {
		if c.ClientID == "dyn-client-1" {
			found = true
			if c.ClientSecretHash != nil {
				t.Error("listing leaked the client secret hash")
			}
		}
	}
	if !found {
		t.Error("created client not returned by GetAllOAuthClients")
	}

	if err := DisableOAuthClient("dyn-client-1"); err != nil {
		t.Fatalf("DisableOAuthClient error: %v", err)
	}
	client, _, _ := GetOAuthClient("dyn-client-1")
	if client.IsEnabled() {
		t.Error("client should be disabled after revoke")
	}
}

func TestDisableFirstPartyClientRefused(t *testing.T) {
	setupTestDB(t)
	config.ConfigFile.PoenskelistenExternalURL = "https://wishlist.example.com"
	config.ConfigFile.PoenskelistenName = "Test"
	if err := SeedFirstPartyClient(); err != nil {
		t.Fatalf("SeedFirstPartyClient error: %v", err)
	}

	if err := DisableOAuthClient(models.FirstPartyClientID); err == nil {
		t.Error("disabling the first-party client should be refused")
	}
}

func TestGetUserConsents(t *testing.T) {
	setupTestDB(t)
	userID := uuid.New()

	if err := UpsertConsent(userID, "client-x", []string{"openid"}); err != nil {
		t.Fatalf("UpsertConsent error: %v", err)
	}
	if err := UpsertConsent(userID, "client-y", []string{"email"}); err != nil {
		t.Fatalf("UpsertConsent error: %v", err)
	}
	// Another user's consent must not leak into this user's list.
	if err := UpsertConsent(uuid.New(), "client-z", []string{"openid"}); err != nil {
		t.Fatalf("UpsertConsent error: %v", err)
	}

	consents, err := GetUserConsents(userID)
	if err != nil {
		t.Fatalf("GetUserConsents error: %v", err)
	}
	if len(consents) != 2 {
		t.Errorf("got %d consents, want 2", len(consents))
	}
}

func TestAuthorizationCodeConsumeOnce(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)

	code := models.AuthorizationCode{
		CodeHash:            "hash-1",
		ClientID:            models.FirstPartyClientID,
		UserID:              user.ID,
		RedirectURI:         "https://x/callback",
		Scope:               "openid",
		Resource:            "https://x/api",
		CodeChallenge:       "chal",
		CodeChallengeMethod: "S256",
		ExpiresAt:           time.Now().Add(60 * time.Second),
	}
	if _, err := CreateAuthorizationCode(code); err != nil {
		t.Fatalf("CreateAuthorizationCode error: %v", err)
	}

	got, err := ConsumeAuthorizationCode("hash-1")
	if err != nil {
		t.Fatalf("ConsumeAuthorizationCode error: %v", err)
	}
	if got.UserID != user.ID || got.Scope != "openid" {
		t.Errorf("consumed code mismatch: %+v", got)
	}

	// A second consume must fail (replay protection).
	if _, err := ConsumeAuthorizationCode("hash-1"); err == nil {
		t.Error("second ConsumeAuthorizationCode succeeded, want error")
	}
}

func TestAuthorizationCodeExpired(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)

	code := models.AuthorizationCode{
		CodeHash:  "hash-expired",
		ClientID:  models.FirstPartyClientID,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(-1 * time.Second),
	}
	if _, err := CreateAuthorizationCode(code); err != nil {
		t.Fatalf("CreateAuthorizationCode error: %v", err)
	}

	if _, err := ConsumeAuthorizationCode("hash-expired"); err == nil {
		t.Error("consuming an expired code succeeded, want error")
	}
}

func TestConsentUpsertAndCover(t *testing.T) {
	setupTestDB(t)
	userID := uuid.New()

	if _, found, _ := GetConsent(userID, "client-a"); found {
		t.Error("unexpected consent before any grant")
	}

	if err := UpsertConsent(userID, "client-a", []string{"openid", "email"}); err != nil {
		t.Fatalf("UpsertConsent error: %v", err)
	}
	consent, found, err := GetConsent(userID, "client-a")
	if err != nil || !found {
		t.Fatalf("consent not found after upsert: found=%v err=%v", found, err)
	}
	if !consent.Covers([]string{"openid"}) {
		t.Error("consent should cover a subset of granted scopes")
	}
	if consent.Covers([]string{"profile"}) {
		t.Error("consent should not cover an ungranted scope")
	}

	// Upsert replaces the scope set.
	if err := UpsertConsent(userID, "client-a", []string{"openid", "profile"}); err != nil {
		t.Fatalf("second UpsertConsent error: %v", err)
	}
	consent, _, _ = GetConsent(userID, "client-a")
	if !consent.Covers([]string{"profile"}) || consent.Covers([]string{"email"}) {
		t.Errorf("consent scopes not replaced: %v", consent.Scopes)
	}

	if err := RevokeConsent(userID, "client-a"); err != nil {
		t.Fatalf("RevokeConsent error: %v", err)
	}
	if _, found, _ := GetConsent(userID, "client-a"); found {
		t.Error("consent still present after revoke")
	}
}
