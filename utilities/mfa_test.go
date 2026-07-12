package utilities

import (
	"aunefyren/poenskelisten/config"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestGenerateAndValidateTOTP(t *testing.T) {
	config.ConfigFile.PoenskelistenName = "TestApp"

	secret, url, qrCode, err := GenerateTOTPSecret("user@example.com")
	if err != nil {
		t.Fatalf("GenerateTOTPSecret error: %v", err)
	}
	if secret == "" {
		t.Fatal("GenerateTOTPSecret returned empty secret")
	}
	if url == "" {
		t.Fatal("GenerateTOTPSecret returned empty otpauth URL")
	}
	if !strings.HasPrefix(qrCode, "data:image/png;base64,") {
		t.Errorf("qrCode = %q, want a PNG data URI", qrCode)
	}
	if len(qrCode) < len("data:image/png;base64,")+100 {
		t.Errorf("qrCode data URI looks too short to be a real image: %d chars", len(qrCode))
	}

	// A freshly computed code for the secret must validate.
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("totp.GenerateCode error: %v", err)
	}
	if !ValidateTOTPCode(secret, code) {
		t.Error("ValidateTOTPCode rejected a freshly generated code")
	}

	// A wrong code must not validate.
	if ValidateTOTPCode(secret, "000000") && code != "000000" {
		t.Error("ValidateTOTPCode accepted an incorrect code")
	}
}

func TestGenerateRecoveryCodes(t *testing.T) {
	codes, err := GenerateRecoveryCodes(RecoveryCodeCount)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes error: %v", err)
	}
	if len(codes) != RecoveryCodeCount {
		t.Errorf("got %d codes, want %d", len(codes), RecoveryCodeCount)
	}

	seen := map[string]bool{}
	for _, c := range codes {
		if c == "" {
			t.Error("got empty recovery code")
		}
		if seen[c] {
			t.Errorf("duplicate recovery code %q", c)
		}
		seen[c] = true
	}
}

func TestRecoveryCodeHashAndCheck(t *testing.T) {
	codes, err := GenerateRecoveryCodes(1)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes error: %v", err)
	}
	code := codes[0]

	hash, err := HashRecoveryCode(code)
	if err != nil {
		t.Fatalf("HashRecoveryCode error: %v", err)
	}
	if hash == code {
		t.Error("HashRecoveryCode returned the code unchanged")
	}

	if !CheckRecoveryCode(hash, code) {
		t.Error("CheckRecoveryCode rejected the correct code")
	}
	// Normalization: lower-case and dashed input still matches.
	if !CheckRecoveryCode(hash, "  "+code[:4]+"-"+code[4:]+"  ") {
		t.Error("CheckRecoveryCode rejected a differently-formatted correct code")
	}
	if CheckRecoveryCode(hash, "WRONGCODE") {
		t.Error("CheckRecoveryCode accepted an incorrect code")
	}
	if CheckRecoveryCode("", code) {
		t.Error("CheckRecoveryCode accepted an empty hash")
	}
}

func TestLooksLikeTOTPCode(t *testing.T) {
	cases := map[string]bool{
		"123456":  true,
		" 123456": true,
		"12345":   false,
		"1234567": false,
		"12345a":  false,
		"ABCDEF":  false,
		"":        false,
	}
	for input, want := range cases {
		if got := LooksLikeTOTPCode(input); got != want {
			t.Errorf("LooksLikeTOTPCode(%q) = %v, want %v", input, got, want)
		}
	}
}
