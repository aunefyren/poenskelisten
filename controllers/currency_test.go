package controllers

import (
	"aunefyren/poenskelisten/config"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetCurrency(t *testing.T) {
	config.ConfigFile.PoenskelistenCurrency = "NOK"
	config.ConfigFile.PoenskelistenCurrencyPad = true
	config.ConfigFile.PoenskelistenCurrencyLeft = false

	code, body := runHandler(APIGetCurrency)
	if code != 200 {
		t.Fatalf("status = %d, want 200", code)
	}
	if body["currency"] != "NOK" {
		t.Errorf("currency = %v, want NOK", body["currency"])
	}
	if body["padding"] != true {
		t.Errorf("padding = %v, want true", body["padding"])
	}
	if body["left"] != false {
		t.Errorf("left = %v, want false", body["left"])
	}
}

func postUpdateCurrency(body string) (int, map[string]interface{}) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/api/admin/currency", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	APIUpdateCurrency(ctx)

	var parsed map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &parsed)
	return w.Code, parsed
}

func TestUpdateCurrencyBadJSON(t *testing.T) {
	if code, _ := postUpdateCurrency("not json"); code != 400 {
		t.Errorf("status = %d, want 400 for malformed JSON", code)
	}
}

func TestUpdateCurrencyRejectsInvalidCharacters(t *testing.T) {
	code, body := postUpdateCurrency(`{"poenskelisten_currency":"<script>","poenskelisten_currency_pad":true,"poenskelisten_currency_left":true}`)
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a currency string with disallowed characters", code)
	}
	if body["error"] == nil {
		t.Error("expected an error message")
	}
}

func TestUpdateCurrencySuccess(t *testing.T) {
	// APIUpdateCurrency persists via config.SaveConfig(), which writes to a
	// fixed absolute path (./files/config.json resolved once at process
	// start) - create+remove that directory around the test rather than
	// chdir, since the path was already resolved before any chdir could apply.
	if err := os.MkdirAll("files", 0755); err != nil {
		t.Fatalf("failed to create files dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll("files") })

	code, body := postUpdateCurrency(`{"poenskelisten_currency":"EUR","poenskelisten_currency_pad":true,"poenskelisten_currency_left":true}`)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, body)
	}
	if body["currency"] != "EUR" {
		t.Errorf("currency = %v, want EUR", body["currency"])
	}
	if config.ConfigFile.PoenskelistenCurrency != "EUR" {
		t.Errorf("config not updated, currency = %v", config.ConfigFile.PoenskelistenCurrency)
	}
	if _, err := os.Stat("files/config.json"); err != nil {
		t.Errorf("expected config.json to be written: %v", err)
	}
}

func TestUpdateCurrencySaveFailure(t *testing.T) {
	// With no "files" directory present, config.SaveConfig()'s WriteFile
	// fails and the handler takes its 500 path.
	os.RemoveAll("files")

	code, body := postUpdateCurrency(`{"poenskelisten_currency":"USD"}`)
	if code != 500 {
		t.Fatalf("status = %d, want 500 when the config file can't be written; body=%v", code, body)
	}
}
