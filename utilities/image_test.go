package utilities

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

// exifSegment builds a JPEG APP1 segment holding a minimal EXIF block whose
// IFD0 contains a single Orientation entry.
func exifSegment(order binary.ByteOrder, orientation uint16) []byte {
	tiff := new(bytes.Buffer)
	if order == binary.LittleEndian {
		tiff.WriteString("II")
	} else {
		tiff.WriteString("MM")
	}
	binary.Write(tiff, order, uint16(0x002A))
	binary.Write(tiff, order, uint32(8)) // IFD0 offset
	binary.Write(tiff, order, uint16(1)) // entry count
	binary.Write(tiff, order, uint16(0x0112))
	binary.Write(tiff, order, uint16(3)) // SHORT
	binary.Write(tiff, order, uint32(1))
	binary.Write(tiff, order, orientation)
	binary.Write(tiff, order, uint16(0)) // value field padding
	binary.Write(tiff, order, uint32(0)) // next IFD

	payload := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	segment := []byte{0xFF, 0xE1}
	segment = binary.BigEndian.AppendUint16(segment, uint16(len(payload)+2))
	return append(segment, payload...)
}

// withEXIF inserts segment directly after the SOI marker of a JPEG file.
func withEXIF(jpegBytes, segment []byte) []byte {
	out := append([]byte{}, jpegBytes[:2]...)
	out = append(out, segment...)
	return append(out, jpegBytes[2:]...)
}

func testJPEG(t *testing.T) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, image.NewRGBA(image.Rect(0, 0, 4, 2)), nil); err != nil {
		t.Fatalf("failed to encode fixture: %v", err)
	}
	return buf.Bytes()
}

func TestJPEGOrientation(t *testing.T) {
	plain := testJPEG(t)
	truncated := withEXIF(plain, exifSegment(binary.BigEndian, 6))[:20]

	cases := []struct {
		name string
		data []byte
		want int
	}{
		{"no exif", plain, 1},
		{"little endian 6", withEXIF(plain, exifSegment(binary.LittleEndian, 6)), 6},
		{"big endian 8", withEXIF(plain, exifSegment(binary.BigEndian, 8)), 8},
		{"big endian 3", withEXIF(plain, exifSegment(binary.BigEndian, 3)), 3},
		{"out of range value", withEXIF(plain, exifSegment(binary.BigEndian, 9)), 1},
		{"truncated segment", truncated, 1},
		{"not a jpeg", []byte("\x89PNG\r\n\x1a\n"), 1},
		{"empty", nil, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := JPEGOrientation(c.data); got != c.want {
				t.Errorf("JPEGOrientation() = %d, want %d", got, c.want)
			}
		})
	}
}

func TestApplyOrientation(t *testing.T) {
	// 3x2 source, each pixel's red channel encodes its position:
	//   0 1 2
	//   3 4 5
	src := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	for i := 0; i < 6; i++ {
		src.SetNRGBA(i%3, i/3, color.NRGBA{R: uint8(i), A: 255})
	}

	cases := []struct {
		orientation int
		want        [][]uint8 // rows of the expected output
	}{
		{1, [][]uint8{{0, 1, 2}, {3, 4, 5}}},
		{2, [][]uint8{{2, 1, 0}, {5, 4, 3}}},
		{3, [][]uint8{{5, 4, 3}, {2, 1, 0}}},
		{4, [][]uint8{{3, 4, 5}, {0, 1, 2}}},
		{5, [][]uint8{{0, 3}, {1, 4}, {2, 5}}},
		{6, [][]uint8{{3, 0}, {4, 1}, {5, 2}}},
		{7, [][]uint8{{5, 2}, {4, 1}, {3, 0}}},
		{8, [][]uint8{{2, 5}, {1, 4}, {0, 3}}},
	}
	for _, c := range cases {
		got := ApplyOrientation(src, c.orientation)
		bounds := got.Bounds()
		if bounds.Dy() != len(c.want) || bounds.Dx() != len(c.want[0]) {
			t.Errorf("orientation %d: size %dx%d, want %dx%d", c.orientation, bounds.Dx(), bounds.Dy(), len(c.want[0]), len(c.want))
			continue
		}
		for y, row := range c.want {
			for x, want := range row {
				r, _, _, _ := got.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
				if uint8(r>>8) != want {
					t.Errorf("orientation %d: pixel (%d,%d) = %d, want %d", c.orientation, x, y, r>>8, want)
				}
			}
		}
	}
}

