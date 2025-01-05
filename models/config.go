package models

type ConfigStruct struct {
	Timezone                 string `json:"timezone"`
	PrivateKey               string `json:"private_key"`
	DBType                   string `json:"db_type"`
	DBUsername               string `json:"db_username"`
	DBPassword               string `json:"db_password"`
	DBName                   string `json:"db_name"`
	DBIP                     string `json:"db_ip"`
	DBPort                   int    `json:"db_port"`
	DBSSL                    bool   `json:"db_ssl"`
	DBLocation               string `json:"db_location"`
	PoenskelistenPort        int    `json:"poenskelisten_port"`
	PoenskelistenName        string `json:"poenskelisten_name"`
	PoenskelistenExternalURL string `json:"poenskelisten_external_url"`
	PoenskelistenVersion     string `json:"poenskelisten_version"`
	PoenskelistenCurrency    string `json:"poenskelisten_currency"`
	PoenskelistenCurrencyPad bool   `json:"poenskelisten_currency_pad"`
	PoenskelistenEnvironment string `json:"poenskelisten_environment"`
	PoenskelistenTestEmail   string `json:"poenskelisten_test_email"`
	SMTPEnabled              bool   `json:"smtp_enabled"`
	SMTPHost                 string `json:"smtp_host"`
	SMTPPort                 int    `json:"smtp_port"`
	SMTPUsername             string `json:"smtp_username"`
	SMTPPassword             string `json:"smtp_password"`
	SMTPFrom                 string `json:"smtp_from"`
}

type UpdateCurrencyrequest struct {
	PoenskelistenCurrency    string `json:"poenskelisten_currency"`
	PoenskelistenCurrencyPad bool   `json:"poenskelisten_currency_pad"`
}
