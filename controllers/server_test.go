package controllers

import (
	"aunefyren/poenskelisten/config"
	"testing"
)

func TestGetServerInfo(t *testing.T) {
	restoreConfig(t)
	config.ConfigFile.PoenskelistenName = "Test App"
	config.ConfigFile.PoenskelistenVersion = "1.2.3"
	config.ConfigFile.DBType = "sqlite"

	code, body := runHandler(APIGetServerInfo)
	if code != 200 {
		t.Fatalf("status = %d, want 200", code)
	}

	server, ok := body["server"].(map[string]interface{})
	if !ok {
		t.Fatalf("server = %v, want an object", body["server"])
	}
	if server["app_name"] != "Test App" {
		t.Errorf("app_name = %v, want 'Test App'", server["app_name"])
	}
	if server["poenskelisten_version"] != "1.2.3" {
		t.Errorf("poenskelisten_version = %v, want '1.2.3'", server["poenskelisten_version"])
	}
	if server["database_type"] != "sqlite" {
		t.Errorf("database_type = %v, want 'sqlite'", server["database_type"])
	}
	// Secrets must never be exposed.
	if _, hasPrivateKey := server["private_key"]; hasPrivateKey {
		t.Error("server info leaked a private_key field")
	}
}
