package controllers

import (
	"aunefyren/poenskelisten/config"
	"testing"
)

func TestGetServerInfoLoginOriginsAndMCP(t *testing.T) {
	restoreConfig(t)
	config.ConfigFile.PoenskelistenExternalURL = ""
	config.ConfigFile.PoenskelistenPort = 9090
	config.ConfigFile.PoenskelistenAdditionalURLs = "http://192.168.1.10:9090,not-a-url"
	config.ConfigFile.MCPEnabled = true

	_, body := runHandler(APIGetServerInfo)
	server, _ := body["server"].(map[string]interface{})

	// The issuer is where login works without an external URL; the admin page
	// warns with it. Invalid additional entries aren't shown as working.
	if server["oauth_issuer"] != "http://localhost:9090" {
		t.Errorf("oauth_issuer = %v, want http://localhost:9090", server["oauth_issuer"])
	}
	additional, _ := server["poenskelisten_additional_urls"].([]interface{})
	if len(additional) != 1 || additional[0] != "http://192.168.1.10:9090" {
		t.Errorf("poenskelisten_additional_urls = %v, want only the valid entry", server["poenskelisten_additional_urls"])
	}
	if server["mcp_enabled"] != true || server["mcp_endpoint"] != "http://localhost:9090/mcp" {
		t.Errorf("mcp_enabled = %v, mcp_endpoint = %v", server["mcp_enabled"], server["mcp_endpoint"])
	}
}

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
	for _, secret := range []string{"oauth_signing_key", "db_password", "smtp_password", "oidc_client_secret"} {
		if _, leaked := server[secret]; leaked {
			t.Errorf("server info leaked %s", secret)
		}
	}
	if _, hasPrivateKey := server["private_key"]; hasPrivateKey {
		t.Error("server info leaked a private_key field")
	}
}
