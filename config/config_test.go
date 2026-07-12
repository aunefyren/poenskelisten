package config

import "testing"

func TestGetPrivateKey(t *testing.T) {
	original := ConfigFile.PrivateKey
	t.Cleanup(func() { ConfigFile.PrivateKey = original })

	t.Run("valid key decodes", func(t *testing.T) {
		key, err := GenerateSecureKey(32)
		if err != nil {
			t.Fatalf("GenerateSecureKey returned error: %v", err)
		}
		ConfigFile.PrivateKey = key

		got, err := GetPrivateKey()
		if err != nil {
			t.Fatalf("GetPrivateKey returned error: %v", err)
		}
		if len(got) != 32 {
			t.Errorf("decoded key length = %d, want 32", len(got))
		}
	})

	t.Run("empty key errors", func(t *testing.T) {
		ConfigFile.PrivateKey = ""

		if _, err := GetPrivateKey(); err == nil {
			t.Error("GetPrivateKey accepted an empty key, want error")
		}
	})

	t.Run("invalid base64 errors", func(t *testing.T) {
		ConfigFile.PrivateKey = "!!!not-valid-base64!!!"

		if _, err := GetPrivateKey(); err == nil {
			t.Error("GetPrivateKey accepted invalid base64, want error")
		}
	})
}
