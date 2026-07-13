// Package oauth holds shared building blocks for the OAuth 2.1 authorization
// server and the MCP resource server. This file defines the scope registry: the
// scopes the server can grant, their human descriptions (shown on the consent
// screen), and whether they are MCP resource scopes.
package oauth

import "strings"

// Scope is a single grantable permission.
type Scope struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// MCP marks scopes that gate access to the MCP resource server (as opposed to
	// the OIDC identity scopes).
	MCP bool `json:"mcp"`
}

// Registry is the full set of scopes this server understands. Consent, metadata
// (`scopes_supported`), and resource-server enforcement all read from here.
var Registry = []Scope{
	{Name: "openid", Description: "Verify your identity"},
	{Name: "profile", Description: "Read your basic profile (name)"},
	{Name: "email", Description: "Read your email address"},
	{Name: "mcp:wishlists.read", Description: "Read your wishlists and wishes", MCP: true},
	{Name: "mcp:wishlists.write", Description: "Create and edit your wishlists and wishes", MCP: true},
	{Name: "mcp:groups.read", Description: "Read the groups you belong to", MCP: true},
	{Name: "mcp:wishes.claim", Description: "Claim wishes on your behalf", MCP: true},
}

// byName indexes the registry for quick lookups.
var byName = func() map[string]Scope {
	m := make(map[string]Scope, len(Registry))
	for _, s := range Registry {
		m[s.Name] = s
	}
	return m
}()

// AllNames returns every supported scope name (for `scopes_supported` metadata).
func AllNames() []string {
	names := make([]string, 0, len(Registry))
	for _, s := range Registry {
		names = append(names, s.Name)
	}
	return names
}

// MCPNames returns the scope names that gate the MCP resource server.
func MCPNames() []string {
	names := make([]string, 0)
	for _, s := range Registry {
		if s.MCP {
			names = append(names, s.Name)
		}
	}
	return names
}

// IsValid reports whether a scope name is known.
func IsValid(name string) bool {
	_, ok := byName[name]
	return ok
}

// Lookup returns the scope for a name, and whether it exists.
func Lookup(name string) (Scope, bool) {
	s, ok := byName[name]
	return s, ok
}

// Parse splits a space-delimited scope string (OAuth's format) into names,
// dropping empties. It does not validate; use FilterValid for that.
func Parse(scope string) []string {
	return strings.Fields(scope)
}

// FilterValid returns the subset of the requested scopes that are known,
// preserving order and dropping duplicates.
func FilterValid(requested []string) []string {
	seen := make(map[string]bool)
	valid := make([]string, 0, len(requested))
	for _, name := range requested {
		if !seen[name] && IsValid(name) {
			seen[name] = true
			valid = append(valid, name)
		}
	}
	return valid
}
