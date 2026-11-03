package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseTrustedProxies(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    []string
		wantErr string
	}{
		{"empty", "", []string{}, ""},
		{"blank entries ignored", " , ,", []string{}, ""},
		{"single IPv4", "172.18.0.2", []string{"172.18.0.2"}, ""},
		{"IPv6 normalized", "2001:DB8::0001", []string{"2001:db8::1"}, ""},
		{"CIDR normalized to network", " 172.18.0.5/16 , 10.0.0.1", []string{"172.18.0.0/16", "10.0.0.1"}, ""},
		{"IPv6 CIDR", "fd00::/8", []string{"fd00::/8"}, ""},
		{"hostname refused", "proxy.lan", nil, "not an IP address"},
		{"bad mask refused", "10.0.0.0/33", nil, "not an IP address"},
		{"trust-all IPv4 refused", "0.0.0.0/0", nil, "every address"},
		{"trust-all IPv6 refused", "10.0.0.1,::/0", nil, "every address"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseTrustedProxies(c.raw)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("error = %v, want one containing %q", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestValidTrustedProxiesSkipsInvalidEntries(t *testing.T) {
	got := ValidTrustedProxies("10.0.0.1, proxy.lan, 0.0.0.0/0, 172.16.0.0/12")
	want := []string{"10.0.0.1", "172.16.0.0/12"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
