package controllers

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
)

// imageTestJFIF is the start of a JPEG with a JFIF segment.
var imageTestJFIF = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0, 1, 1, 0, 0, 1, 0, 1, 0, 0}

// imageTestJPEGFrame is a frame header with one sampling byte (h<<4 | v) per
// component, identified 1, 2, 3... marker is 0xC0 for baseline or 0xC2 for
// progressive.
func imageTestJPEGFrame(marker byte, width, height int, sampling ...byte) []byte {
	frame := []byte{8, byte(height >> 8), byte(height), byte(width >> 8), byte(width), byte(len(sampling))}
	for i, hv := range sampling {
		frame = append(frame, byte(i+1), hv, 0)
	}
	return append([]byte{0xFF, marker, byte((len(frame) + 2) >> 8), byte(len(frame) + 2)}, frame...)
}

// imageTestJPEGScan is the start of a scan, up to which a header is read.
var imageTestJPEGScan = []byte{0xFF, 0xDA, 0x00, 0x02}

// imageTestJPEGHeader builds the start of a JFIF JPEG, up to its first scan
// (enough for jpeg.DecodeConfig).
func imageTestJPEGHeader(marker byte, width, height int, sampling ...byte) []byte {
	return imageTestJoin(imageTestJFIF, imageTestJPEGFrame(marker, width, height, sampling...), imageTestJPEGScan)
}

func imageTestJoin(parts ...[]byte) []byte {
	return bytes.Join(parts, nil)
}

// imageTestPNGChunk encodes one PNG chunk.
func imageTestPNGChunk(kind string, data []byte) []byte {
	chunk := append([]byte(kind), data...)
	b := binary.BigEndian.AppendUint32(nil, uint32(len(data)))
	b = append(b, chunk...)
	return binary.BigEndian.AppendUint32(b, crc32.ChecksumIEEE(chunk))
}

// imageTestInterlacedPNG encodes a blank width x height 8-bit RGBA PNG with
// Adam7 interlacing, which image/png can decode but not encode.
func imageTestInterlacedPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	ihdr := binary.BigEndian.AppendUint32(nil, uint32(width))
	ihdr = binary.BigEndian.AppendUint32(ihdr, uint32(height))
	ihdr = append(ihdr, 8, 6, 0, 0, 1)

	passes := []struct{ x, y, dx, dy int }{{0, 0, 8, 8}, {4, 0, 8, 8}, {0, 4, 4, 8}, {2, 0, 4, 4}, {0, 2, 2, 4}, {1, 0, 2, 2}, {0, 1, 1, 2}}
	raw := new(bytes.Buffer)
	for _, p := range passes {
		passWidth := (width - p.x + p.dx - 1) / p.dx
		passHeight := (height - p.y + p.dy - 1) / p.dy
		if passWidth <= 0 || passHeight <= 0 {
			continue
		}
		row := make([]byte, 1+4*passWidth) // filter type 0, then the pixels
		for range passHeight {
			raw.Write(row)
		}
	}
	compressed := new(bytes.Buffer)
	w := zlib.NewWriter(compressed)
	if _, err := w.Write(raw.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	b := []byte("\x89PNG\r\n\x1a\n")
	b = append(b, imageTestPNGChunk("IHDR", ihdr)...)
	b = append(b, imageTestPNGChunk("IDAT", compressed.Bytes())...)
	return append(b, imageTestPNGChunk("IEND", nil)...)
}

func imageTestEncodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func imageTestReadFixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestFittedSizeMatchesFitWithin(t *testing.T) {
	for _, size := range [][2]int{{1600, 1200}, {1200, 1600}, {999, 1000}, {1, 5000}, {5000, 1}, {1001, 1001}} {
		width, height := fittedSize(size[0], size[1], maxImageWidth, maxImageHeight)
		bounds := fitWithin(image.NewGray(image.Rect(0, 0, size[0], size[1])), maxImageWidth, maxImageHeight).Bounds()
		if bounds.Dx() != width || bounds.Dy() != height {
			t.Errorf("%v: fittedSize = %dx%d, fitWithin made %dx%d", size, width, height, bounds.Dx(), bounds.Dy())
		}
	}
}

