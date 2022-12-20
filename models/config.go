package models

type ConfigStruct struct {
	Timezone                 string `json:"timezone"`
	PrivateKey               string `json:"private_key"`
	DBUsername               string `json:"db_username"`
	DBPassword               string `json:"db_password"`
	DBName                   string `json:"db_name"`
	DBIP                     string `json:"db_ip"`
	DBPort                   int    `json:"db_port"`
	PoenskelistenPort        int    `json:"poenskelisten_port"`
	PoenskelistenName        string `json:"poenskelisten_name"`
	PoenskelistenExternalURL string `json:"poenskelisten_external_url"`
	SMTPEnabled              bool   `json:"smtp_enabled"`
	SMTPHost                 string `json:"smtp_host"`
	SMTPPort                 int    `json:"smtp_port"`
	SMTPUsername             string `json:"smtp_username"`
	SMTPPassword             string `json:"smtp_password"`
	SMTPFrom                 string `json:"smtp_from"`
}
