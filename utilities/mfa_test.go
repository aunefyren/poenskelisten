package utilities

import (
	"aunefyren/poenskelisten/config"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestGenerateAndValidateTOTP(t *testing.T) {
	original := config.ConfigFile
	t.Cleanup(func() { config.ConfigFile = original })
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
	if _, valid := TOTPStep(secret, code, time.Now()); !valid {
		t.Error("TOTPStep rejected a freshly generated code")
	}
}

// TOTPStep tells which 30-second step a code belongs to, accepting the steps
// next to now as totp.Validate does, so a caller can accept each code once.
func TestTOTPStep(t *testing.T) {
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "Test", AccountName: "ada@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_015, 0)
	current := now.Unix() / 30
	for offset, want := range map[int64]bool{-2: false, -1: true, 0: true, 1: true, 2: false} {
		code, err := totp.GenerateCode(key.Secret(), time.Unix((current+offset)*30, 0))
		if err != nil {
			t.Fatal(err)
		}
		step, valid := TOTPStep(key.Secret(), " "+code+" ", now)
		if valid != want || (valid && step != current+offset) {
			t.Errorf("offset %d: TOTPStep = (%d, %v), want step %d valid %v", offset, step, valid, current+offset, want)
		}
	}
	if _, valid := TOTPStep(key.Secret(), "not-a-code", now); valid {
		t.Error("TOTPStep accepted a malformed code")
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

func TestGenerateTOTPSecretDefaultsIssuer(t *testing.T) {
	orig := config.ConfigFile.PoenskelistenName
	t.Cleanup(func() { config.ConfigFile.PoenskelistenName = orig })
	config.ConfigFile.PoenskelistenName = "   "

	_, url, _, err := GenerateTOTPSecret("user@example.com")
	if err != nil {
		t.Fatalf("GenerateTOTPSecret error: %v", err)
	}
	if !strings.Contains(url, "issuer=Poenskelisten") {
		t.Errorf("otpauth URL = %q, want the default issuer when the app name is blank", url)
	}
}

func TestGenerateTOTPSecretRequiresAccountName(t *testing.T) {
	secret, url, qrCode, err := GenerateTOTPSecret("")
	if err == nil {
		t.Fatal("GenerateTOTPSecret(\"\") succeeded, want an error")
	}
	if secret != "" || url != "" || qrCode != "" {
		t.Errorf("expected empty results on error, got %q, %q, %d-char QR", secret, url, len(qrCode))
	}
}

func TestHashRecoveryCodeTooLong(t *testing.T) {
	// bcrypt refuses inputs over 72 bytes.
	if hash, err := HashRecoveryCode(strings.Repeat("A", 73)); err == nil {
		t.Errorf("HashRecoveryCode of a 73-byte code = %q, want an error", hash)
	}
}
