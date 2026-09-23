package config

import (
	"errors"
	"net/url"
	"strings"
)

// NormalizeOrigin reduces a URL to the origin a browser reports for it
// (scheme://host[:port], lowercase, default port dropped), so it can be compared
// with window.location.origin. Anything beyond an origin is refused rather than
// silently dropped: a path would mean the URL doesn't point at the app's root.
func NormalizeOrigin(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", errors.New("'" + raw + "' is not a valid URL")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", errors.New("'" + raw + "' must start with http:// or https://")
	}
	if parsed.Hostname() == "" {
		return "", errors.New("'" + raw + "' has no host")
	}
	if parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("'" + raw + "' must be just scheme, host and port (e.g. http://192.168.1.10:8080)")
	}

	host := strings.ToLower(parsed.Host)
	port := parsed.Port()
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		host = strings.ToLower(parsed.Hostname())
		if strings.Contains(host, ":") {
			host = "[" + host + "]"
		}
	}
	return scheme + "://" + host, nil
}

// ParseAdditionalURLs splits the comma-separated additional URL list and
// normalizes each entry. Empty entries are ignored; the first invalid one is an
// error.
func ParseAdditionalURLs(raw string) ([]string, error) {
	origins := []string{}
	for _, entry := range strings.Split(raw, ",") {
		if strings.TrimSpace(entry) == "" {
			continue
		}
		origin, err := NormalizeOrigin(entry)
		if err != nil {
			return nil, err
		}
		origins = append(origins, origin)
	}
	return origins, nil
}

// AllowedOrigins returns every origin the web app may log in from: the OAuth
// issuer (the external URL) first, then each additional URL, deduplicated. Only
// login is multi-origin; e-mail links, OIDC callbacks and the issuer itself keep
// using the external URL. Invalid additional entries can only come from a
// hand-edited config.json (flags are validated on parse) and are skipped here;
// startup logs a warning about them.
func AllowedOrigins() []string {
	issuer := OAuthIssuer()
	origins := []string{issuer}
	seen := map[string]bool{strings.ToLower(issuer): true}
	for _, entry := range strings.Split(ConfigFile.PoenskelistenAdditionalURLs, ",") {
		origin, err := NormalizeOrigin(entry)
		if err != nil || seen[origin] {
			continue
		}
		seen[origin] = true
		origins = append(origins, origin)
	}
	return origins
}
