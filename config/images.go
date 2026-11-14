package config

import "fmt"

// Bounds for the pixel count of an uploaded image, read from its header
// before it is decoded. Decoding takes memory in proportion to the pixel
// count, whatever the file size. A zero value in config.json means "use the
// default", like the token lifetimes.
const (
	DefaultMaxImagePixels = 24_000_000
	MinMaxImagePixels     = 1_000_000
	MaxMaxImagePixels     = 50_000_000
)

// ValidateMaxImagePixels accepts 0 (the default) or a value within bounds.
func ValidateMaxImagePixels(pixels int) error {
	if pixels != 0 && (pixels < MinMaxImagePixels || pixels > MaxMaxImagePixels) {
		return fmt.Errorf("image pixel limit must be between %d and %d, got %d", MinMaxImagePixels, MaxMaxImagePixels, pixels)
	}
	return nil
}

// MaxImagePixels is the largest width x height accepted for an uploaded image.
// An unset or out-of-range value falls back to the default.
func MaxImagePixels() int64 {
	pixels := ConfigFile.MaxImagePixels
	if pixels == 0 || ValidateMaxImagePixels(pixels) != nil {
		pixels = DefaultMaxImagePixels
	}
	return int64(pixels)
}