func TestResizeScratchBytes(t *testing.T) {
	for _, c := range []struct {
		width, height int
		want          int64
	}{
		{1000, 1000, 0},
		{800, 600, 0},
		{6000, 4000, 1000 * 4000 * 32},
		{4000, 6000, 667 * 6000 * 32},
		// The scaled width can't go below one column, so a very tall, narrow
		// image needs scratch for every source row.
		{1, 24_000_000, 24_000_000 * 32},
	} {
		if got := resizeScratchBytes(c.width, c.height); got != c.want {
			t.Errorf("%dx%d: resizeScratchBytes = %d, want %d", c.width, c.height, got, c.want)
		}
	}
}

func TestReadJPEGHeader(t *testing.T) {
	header, ok := readJPEGHeader(imageTestJPEGHeader(0xC2, 10, 10, 0x22, 0x11, 0x11))
	if !ok || !header.progressive || !header.jfif || !header.complete || len(header.sampling) != 3 ||
		header.sampling[0] != [2]int{2, 2} || header.sampling[1] != [2]int{1, 1} || string(header.ids) != "\x01\x02\x03" {
		t.Errorf("readJPEGHeader = (%+v, %v), want a complete progressive 4:2:0 JFIF header", header, ok)
	}

	// Fill bytes before a marker are skipped, as image/jpeg does.
	frame := imageTestJPEGFrame(0xC0, 10, 10, 0x11)
	if header, ok := readJPEGHeader(imageTestJoin(imageTestJFIF, []byte{0xFF, 0xFF}, frame, imageTestJPEGScan)); !ok || header.progressive || len(header.sampling) != 1 {
		t.Errorf("with fill bytes: readJPEGHeader = (%+v, %v), want baseline grayscale", header, ok)
	}

	// Markers after the frame header still count; a file that ends before
	// its first scan has a frame header but is incomplete.
	adobe := []byte{0xFF, 0xEE, 0x00, 0x0E, 'A', 'd', 'o', 'b', 'e', 0, 100, 0, 0, 0, 0, 0}
	if header, ok := readJPEGHeader(imageTestJoin([]byte{0xFF, 0xD8}, frame, adobe, imageTestJPEGScan)); !ok || !header.adobe || header.adobeTransform != 0 || header.jfif {
		t.Errorf("with an Adobe segment: readJPEGHeader = (%+v, %v), want transform 0", header, ok)
	}
	for name, b := range map[string][]byte{
		"ends after the frame":  imageTestJoin(imageTestJFIF, frame),
		"truncated segment":     imageTestJoin(imageTestJFIF, frame, adobe[:8]),
		"truncated length":      imageTestJoin(imageTestJFIF, frame, []byte{0xFF, 0xEE, 0x00}),
		"ends with fill bytes":  imageTestJoin(imageTestJFIF, frame, []byte{0xFF, 0xFF}),
		"restart marker before": imageTestJoin([]byte{0xFF, 0xD8, 0xFF, 0xD0}, frame),
	} {
		if header, ok := readJPEGHeader(b); !ok || header.complete {
			t.Errorf("%s: readJPEGHeader = (%+v, %v), want an incomplete header", name, header, ok)
		}
	}

	for name, b := range map[string][]byte{
		"empty":                 nil,
		"not a JPEG":            []byte("\x89PNG\r\n\x1a\n"),
		"truncated frame":       imageTestJoin(imageTestJFIF, frame[:len(frame)-2]),
		"no frame":              imageTestJoin(imageTestJFIF, imageTestJPEGScan),
		"end before the frame":  {0xFF, 0xD8, 0xFF, 0xD9},
		"bytes between markers": imageTestJoin(imageTestJFIF, []byte{0x00}, frame),
		"no components":         {0xFF, 0xD8, 0xFF, 0xC0, 0x00, 0x08, 8, 0, 1, 0, 1, 0},
		"short frame":           {0xFF, 0xD8, 0xFF, 0xC0, 0x00, 0x04, 8, 0},
		"bad length":            {0xFF, 0xD8, 0xFF, 0xC0, 0x00, 0x01},
		"two frames":            imageTestJoin(imageTestJFIF, frame, frame, imageTestJPEGScan),
	} {
		if _, ok := readJPEGHeader(b); ok {
			t.Errorf("%s: readJPEGHeader ok, want it refused", name)
		}
	}
}

