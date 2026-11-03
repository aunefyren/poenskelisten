package config

import (
	"fmt"
	"time"
)

// Token lifetime bounds. Access tokens are validated statelessly, so a revoked
// session's access token keeps working until it expires: the upper bound caps
// that window. A zero value in config.json means "use the default", which keeps
// following the default if it ever changes rather than freezing today's value.
const (
	DefaultAccessTokenMinutes = 15
	MinAccessTokenMinutes     = 1
	MaxAccessTokenMinutes     = 60

	DefaultRefreshTokenDays = 7
	MinRefreshTokenDays     = 1
	MaxRefreshTokenDays     = 365
)

// ValidateAccessTokenMinutes accepts 0 (the default) or a value within bounds.
func ValidateAccessTokenMinutes(minutes int) error {
	if minutes != 0 && (minutes < MinAccessTokenMinutes || minutes > MaxAccessTokenMinutes) {
		return fmt.Errorf("access token lifetime must be between %d and %d minutes, got %d", MinAccessTokenMinutes, MaxAccessTokenMinutes, minutes)
	}
	return nil
}

// ValidateRefreshTokenDays accepts 0 (the default) or a value within bounds.
func ValidateRefreshTokenDays(days int) error {
	if days != 0 && (days < MinRefreshTokenDays || days > MaxRefreshTokenDays) {
		return fmt.Errorf("refresh token lifetime must be between %d and %d days, got %d", MinRefreshTokenDays, MaxRefreshTokenDays, days)
	}
	return nil
}

// AccessTokenLifetime is how long an OAuth access token is valid. An unset or
// out-of-range value (a hand-edited config.json; flags are rejected at startup)
// falls back to the default instead of issuing an unbounded token.
func AccessTokenLifetime() time.Duration {
	minutes := ConfigFile.AccessTokenMinutes
	if minutes == 0 || ValidateAccessTokenMinutes(minutes) != nil {
		minutes = DefaultAccessTokenMinutes
	}
	return time.Duration(minutes) * time.Minute
}

// RefreshTokenLifetime is how long a refresh token (and thus a session) is valid
// before the user must log in again. Falls back to the default like
// AccessTokenLifetime.
func RefreshTokenLifetime() time.Duration {
	days := ConfigFile.RefreshTokenDays
	if days == 0 || ValidateRefreshTokenDays(days) != nil {
		days = DefaultRefreshTokenDays
	}
	return time.Duration(days) * 24 * time.Hour
}
