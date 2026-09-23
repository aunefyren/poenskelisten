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
