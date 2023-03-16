package models

type ServerInfoReply struct {
	Timezone             string `json:"timezone"`
	PoenskelistenVersion string `json:"poenskelisten_version"`
}
