package utilities

import (
	"aunefyren/poenskelisten/config"
	"bytes"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"image/png"
	"strings"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

// qrCodeSize is the width/height in pixels of the generated enrollment QR image.
const qrCodeSize = 256

// RecoveryCodeCount is how many single-use backup codes are generated at
// enrollment.
const RecoveryCodeCount = 10

// recoveryCodeBytes is the amount of entropy per recovery code before base32
// encoding (10 bytes -> 16 base32 chars).
const recoveryCodeBytes = 10

// GenerateTOTPSecret creates a new TOTP secret for the given account (typically
// the user's e-mail). It returns the base32 secret to persist, the otpauth://
// provisioning URL, and a ready-to-display QR code as a PNG data URI. The QR is
// rendered server-side so the frontend needs no QR library and no external
// network access. The issuer is the configured application name so authenticator
// apps label the entry sensibly.
func GenerateTOTPSecret(accountName string) (secret string, otpauthURL string, qrCodeDataURI string, err error) {
	issuer := config.ConfigFile.PoenskelistenName
	if strings.TrimSpace(issuer) == "" {
		issuer = "Poenskelisten"
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
	})
	if err != nil {
		return "", "", "", err
	}

	img, err := key.Image(qrCodeSize, qrCodeSize)
	if err != nil {
		return "", "", "", err
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", "", "", err
	}
	qrCodeDataURI = "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	return key.Secret(), key.URL(), qrCodeDataURI, nil
}

// ValidateTOTPCode reports whether the provided 6-digit code is currently valid
// for the given base32 secret.
func ValidateTOTPCode(secret string, code string) bool {
	return totp.Validate(strings.TrimSpace(code), secret)
}

// GenerateRecoveryCodes returns n human-friendly single-use recovery codes. The
// codes are returned in plaintext for one-time display; callers persist only the
// hashes (see HashRecoveryCode).
func GenerateRecoveryCodes(n int) ([]string, error) {
	codes := make([]string, 0, n)
	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)

	for i := 0; i < n; i++ {
		buf := make([]byte, recoveryCodeBytes)
		if _, err := rand.Read(buf); err != nil {
			return nil, err
		}
		codes = append(codes, strings.ToUpper(encoder.EncodeToString(buf)))
	}

	return codes, nil
}

// NormalizeRecoveryCode canonicalizes user-entered recovery codes so formatting
// differences (case, surrounding whitespace, separating dashes/spaces) don't
// cause false mismatches.
func NormalizeRecoveryCode(code string) string {
	replacer := strings.NewReplacer(" ", "", "-", "")
	return strings.ToUpper(replacer.Replace(strings.TrimSpace(code)))
}

// HashRecoveryCode hashes a recovery code with bcrypt for storage. Unlike a
// user-chosen password, a recovery code carries ~80 bits of entropy, so
// bcrypt.DefaultCost is used rather than the heavier password cost: a slow KDF
// adds no meaningful brute-force resistance here but would make recovery logins
// (which may check several codes) needlessly slow.
func HashRecoveryCode(code string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(NormalizeRecoveryCode(code)), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckRecoveryCode reports whether a plaintext code matches a stored hash.
func CheckRecoveryCode(hash string, code string) bool {
	if hash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(NormalizeRecoveryCode(code))) == nil
}

// LooksLikeTOTPCode reports whether the input looks like a 6-digit TOTP code
// (rather than a recovery code), so the second-factor endpoints can decide which
// verification path to try.
func LooksLikeTOTPCode(code string) bool {
	trimmed := strings.TrimSpace(code)
	if len(trimmed) != 6 {
		return false
	}
	for _, r := range trimmed {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ErrNoTOTPSecret is returned when TOTP validation is attempted for a user that
// has no secret configured.
var ErrNoTOTPSecret = errors.New("no TOTP secret configured")
