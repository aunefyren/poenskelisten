package controllers

import (
	"aunefyren/poenskelisten/config"
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// BenchmarkDecodeImageUpload reports the memory one upload at the default
// pixel limit allocates (B/op), by format. Most of it is the decoded image,
// held until it is scaled down.
func BenchmarkDecodeImageUpload(b *testing.B) {
	const width, height = 6000, 4000
	if int64(width*height) != config.DefaultMaxImagePixels {
		b.Fatalf("%dx%d is not the default pixel limit", width, height)
	}

	gradient := func(img interface{ Set(int, int, color.Color) }, alpha bool) {
		for y := range height {
			for x := range width {
				a := uint8(255)
				if alpha {
					a = uint8(y)
				}
				img.Set(x, y, color.NRGBA{uint8(x), uint8(y), uint8(x + y), a})
			}
		}
	}
	encodePNG := func(img image.Image) []byte {
		buf := new(bytes.Buffer)
		if err := png.Encode(buf, img); err != nil {
			b.Fatal(err)
		}
		return buf.Bytes()
	}

	rgba := image.NewNRGBA(image.Rect(0, 0, width, height))
	gradient(rgba, false)
	jpegBuf := new(bytes.Buffer)
	if err := jpeg.Encode(jpegBuf, rgba, &jpeg.Options{Quality: 85}); err != nil {
		b.Fatal(err)
	}
	transparent := image.NewNRGBA(image.Rect(0, 0, width, height))
	gradient(transparent, true)
	deep := image.NewNRGBA64(image.Rect(0, 0, width, height))
	gradient(deep, true)

	for _, c := range []struct {
		name, mime string
		data       []byte
	}{
		{"jpeg", "image/jpeg", jpegBuf.Bytes()},
		{"png-rgb", "image/png", encodePNG(rgba)},
		{"png-rgba", "image/png", encodePNG(transparent)},
		{"png-rgba-16bit", "image/png", encodePNG(deep)},
	} {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := decodeUploadedImage(context.Background(), c.data, c.mime); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
