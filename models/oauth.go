package models

import (
	"time"

	"github.com/google/uuid"
)

// FirstPartyClientID is the fixed client_id of the built-in web app client seeded
// on migration. It is a public PKCE client with auto-consent.
const FirstPartyClientID = "poenskelisten-web"

// OAuth token endpoint auth methods.
const (
	TokenEndpointAuthNone         = "none"
	TokenEndpointAuthClientSecret = "client_secret_post"
)

// OAuthClient is a registered OAuth 2.1 client. Public clients (ClientSecretHash
// nil) authenticate with PKCE only; confidential clients present a secret.
type OAuthClient struct {
	GormModel
	ClientID                string   `json:"client_id" gorm:"unique; not null; index"`
	ClientSecretHash        *string  `json:"-"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris" gorm:"serializer:json"`
	Scopes                  []string `json:"scopes" gorm:"serializer:json"`
	GrantTypes              []string `json:"grant_types" gorm:"serializer:json"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	IsPublic                bool     `json:"is_public"`
	IsFirstParty            bool     `json:"is_first_party"`
	Registered              bool     `json:"registered"`
	Enabled                 *bool    `json:"enabled"`
}

// IsEnabled tolerates a nil Enabled pointer (treated as disabled).
func (c *OAuthClient) IsEnabled() bool {
	return c.Enabled != nil && *c.Enabled
}

// HasRedirectURI reports whether uri exactly matches one of the client's
// registered redirect URIs (exact match, per OAuth 2.1).
func (c *OAuthClient) HasRedirectURI(uri string) bool {
	for _, registered := range c.RedirectURIs {
		if registered == uri {
			return true
		}
	}
	return false
}

// AllowsScopes reports whether every requested scope is permitted for the client.
func (c *OAuthClient) AllowsScopes(requested []string) bool {
	allowed := make(map[string]bool, len(c.Scopes))
	for _, s := range c.Scopes {
		allowed[s] = true
	}
	for _, s := range requested {
		if !allowed[s] {
			return false
		}
	}
	return true
}

// OAuthRegisterRequest is the RFC 7591 dynamic client registration payload.
type OAuthRegisterRequest struct {
	RedirectURIs            []string `json:"redirect_uris"`
	ClientName              string   `json:"client_name"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	Scope                   string   `json:"scope"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// AuthorizationCode is a single-use, short-lived code exchanged at the token
// endpoint. Only the SHA-256 hash of the code is stored.
type AuthorizationCode struct {
	GormModel
	CodeHash            string     `json:"-" gorm:"not null; index"`
	ClientID            string     `json:"client_id" gorm:"not null"`
	UserID              uuid.UUID  `json:"user_id" gorm:"not null"`
	RedirectURI         string     `json:"redirect_uri"`
	Scope               string     `json:"scope"`
	Resource            string     `json:"resource"`
	CodeChallenge       string     `json:"code_challenge"`
	CodeChallengeMethod string     `json:"code_challenge_method"`
	ExpiresAt           time.Time  `json:"expires_at"`
	UsedAt              *time.Time `json:"used_at"`
}

// OAuthConsent records the scopes a user has granted to a client, so a returning
// user skips the consent screen when the scope set is unchanged.
type OAuthConsent struct {
	GormModel
	UserID   uuid.UUID `json:"user_id" gorm:"not null; index"`
	ClientID string    `json:"client_id" gorm:"not null; index"`
	Scopes   []string  `json:"scopes" gorm:"serializer:json"`
}

// Covers reports whether this consent already grants every requested scope.
func (c *OAuthConsent) Covers(requested []string) bool {
	granted := make(map[string]bool, len(c.Scopes))
	for _, s := range c.Scopes {
		granted[s] = true
	}
	for _, s := range requested {
		if !granted[s] {
			return false
		}
	}
	return true
}
