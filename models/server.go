package models

type ServerInfoReply struct {
	Timezone                 string `json:"timezone"`
	PoenskelistenVersion     string `json:"poenskelisten_version"`
	PoenskelistenPort        int    `json:"poenskelisten_port"`
	PoenskelistenExternalURL string `json:"poenskelisten_external_url"`
	PoenskelistenEnvironment string `json:"poenskelisten_environment"`
	PoenskelistenTestEmail   string `json:"poenskelisten_test_email"`
	DatabaseType             string `json:"database_type"`
	SMTPEnabled              bool   `json:"smtp_enabled"`
}