// app1 wraps a raw TIFF payload in an EXIF APP1 segment and prefixes SOI.
func app1(tiff []byte) []byte {
	payload := append([]byte("Exif\x00\x00"), tiff...)
	out := []byte{0xFF, 0xD8, 0xFF, 0xE1}
	out = binary.BigEndian.AppendUint16(out, uint16(len(payload)+2))
	return append(out, payload...)
}

// tiffIFD builds a little-endian TIFF header plus an IFD0 holding the given
// (tag, type, value) entries.
func tiffIFD(entries ...[3]uint16) []byte {
	b := []byte("II")
	b = binary.LittleEndian.AppendUint16(b, 0x002A)
	b = binary.LittleEndian.AppendUint32(b, 8)
	b = binary.LittleEndian.AppendUint16(b, uint16(len(entries)))
	for _, e := range entries {
		b = binary.LittleEndian.AppendUint16(b, e[0])
		b = binary.LittleEndian.AppendUint16(b, e[1])
		b = binary.LittleEndian.AppendUint32(b, 1)
		b = binary.LittleEndian.AppendUint16(b, e[2])
		b = binary.LittleEndian.AppendUint16(b, 0)
	}
	return binary.LittleEndian.AppendUint32(b, 0)
}

// TestJPEGOrientationMalformed feeds hand-built marker streams and EXIF
// payloads through every fallback path: anything unparseable must read as
// upright (1) rather than guessing or panicking, while a valid tag behind
// padding or an unrelated tag must still be found.
func TestJPEGOrientationMalformed(t *testing.T) {
	validIFD := tiffIFD([3]uint16{0x0112, 3, 6})

	withMagic := append([]byte{}, validIFD...)
	binary.LittleEndian.PutUint16(withMagic[2:4], 0x002B)

	badOffset := append([]byte{}, validIFD...)
	binary.LittleEndian.PutUint32(badOffset[4:8], 4)

	farOffset := append([]byte{}, validIFD...)
	binary.LittleEndian.PutUint32(farOffset[4:8], 1000)

	// Entry count claims two entries but only an unrelated one is present.
	truncatedEntries := tiffIFD([3]uint16{0x010F, 2, 0})[:22]
	binary.LittleEndian.PutUint16(truncatedEntries[8:10], 2)

	bigEndianHeader := append([]byte("MM"), validIFD[2:]...) // magic now reads 0x2A00

	cases := []struct {
		name string
		data []byte
		want int
	}{
		{"fill bytes before exif marker", append([]byte{0xFF, 0xD8, 0xFF}, app1(validIFD)[2:]...), 6},
		{"unrelated tag before orientation", app1(tiffIFD([3]uint16{0x010F, 2, 0}, [3]uint16{0x0112, 3, 8})), 8},
		{"orientation not SHORT", app1(tiffIFD([3]uint16{0x0112, 4, 6})), 1},
		{"no orientation tag", app1(tiffIFD([3]uint16{0x010F, 2, 0})), 1},
		{"empty IFD", app1(tiffIFD()), 1},
		{"tiff shorter than header", app1([]byte("II*\x00")), 1},
		{"unknown byte order", app1(append([]byte("XX"), validIFD[2:]...)), 1},
		{"bad magic", app1(withMagic), 1},
		{"byte order mismatch breaks magic", app1(bigEndianHeader), 1},
		{"IFD offset inside header", app1(badOffset), 1},
		{"IFD offset past end", app1(farOffset), 1},
		{"entry truncated", app1(truncatedEntries), 1},
		{"garbage instead of marker", []byte{0xFF, 0xD8, 0x00, 0x11, 0x22, 0x33}, 1},
		{"segment length below 2", []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x01, 0x00, 0x00}, 1},
		{"end of image marker", []byte{0xFF, 0xD8, 0xFF, 0xD9, 0x00, 0x00}, 1},
		// APP0 segment consumes everything, leaving too few bytes for another marker.
		{"stream ends after segment", []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x04, 0x4A, 0x46, 0xFF}, 1},
		{"non-exif APP1", []byte{0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x06, 'h', 't', 't', 'p', 0xFF, 0xD9, 0x00, 0x00}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := JPEGOrientation(c.data); got != c.want {
				t.Errorf("JPEGOrientation() = %d, want %d", got, c.want)
			}
		})
	}
}

func TestApplyOrientationOutOfRangeReturnsInput(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	for _, o := range []int{0, 1, 9, -3} {
		if got := ApplyOrientation(src, o); got != image.Image(src) {
			t.Errorf("ApplyOrientation(%d) returned a new image, want the input unchanged", o)
		}
	}
}