func TestJPEGDecodeBytesPerPixel(t *testing.T) {
	rgbByIDs := imageTestJoin([]byte{0xFF, 0xD8}, imageTestJPEGFrame(0xC0, 10, 10, 0x11, 0x11, 0x11), imageTestJPEGScan)
	rgbByIDs[2+4+6], rgbByIDs[2+4+9], rgbByIDs[2+4+12] = 'R', 'G', 'B'
	adobeRGB := []byte{0xFF, 0xEE, 0x00, 0x0E, 'A', 'd', 'o', 'b', 'e', 0, 100, 0, 0, 0, 0, 0}

	for _, c := range []struct {
		name string
		raw  []byte
		want float64
	}{
		{"baseline gray", imageTestJPEGHeader(0xC0, 10, 10, 0x22), 1},
		{"progressive gray", imageTestJPEGHeader(0xC2, 10, 10, 0x11), 5},
		{"baseline 4:2:0", imageTestJPEGHeader(0xC0, 10, 10, 0x22, 0x11, 0x11), 1.5},
		{"baseline 4:4:4", imageTestJPEGHeader(0xC0, 10, 10, 0x11, 0x11, 0x11), 3},
		{"progressive 4:2:0", imageTestJPEGHeader(0xC2, 10, 10, 0x22, 0x11, 0x11), 1.5 + 6},
		{"progressive 4:4:4", imageTestJPEGHeader(0xC2, 10, 10, 0x11, 0x11, 0x11), 3 + 12},
		// Converted into an RGBA copy: RGB by its component identifiers or
		// Adobe transform, or possibly RGB as the header is incomplete.
		{"RGB by identifiers", rgbByIDs, 3 + 4},
		{"RGB by Adobe transform", imageTestJoin([]byte{0xFF, 0xD8}, imageTestJPEGFrame(0xC0, 10, 10, 0x11, 0x11, 0x11), adobeRGB, imageTestJPEGScan), 3 + 4},
		{"YCbCr without JFIF", imageTestJoin([]byte{0xFF, 0xD8}, imageTestJPEGFrame(0xC0, 10, 10, 0x22, 0x11, 0x11), imageTestJPEGScan), 1.5},
		{"incomplete 4:2:0", imageTestJoin(imageTestJFIF, imageTestJPEGFrame(0xC0, 10, 10, 0x22, 0x11, 0x11)), 1.5 + 4},
		{"baseline CMYK", imageTestJPEGHeader(0xC0, 10, 10, 0x11, 0x11, 0x11, 0x11), 4 + 4},
		{"progressive CMYK", imageTestJPEGHeader(0xC2, 10, 10, 0x11, 0x11, 0x11, 0x11), 4 + 4 + 16},
		{"zero sampling", imageTestJPEGHeader(0xC0, 10, 10, 0x00, 0x11, 0x11), maxJPEGDecodeBytesPerPixel},
		{"unreadable", []byte("garbage"), maxJPEGDecodeBytesPerPixel},
	} {
		if got := jpegDecodeBytesPerPixel(c.raw); got != c.want {
			t.Errorf("%s: jpegDecodeBytesPerPixel = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestPNGDecodeBytesPerPixel(t *testing.T) {
	ihdr := func(depth, colorType, interlace byte) []byte {
		header := imageTestPNGHeader(10, 10)
		header[24], header[25], header[28] = depth, colorType, interlace
		return header
	}
	for _, c := range []struct {
		name string
		raw  []byte
		want float64
	}{
		{"8-bit gray, as if transparent", ihdr(8, 0, 0), 4},
		{"16-bit gray", ihdr(16, 0, 0), 8},
		{"8-bit RGB", ihdr(8, 2, 0), 4},
		{"paletted", ihdr(8, 3, 0), 1},
		{"16-bit RGBA", ihdr(16, 6, 0), 8},
		{"interlaced 16-bit RGBA", ihdr(16, 6, 1), 16},
		{"unreadable", []byte("garbage"), maxPNGDecodeBytesPerPixel},
	} {
		if got := pngDecodeBytesPerPixel(c.raw); got != c.want {
			t.Errorf("%s: pngDecodeBytesPerPixel = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestDecodeCostOfUnknownFormatIsTheWorstCase(t *testing.T) {
	header := image.Config{Width: 100, Height: 100}
	if got, want := decodeCost(nil, "gif", header), int64(100*100*maxJPEGDecodeBytesPerPixel+decodeOverheadBytes); got != want {
		t.Errorf("decodeCost = %d, want %d", got, want)
	}
}

// The estimate must cover what decoding actually allocates, or the budget
// doesn't bound memory. Allocations are counted in total, which is at least
// the peak. Only part of the fixed allowance is granted here, so that a
// size-dependent part the model got wrong can't hide in it.
func TestDecodeCostCoversAllocations(t *testing.T) {
	const width, height = 1600, 1200
	gradient := image.NewNRGBA64(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			gradient.Set(x, y, color.NRGBA64{uint16(x * 40), uint16(y * 50), 0, uint16(y * 50)})
		}
	}
	rgba := image.NewNRGBA(gradient.Bounds())
	gray := image.NewGray(gradient.Bounds())
	for y := range height {
		for x := range width {
			rgba.Set(x, y, gradient.At(x, y))
			gray.Set(x, y, gradient.At(x, y))
		}
	}
	jpegBuf := new(bytes.Buffer)
	if err := jpeg.Encode(jpegBuf, rgba, &jpeg.Options{Quality: 85}); err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct {
		name, mime string
		raw        []byte
	}{
		{"baseline 4:2:0 JPEG", "image/jpeg", jpegBuf.Bytes()},
		{"baseline CMYK JPEG", "image/jpeg", imageTestReadFixture(t, "baseline-cmyk.jpg")},
		{"progressive 4:2:0 JPEG", "image/jpeg", imageTestReadFixture(t, "progressive-420.jpg")},
		{"progressive 4:4:4 JPEG", "image/jpeg", imageTestReadFixture(t, "progressive-444.jpg")},
		{"progressive CMYK JPEG", "image/jpeg", imageTestReadFixture(t, "progressive-cmyk.jpg")},
		{"gray PNG", "image/png", imageTestEncodePNG(t, gray)},
		{"RGBA PNG", "image/png", imageTestEncodePNG(t, rgba)},
		{"16-bit RGBA PNG", "image/png", imageTestEncodePNG(t, gradient)},
		{"interlaced RGBA PNG", "image/png", imageTestInterlacedPNG(t, width, height)},
	} {
		header, format, err := image.DecodeConfig(bytes.NewReader(c.raw))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if header.Width != width || header.Height != height {
			t.Fatalf("%s: fixture is %dx%d, want %dx%d", c.name, header.Width, header.Height, width, height)
		}
		allowed := decodeCost(c.raw, format, header) - decodeOverheadBytes + 16<<20

		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		if _, err := decodeUploadedImage(context.Background(), c.raw, c.mime); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		runtime.ReadMemStats(&after)

		allocated := int64(after.TotalAlloc - before.TotalAlloc)
		t.Logf("%s: allocated %d MB, estimated %d MB", c.name, allocated>>20, allowed>>20)
		if allocated > allowed {
			t.Errorf("%s: decoding allocated %d bytes, more than the %d estimated", c.name, allocated, allowed)
		}
	}
}

func TestMaxDecodeCost(t *testing.T) {
	imageTestPixelLimit(t, 0)
	// 24 MP at 8 bytes, the scratch to resize a 4898x4898 image, the overhead.
	if got, want := maxDecodeCost(), int64(24_000_000*8+1000*4898*32+decodeOverheadBytes); got != want {
		t.Errorf("maxDecodeCost = %d, want %d", got, want)
	}

	// Any shape at the pixel limit is within the cap at 8 bytes per pixel,
	// unless it's too narrow for the resize scratch to stay small.
	for _, size := range [][2]int{{6000, 4000}, {4000, 6000}, {4898, 4898}, {24000, 1000}, {1000, 24000}} {
		header := image.Config{Width: size[0], Height: size[1]}
		if cost := decodeCost(imageTestPNGHeaderDepth(size[0], size[1], 16, 0), "png", header); cost > maxDecodeCost() {
			t.Errorf("%v 16-bit PNG: decodeCost %d over the cap %d", size, cost, maxDecodeCost())
		}
	}

	imageTestPixelLimit(t, 1_000_000)
	if got, want := maxDecodeCost(), int64(1_000_000*8+decodeOverheadBytes); got != want {
		t.Errorf("1 MP limit: maxDecodeCost = %d, want %d", got, want)
	}
}

// imageTestPNGHeaderDepth is the signature and IHDR chunk of an RGBA PNG with
// the given bit depth and interlace method.
func imageTestPNGHeaderDepth(width, height int, depth, interlace byte) []byte {
	ihdr := binary.BigEndian.AppendUint32(nil, uint32(width))
	ihdr = binary.BigEndian.AppendUint32(ihdr, uint32(height))
	ihdr = append(ihdr, depth, 6, 0, 0, interlace)
	return append([]byte("\x89PNG\r\n\x1a\n"), imageTestPNGChunk("IHDR", ihdr)...)
}

// Images within the pixel limit that would take more memory than one decode
// may are refused before decoding: progressive CMYK at the default limit,
// and a very tall, narrow image.
func TestDecodeImageUploadRefusesCostlyImage(t *testing.T) {
	imageTestPixelLimit(t, 0)
	for _, c := range []struct {
		name, mime string
		raw        []byte
		refused    bool
	}{
		{"progressive CMYK 6000x4000", "image/jpeg", imageTestJPEGHeader(0xC2, 6000, 4000, 0x11, 0x11, 0x11, 0x11), true},
		{"progressive 4:4:4 6000x4000", "image/jpeg", imageTestJPEGHeader(0xC2, 6000, 4000, 0x11, 0x11, 0x11), true},
		{"progressive 4:2:0 6000x4000", "image/jpeg", imageTestJPEGHeader(0xC2, 6000, 4000, 0x22, 0x11, 0x11), false},
		{"progressive 4:4:4 4000x3000", "image/jpeg", imageTestJPEGHeader(0xC2, 4000, 3000, 0x11, 0x11, 0x11), false},
		{"progressive CMYK 3000x2000", "image/jpeg", imageTestJPEGHeader(0xC2, 3000, 2000, 0x11, 0x11, 0x11, 0x11), false},
		{"baseline CMYK 6000x4000", "image/jpeg", imageTestJPEGHeader(0xC0, 6000, 4000, 0x11, 0x11, 0x11, 0x11), false},
		{"16-bit PNG 6000x4000", "image/png", imageTestPNGHeaderDepth(6000, 4000, 16, 0), false},
		{"interlaced 16-bit PNG 6000x4000", "image/png", imageTestPNGHeaderDepth(6000, 4000, 16, 1), true},
		{"PNG 2x12000000", "image/png", imageTestPNGHeader(2, 12_000_000), true},
	} {
		_, err := decodeImageUpload(context.Background(), uuid.New(), imageTestDataURI(c.mime, c.raw))
		if refused := errors.Is(err, errImageTooCostly); refused != c.refused {
			t.Errorf("%s: err = %v, want refused as too costly: %v", c.name, err, c.refused)
		}
	}
}

// The budget holds two images at the cap at once, and a third waits; small
// images take a small share.
func TestAcquireDecodeBudget(t *testing.T) {
	imageTestPixelLimit(t, 0)
	first, err := acquireDecodeBudget(context.Background(), maxDecodeCost())
	if err != nil {
		t.Fatal(err)
	}
	defer first()
	second, err := acquireDecodeBudget(context.Background(), maxDecodeCost())
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := acquireDecodeBudget(ctx, decodeOverheadBytes); !errors.Is(err, errImageBusy) || !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("third acquire: err = %v, want errImageBusy from the deadline", err)
	}

	second()
	var releases []func()
	for range decodeSharesPerImage / 100 {
		release, err := acquireDecodeBudget(context.Background(), maxDecodeCost()/100)
		if err != nil {
			t.Fatalf("small image: %v", err)
		}
		releases = append(releases, release)
	}
	for _, release := range releases {
		release()
	}
}
