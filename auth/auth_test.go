package auth

import (
	"aunefyren/poenskelisten/config"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// setupAuthTestConfig points the package-global config at a freshly generated,
// valid base64 signing key so token generation/validation works without a
// config.json on disk. It returns the userID used for the default claims.
func setupAuthTestConfig(t *testing.T) {
	t.Helper()

	key, err := config.GenerateSecureKey(64)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}
	config.ConfigFile.PrivateKey = key
	config.ConfigFile.PoenskelistenName = "TestIssuer"
}

func TestGenerateJWTAndParseTokenRoundTrip(t *testing.T) {
	setupAuthTestConfig(t)

	userID := uuid.New()
	tokenString, err := GenerateJWT("Ada", "Lovelace", "ada@example.com", userID, true, true)
	if err != nil {
		t.Fatalf("GenerateJWT returned error: %v", err)
	}
	if tokenString == "" {
		t.Fatal("GenerateJWT returned empty token")
	}

	claims, err := ParseToken(tokenString)
	if err != nil {
		t.Fatalf("ParseToken returned error: %v", err)
	}

	if claims.Firstname != "Ada" {
		t.Errorf("Firstname = %q, want %q", claims.Firstname, "Ada")
	}
	if claims.Lastname != "Lovelace" {
		t.Errorf("Lastname = %q, want %q", claims.Lastname, "Lovelace")
	}
	if claims.Email != "ada@example.com" {
		t.Errorf("Email = %q, want %q", claims.Email, "ada@example.com")
	}
	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}
	if !claims.Admin {
		t.Error("Admin = false, want true")
	}
	if !claims.Verified {
		t.Error("Verified = false, want true")
	}
	if claims.Issuer != "TestIssuer" {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, "TestIssuer")
	}
	if claims.ExpiresAt == nil {
		t.Error("ExpiresAt is nil, want a value")
	}
}

func TestValidateTokenValid(t *testing.T) {
	setupAuthTestConfig(t)

	tokenString, err := GenerateJWT("Ada", "Lovelace", "ada@example.com", uuid.New(), false, true)
	if err != nil {
		t.Fatalf("GenerateJWT returned error: %v", err)
	}

	if err := ValidateToken(tokenString, false); err != nil {
		t.Errorf("ValidateToken rejected a fresh token: %v", err)
	}
}

func TestValidateTokenExpired(t *testing.T) {
	setupAuthTestConfig(t)

	now := time.Now()
	claims := &JWTClaim{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
			NotBefore: jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
		},
	}
	tokenString, err := GenerateJWTFromClaims(claims)
	if err != nil {
		t.Fatalf("GenerateJWTFromClaims returned error: %v", err)
	}

	if err := ValidateToken(tokenString, false); err == nil {
		t.Error("ValidateToken accepted an expired token, want error")
	}
}

