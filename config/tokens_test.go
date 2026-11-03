package config

import (
	"testing"
	"time"
)

func TestAccessTokenLifetime(t *testing.T) {
	original := ConfigFile.AccessTokenMinutes
	t.Cleanup(func() { ConfigFile.AccessTokenMinutes = original })

	cases := []struct {
		minutes int
		want    time.Duration
		valid   bool
	}{
		{0, 15 * time.Minute, true},
		{1, time.Minute, true},
		{30, 30 * time.Minute, true},
		{60, time.Hour, true},
		{61, 15 * time.Minute, false},
		{-5, 15 * time.Minute, false},
	}
	for _, c := range cases {
		ConfigFile.AccessTokenMinutes = c.minutes
		if got := AccessTokenLifetime(); got != c.want {
			t.Errorf("AccessTokenLifetime() with %d = %v, want %v", c.minutes, got, c.want)
		}
		if err := ValidateAccessTokenMinutes(c.minutes); (err == nil) != c.valid {
			t.Errorf("ValidateAccessTokenMinutes(%d) error = %v, want valid %v", c.minutes, err, c.valid)
		}
	}
}

func TestRefreshTokenLifetime(t *testing.T) {
	original := ConfigFile.RefreshTokenDays
	t.Cleanup(func() { ConfigFile.RefreshTokenDays = original })

	day := 24 * time.Hour
	cases := []struct {
		days  int
		want  time.Duration
		valid bool
	}{
		{0, 7 * day, true},
		{1, day, true},
		{30, 30 * day, true},
		{365, 365 * day, true},
		{366, 7 * day, false},
		{-1, 7 * day, false},
	}
	for _, c := range cases {
		ConfigFile.RefreshTokenDays = c.days
		if got := RefreshTokenLifetime(); got != c.want {
			t.Errorf("RefreshTokenLifetime() with %d = %v, want %v", c.days, got, c.want)
		}
		if err := ValidateRefreshTokenDays(c.days); (err == nil) != c.valid {
			t.Errorf("ValidateRefreshTokenDays(%d) error = %v, want valid %v", c.days, err, c.valid)
		}
	}
}
