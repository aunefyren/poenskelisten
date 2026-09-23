package config

import (
	"reflect"
	"testing"
)

func TestNormalizeOrigin(t *testing.T) {
	cases := []struct {
		raw, want string
		wantErr   bool
	}{
		{raw: "http://192.168.1.10:8080", want: "http://192.168.1.10:8080"},
		{raw: "  HTTP://Wish.LAN/  ", want: "http://wish.lan"},
		{raw: "http://wish.lan:80", want: "http://wish.lan"},
		{raw: "https://wish.example.com:443", want: "https://wish.example.com"},
		{raw: "https://wish.example.com:8443", want: "https://wish.example.com:8443"},
		{raw: "http://[::1]:80", want: "http://[::1]"},
		{raw: "http://[::1]:8080", want: "http://[::1]:8080"},
		{raw: "192.168.1.10:8080", wantErr: true},
		{raw: "ftp://wish.lan", wantErr: true},
		{raw: "http://", wantErr: true},
		{raw: "http://wish.lan/app", wantErr: true},
		{raw: "http://wish.lan/?a=b", wantErr: true},
		{raw: "http://wish.lan/#top", wantErr: true},
		{raw: "http://user:pass@wish.lan", wantErr: true},
		{raw: "http://wish lan", wantErr: true},
	}
	for _, c := range cases {
		got, err := NormalizeOrigin(c.raw)
		if c.wantErr {
			if err == nil {
				t.Errorf("NormalizeOrigin(%q) = %q, want an error", c.raw, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("NormalizeOrigin(%q) = %q, %v; want %q", c.raw, got, err, c.want)
		}
	}
}

func TestParseAdditionalURLs(t *testing.T) {
	got, err := ParseAdditionalURLs(" http://192.168.1.10:8080/ ,, HTTP://wish.lan ,")
	if err != nil {
		t.Fatalf("ParseAdditionalURLs returned error: %v", err)
	}
	if want := []string{"http://192.168.1.10:8080", "http://wish.lan"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ParseAdditionalURLs = %v, want %v", got, want)
	}

	if got, err := ParseAdditionalURLs(""); err != nil || len(got) != 0 {
		t.Errorf("ParseAdditionalURLs(\"\") = %v, %v; want empty and no error", got, err)
	}

	if _, err := ParseAdditionalURLs("http://wish.lan,192.168.1.10"); err == nil {
		t.Error("ParseAdditionalURLs accepted an entry without a scheme")
	}
}

func TestAllowedOrigins(t *testing.T) {
	original := ConfigFile
	t.Cleanup(func() { ConfigFile = original })

	t.Run("issuer first, then valid deduplicated additional URLs", func(t *testing.T) {
		ConfigFile.PoenskelistenExternalURL = "https://wish.example.com/"
		ConfigFile.PoenskelistenAdditionalURLs = "http://192.168.1.10:8080,https://wish.example.com,not-a-url,http://192.168.1.10:8080/,http://wish.lan"
		want := []string{"https://wish.example.com", "http://192.168.1.10:8080", "http://wish.lan"}
		if got := AllowedOrigins(); !reflect.DeepEqual(got, want) {
			t.Errorf("AllowedOrigins() = %v, want %v", got, want)
		}
	})

	t.Run("only the issuer when no additional URLs are set", func(t *testing.T) {
		ConfigFile.PoenskelistenExternalURL = ""
		ConfigFile.PoenskelistenPort = 9090
		ConfigFile.PoenskelistenAdditionalURLs = ""
		if got := AllowedOrigins(); !reflect.DeepEqual(got, []string{"http://localhost:9090"}) {
			t.Errorf("AllowedOrigins() = %v, want only the localhost issuer", got)
		}
	})
}