func TestValidateTokenNotYetValid(t *testing.T) {
	setupAuthTestConfig(t)

	now := time.Now()
	claims := &JWTClaim{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(now.Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	tokenString, err := GenerateJWTFromClaims(claims)
	if err != nil {
		t.Fatalf("GenerateJWTFromClaims returned error: %v", err)
	}

	if err := ValidateToken(tokenString, false); err == nil {
		t.Error("ValidateToken accepted a not-yet-valid token, want error")
	}
}

func TestValidateTokenMissingClaims(t *testing.T) {
	setupAuthTestConfig(t)

	// No ExpiresAt / NotBefore: the signature is valid, but the manual guard in
	// ValidateToken must reject the token for lacking these claims.
	claims := &JWTClaim{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}
	tokenString, err := GenerateJWTFromClaims(claims)
	if err != nil {
		t.Fatalf("GenerateJWTFromClaims returned error: %v", err)
	}

	if err := ValidateToken(tokenString, false); err == nil {
		t.Error("ValidateToken accepted a token missing exp/nbf, want error")
	}
}

func TestValidateTokenTamperedSignature(t *testing.T) {
	setupAuthTestConfig(t)

	tokenString, err := GenerateJWT("Ada", "Lovelace", "ada@example.com", uuid.New(), false, true)
	if err != nil {
		t.Fatalf("GenerateJWT returned error: %v", err)
	}

	// Tamper with the signature segment. Flip its *first* character rather than
	// its last: the final base64url char of a 32-byte HMAC signature carries
	// unused low bits, so flipping it can decode to the same signature bytes and
	// still verify. The first char always maps to distinct bytes.
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 token segments, got %d", len(parts))
	}
	sig := []byte(parts[2])
	if sig[0] == 'A' {
		sig[0] = 'B'
	} else {
		sig[0] = 'A'
	}
	parts[2] = string(sig)
	tampered := strings.Join(parts, ".")

	if err := ValidateToken(tampered, false); err == nil {
		t.Error("ValidateToken accepted a token with a tampered signature, want error")
	}
}

func TestValidateTokenWrongKey(t *testing.T) {
	setupAuthTestConfig(t)

	tokenString, err := GenerateJWT("Ada", "Lovelace", "ada@example.com", uuid.New(), false, true)
	if err != nil {
		t.Fatalf("GenerateJWT returned error: %v", err)
	}

	// Rotate the signing key: the previously issued token must no longer verify.
	newKey, err := config.GenerateSecureKey(64)
	if err != nil {
		t.Fatalf("failed to generate replacement key: %v", err)
	}
	config.ConfigFile.PrivateKey = newKey

	if err := ValidateToken(tokenString, false); err == nil {
		t.Error("ValidateToken accepted a token signed with a different key, want error")
	}
}

func TestValidateTokenGarbage(t *testing.T) {
	setupAuthTestConfig(t)

	for _, garbage := range []string{"", "not-a-token", "a.b.c", "not.a.token"} {
		if err := ValidateToken(garbage, false); err == nil {
			t.Errorf("ValidateToken accepted garbage input %q, want error", garbage)
		}
	}
}

func TestValidateTokenAdmin(t *testing.T) {
	setupAuthTestConfig(t)

	adminToken, err := GenerateJWT("Ada", "Lovelace", "ada@example.com", uuid.New(), true, true)
	if err != nil {
		t.Fatalf("GenerateJWT (admin) returned error: %v", err)
	}
	userToken, err := GenerateJWT("Bob", "User", "bob@example.com", uuid.New(), false, true)
	if err != nil {
		t.Fatalf("GenerateJWT (user) returned error: %v", err)
	}

	if err := ValidateToken(adminToken, true); err != nil {
		t.Errorf("ValidateToken(admin=true) rejected an admin token: %v", err)
	}
	if err := ValidateToken(userToken, true); err == nil {
		t.Error("ValidateToken(admin=true) accepted a non-admin token, want error")
	}
	if err := ValidateToken(userToken, false); err != nil {
		t.Errorf("ValidateToken(admin=false) rejected a valid non-admin token: %v", err)
	}
}

func TestValidateTokenAdminReturnsSentinel(t *testing.T) {
	setupAuthTestConfig(t)

	userToken, err := GenerateJWT("Bob", "User", "bob@example.com", uuid.New(), false, true)
	if err != nil {
		t.Fatalf("GenerateJWT returned error: %v", err)
	}

	err = ValidateToken(userToken, true)
	if err == nil {
		t.Fatal("ValidateToken(admin=true) accepted a non-admin token, want error")
	}
	if !errors.Is(err, ErrNotAdmin) {
		t.Errorf("ValidateToken returned %v, want ErrNotAdmin", err)
	}
}

func TestParseTokenRejectsNonHMAC(t *testing.T) {
	setupAuthTestConfig(t)

	// A token signed with the "none" algorithm must be rejected: the keyfunc
	// only trusts HMAC signing.
	claims := &JWTClaim{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	noneToken, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("failed to sign 'none' token: %v", err)
	}

	if _, err := ParseToken(noneToken); err == nil {
		t.Error("ParseToken accepted an alg:none token, want error")
	}
	if err := ValidateToken(noneToken, false); err == nil {
		t.Error("ValidateToken accepted an alg:none token, want error")
	}
}

func TestParseTokenGarbage(t *testing.T) {
	setupAuthTestConfig(t)

	claims, err := ParseToken("not.a.valid.token")
	if err == nil {
		t.Error("ParseToken accepted garbage input, want error")
	}
	if claims != nil {
		t.Errorf("ParseToken returned non-nil claims on error: %v", claims)
	}
}

func TestParseTokenStripsBearerPrefix(t *testing.T) {
	setupAuthTestConfig(t)

	tokenString, err := GenerateJWT("Ada", "Lovelace", "ada@example.com", uuid.New(), false, true)
	if err != nil {
		t.Fatalf("GenerateJWT returned error: %v", err)
	}

	for _, prefix := range []string{"Bearer ", "bearer ", "BEARER "} {
		if _, err := ParseToken(prefix + tokenString); err != nil {
			t.Errorf("ParseToken(%q + token) returned error: %v", prefix, err)
		}
		if err := ValidateToken(prefix+tokenString, false); err != nil {
			t.Errorf("ValidateToken(%q + token) returned error: %v", prefix, err)
		}
	}
}

func TestValidateTokenGetClaims(t *testing.T) {
	setupAuthTestConfig(t)

	userID := uuid.New()
	tokenString, err := GenerateJWT("Ada", "Lovelace", "ada@example.com", userID, true, true)
	if err != nil {
		t.Fatalf("GenerateJWT returned error: %v", err)
	}

	claims, err := ValidateTokenGetClaims(tokenString, true)
	if err != nil {
		t.Fatalf("ValidateTokenGetClaims returned error: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}

	// Invalid token yields nil claims and an error.
	if claims, err := ValidateTokenGetClaims("garbage", false); err == nil || claims != nil {
		t.Errorf("ValidateTokenGetClaims(garbage) = (%v, %v), want (nil, error)", claims, err)
	}
}

func TestGenerateJWTFromClaimsRoundTrip(t *testing.T) {
	setupAuthTestConfig(t)

	userID := uuid.New()
	now := time.Now()
	original := &JWTClaim{
		Firstname: "Grace",
		Lastname:  "Hopper",
		Email:     "grace@example.com",
		Admin:     false,
		Verified:  true,
		UserID:    userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "TestIssuer",
		},
	}

	tokenString, err := GenerateJWTFromClaims(original)
	if err != nil {
		t.Fatalf("GenerateJWTFromClaims returned error: %v", err)
	}

	parsed, err := ParseToken(tokenString)
	if err != nil {
		t.Fatalf("ParseToken returned error: %v", err)
	}

	if parsed.UserID != userID {
		t.Errorf("UserID = %v, want %v", parsed.UserID, userID)
	}
	if parsed.Email != "grace@example.com" {
		t.Errorf("Email = %q, want %q", parsed.Email, "grace@example.com")
	}
	if !strings.EqualFold(parsed.Firstname, "Grace") {
		t.Errorf("Firstname = %q, want %q", parsed.Firstname, "Grace")
	}
	if err := ValidateToken(tokenString, false); err != nil {
		t.Errorf("ValidateToken rejected a token built from valid claims: %v", err)
	}
}
