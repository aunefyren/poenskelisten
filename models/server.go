package models

// ServerInfoReply is the admin-only overview of the running configuration. It
// deliberately excludes every secret (private key, DB/SMTP passwords, OIDC client
// secret); only non-sensitive settings are exposed.
type ServerInfoReply struct {
	// Application
	AppName                  string `json:"app_name"`
	PoenskelistenVersion     string `json:"poenskelisten_version"`
	PoenskelistenEnvironment string `json:"poenskelisten_environment"`
	PoenskelistenExternalURL string `json:"poenskelisten_external_url"`
	PoenskelistenPort        int    `json:"poenskelisten_port"`
	Timezone                 string `json:"timezone"`
	PoenskelistenLogLevel    string `json:"poenskelisten_log_level"`
	PoenskelistenTestEmail   string `json:"poenskelisten_test_email"`

	// Database
	DatabaseType     string `json:"database_type"`
	DatabaseName     string `json:"database_name"`
	DatabaseHost     string `json:"database_host"`
	DatabasePort     int    `json:"database_port"`
	DatabaseSSL      bool   `json:"database_ssl"`
	DatabaseLocation string `json:"database_location"`

	// Email (SMTP)
	SMTPEnabled bool   `json:"smtp_enabled"`
	SMTPHost    string `json:"smtp_host"`
	SMTPPort    int    `json:"smtp_port"`
	SMTPFrom    string `json:"smtp_from"`

	// Single sign-on (OIDC)
	OIDCEnabled         bool   `json:"oidc_enabled"`
	OIDCProviderName    string `json:"oidc_provider_name"`
	OIDCIssuerURL       string `json:"oidc_issuer_url"`
	OIDCClientID        string `json:"oidc_client_id"`
	OIDCRedirectURL     string `json:"oidc_redirect_url"`
	OIDCAutoCreateUsers bool   `json:"oidc_auto_create_users"`

	// Security (MFA)
	MFAEnforced             bool `json:"mfa_enforced"`
	MFARecoveryCodesEnabled bool `json:"mfa_recovery_codes_enabled"`
}
