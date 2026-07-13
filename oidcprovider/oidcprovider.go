// Package oidcprovider builds and caches the OpenID Connect provider, ID-token
// verifier, and OAuth2 config from the application configuration. The client is
// created lazily on first use (so the app still starts if the IdP is briefly
// unreachable) and rebuilt if the relevant configuration changes.
package oidcprovider

import (
	"aunefyren/poenskelisten/config"
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Client bundles everything a handler needs to run the OIDC flow.
type Client struct {
	Provider     *oidc.Provider
	Verifier     *oidc.IDTokenVerifier
	OAuth2Config oauth2.Config
}

var (
	mutex  sync.Mutex
	cached *Client
	// cacheKey captures the config the cached client was built from, so we rebuild
	// when any of it changes.
	cacheKey string
)

// ErrNotConfigured is returned when OIDC is disabled or missing required settings.
var ErrNotConfigured = errors.New("oidc is not configured")

func currentKey() string {
	c := config.ConfigFile
	return strings.Join([]string{c.OIDCIssuerURL, c.OIDCClientID, c.OIDCClientSecret, c.OIDCRedirectURL}, "|")
}

// Get returns a ready OIDC client, building (and caching) it on first use.
func Get() (*Client, error) {
	c := config.ConfigFile
	if !c.OIDCEnabled {
		return nil, ErrNotConfigured
	}
	if strings.TrimSpace(c.OIDCIssuerURL) == "" || strings.TrimSpace(c.OIDCClientID) == "" || strings.TrimSpace(c.OIDCRedirectURL) == "" {
		return nil, ErrNotConfigured
	}

	mutex.Lock()
	defer mutex.Unlock()

	key := currentKey()
	if cached != nil && cacheKey == key {
		return cached, nil
	}

	provider, err := oidc.NewProvider(context.Background(), c.OIDCIssuerURL)
	if err != nil {
		return nil, err
	}

	client := &Client{
		Provider: provider,
		Verifier: provider.Verifier(&oidc.Config{ClientID: c.OIDCClientID}),
		OAuth2Config: oauth2.Config{
			ClientID:     c.OIDCClientID,
			ClientSecret: c.OIDCClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  c.OIDCRedirectURL,
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
	}

	cached = client
	cacheKey = key
	return client, nil
}
