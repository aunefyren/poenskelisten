package config

import "testing"

func TestMaxImagePixels(t *testing.T) {
	original := ConfigFile.MaxImagePixels
	t.Cleanup(func() { ConfigFile.MaxImagePixels = original })

	cases := []struct {
		pixels int
		want   int64
		valid  bool
	}{
		{0, 24_000_000, true},
		{1_000_000, 1_000_000, true},
		{12_000_000, 12_000_000, true},
		{50_000_000, 50_000_000, true},
		{999_999, 24_000_000, false},
		{50_000_001, 24_000_000, false},
		{-1, 24_000_000, false},
	}
	for _, c := range cases {
		ConfigFile.MaxImagePixels = c.pixels
		if got := MaxImagePixels(); got != c.want {
			t.Errorf("MaxImagePixels() with %d = %d, want %d", c.pixels, got, c.want)
		}
		if err := ValidateMaxImagePixels(c.pixels); (err == nil) != c.valid {
			t.Errorf("ValidateMaxImagePixels(%d) error = %v, want valid %v", c.pixels, err, c.valid)
		}
	}
}
