package models

type ConfigStruct struct {
	Timezone          string `json:"timezone"`
	PrivateKey        string `json:"private_key"`
	DBUsername        string `json:"db_username"`
	DBPassword        string `json:"db_password"`
	DBName            string `json:"db_name"`
	DBIP              string `json:"db_ip"`
	DBPort            int    `json:"db_port"`
	PoenskelistenPort int    `json:"poenskelisten_port"`
}
