package models

type ConfigStruct struct {
	Timezone                  string `json:"timezone"`
	PrivateKey                string `json:"private_key"`
	DBType                    string `json:"db_type"`
	DBUsername                string `json:"db_username"`
	DBPassword                string `json:"db_password"`
	DBName                    string `json:"db_name"`
	DBIP                      string `json:"db_ip"`
	DBPort                    int    `json:"db_port"`
	DBSSL                     bool   `json:"db_ssl"`
	DBLocation                string `json:"db_location"`
	PoenskelistenPort         int    `json:"poenskelisten_port"`
	PoenskelistenName         string `json:"poenskelisten_name"`
	PoenskelistenDescription  string `json:"poenskelisten_description"`
	PoenskelistenExternalURL  string `json:"poenskelisten_external_url"`
	PoenskelistenVersion      string `json:"poenskelisten_version"`
	PoenskelistenCurrency     string `json:"poenskelisten_currency"`
	PoenskelistenCurrencyPad  bool   `json:"poenskelisten_currency_pad"`
	PoenskelistenCurrencyLeft bool   `json:"poenskelisten_currency_left"`
	PoenskelistenEnvironment  string `json:"poenskelisten_environment"`
	PoenskelistenTestEmail    string `json:"poenskelisten_test_email"`
	PoenskelistenLogLevel     string `json:"poenskelisten_log_level"`
	MFAEnforced               bool   `json:"mfa_enforced"`
	MFARecoveryCodesEnabled   bool   `json:"mfa_recovery_codes_enabled"`
	OIDCEnabled               bool   `json:"oidc_enabled"`
	OIDCProviderName          string `json:"oidc_provider_name"`
	OIDCIssuerURL             string `json:"oidc_issuer_url"`
	OIDCClientID              string `json:"oidc_client_id"`
	OIDCClientSecret          string `json:"oidc_client_secret"`
	OIDCRedirectURL           string `json:"oidc_redirect_url"`
	OIDCAutoCreateUsers       bool   `json:"oidc_auto_create_users"`
	SMTPEnabled               bool   `json:"smtp_enabled"`
	SMTPHost                  string `json:"smtp_host"`
	SMTPPort                  int    `json:"smtp_port"`
	SMTPUsername              string `json:"smtp_username"`
	SMTPPassword              string `json:"smtp_password"`
	SMTPFrom                  string `json:"smtp_from"`
}

type UpdateCurrencyRequest struct {
	PoenskelistenCurrency     string `json:"poenskelisten_currency"`
	PoenskelistenCurrencyPad  bool   `json:"poenskelisten_currency_pad"`
	PoenskelistenCurrencyLeft bool   `json:"poenskelisten_currency_left"`
}
