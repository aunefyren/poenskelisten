package config

import (
	"errors"
	"net"
	"strings"
)

// normalizeProxy accepts one IP address or CIDR range and returns it in
// canonical form. A range covering every address is refused: trusting all
// peers lets any client pick its own IP through X-Forwarded-For, which is
// exactly what the setting exists to prevent.
func normalizeProxy(raw string) (string, error) {
	entry := strings.TrimSpace(raw)
	if ip := net.ParseIP(entry); ip != nil {
		return ip.String(), nil
	}
	_, network, err := net.ParseCIDR(entry)
	if err != nil {
		return "", errors.New("'" + raw + "' is not an IP address or CIDR range (e.g. 172.18.0.2 or 172.16.0.0/12)")
	}
	if ones, _ := network.Mask.Size(); ones == 0 {
		return "", errors.New("'" + raw + "' would trust every address; list your reverse proxy's address instead")
	}
	return network.String(), nil
}

// ParseTrustedProxies splits the comma-separated trusted proxy list and
// normalizes each entry. Empty entries are ignored; the first invalid one is an
// error.
func ParseTrustedProxies(raw string) ([]string, error) {
	proxies := []string{}
	for _, entry := range strings.Split(raw, ",") {
		if strings.TrimSpace(entry) == "" {
			continue
		}
		proxy, err := normalizeProxy(entry)
		if err != nil {
			return nil, err
		}
		proxies = append(proxies, proxy)
	}
	return proxies, nil
}

// ValidTrustedProxies is ParseTrustedProxies minus the invalid entries, which
// can only come from a hand-edited config.json (flags are validated on parse);
// startup logs a warning about them. Skipping an entry only ever trusts fewer
// peers, so it fails safe.
func ValidTrustedProxies(raw string) []string {
	proxies := []string{}
	for _, entry := range strings.Split(raw, ",") {
		if proxy, err := normalizeProxy(entry); err == nil {
			proxies = append(proxies, proxy)
		}
	}
	return proxies
}
