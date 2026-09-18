package utilities

import "testing"

func TestGenerateOpaqueTokenIsUniqueAndURLSafe(t *testing.T) {
	a, err := GenerateOpaqueToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := GenerateOpaqueToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a == b {
		t.Error("expected two distinct tokens")
	}
	if a == "" {
		t.Error("expected a non-empty token")
	}
	for _, r := range a {
		if r == '+' || r == '/' || r == '=' {
			t.Fatalf("token %q contains non-URL-safe base64 character %q", a, r)
		}
	}
}

func TestHashOpaqueTokenIsDeterministicAndDistinct(t *testing.T) {
	h1 := HashOpaqueToken("token-a")
	h2 := HashOpaqueToken("token-a")
	h3 := HashOpaqueToken("token-b")

	if h1 != h2 {
		t.Errorf("HashOpaqueToken is not deterministic: %q != %q", h1, h2)
	}
	if h1 == h3 {
		t.Error("expected different tokens to hash differently")
	}
	if len(h1) != 64 {
		t.Errorf("len(hash) = %d, want 64 (hex-encoded SHA-256)", len(h1))
	}
}
