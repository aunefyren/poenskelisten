package utilities

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/draw"
)

// JPEGOrientation returns the EXIF Orientation tag (1-8) of a JPEG file, or 1
// (upright) when the file has no EXIF block or it can't be parsed. Phone
// cameras store pixels in sensor order and rely on this tag for display;
// Go's image/jpeg ignores it and our re-encode drops all metadata, so it has
// to be read from the original upload bytes and baked into the pixels.
func JPEGOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}

	pos := 2
	for pos+4 <= len(data) {
		if data[pos] != 0xFF {
			return 1
		}
		marker := data[pos+1]
		// Fill bytes: any number of 0xFF may precede a marker.
		if marker == 0xFF {
			pos++
			continue
		}
		// Start of scan / end of image: metadata segments always come earlier.
		if marker == 0xDA || marker == 0xD9 {
			return 1
		}
		segmentLength := int(binary.BigEndian.Uint16(data[pos+2 : pos+4]))
		if segmentLength < 2 || pos+2+segmentLength > len(data) {
			return 1
		}
		segment := data[pos+4 : pos+2+segmentLength]
		if marker == 0xE1 && bytes.HasPrefix(segment, []byte("Exif\x00\x00")) {
			return exifOrientation(segment[6:])
		}
		pos += 2 + segmentLength
	}

	return 1
}

// exifOrientation reads tag 0x0112 from IFD0 of a TIFF-structured EXIF payload.
func exifOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}

	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	if order.Uint16(tiff[2:4]) != 0x002A {
		return 1
	}

	ifdOffset := int(order.Uint32(tiff[4:8]))
	if ifdOffset < 8 || ifdOffset+2 > len(tiff) {
		return 1
	}
	entryCount := int(order.Uint16(tiff[ifdOffset : ifdOffset+2]))
	for i := 0; i < entryCount; i++ {
		entry := ifdOffset + 2 + i*12
		if entry+12 > len(tiff) {
			return 1
		}
		if order.Uint16(tiff[entry:entry+2]) != 0x0112 {
			continue
		}
		// Type 3 = SHORT; the value sits left-aligned in the 4-byte value field.
		if order.Uint16(tiff[entry+2:entry+4]) != 3 {
			return 1
		}
		orientation := int(order.Uint16(tiff[entry+8 : entry+10]))
		if orientation < 1 || orientation > 8 {
			return 1
		}
		return orientation
	}

	return 1
}

// ApplyOrientation returns img transformed so that an image carrying the given
// EXIF orientation displays upright without the tag.
func ApplyOrientation(img image.Image, orientation int) image.Image {
	if orientation < 2 || orientation > 8 {
		return img
	}

	bounds := img.Bounds()
	src := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(src, src.Bounds(), img, bounds.Min, draw.Src)

	w, h := src.Rect.Dx(), src.Rect.Dy()
	dstW, dstH := w, h
	// Orientations 5-8 swap the axes.
	if orientation >= 5 {
		dstW, dstH = h, w
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))

	for y := 0; y < dstH; y++ {
		for x := 0; x < dstW; x++ {
			var sx, sy int
			switch orientation {
			case 2: // mirror horizontal
				sx, sy = w-1-x, y
			case 3: // rotate 180
				sx, sy = w-1-x, h-1-y
			case 4: // mirror vertical
				sx, sy = x, h-1-y
			case 5: // transpose
				sx, sy = y, x
			case 6: // rotate 90 clockwise
				sx, sy = y, h-1-x
			case 7: // transverse
				sx, sy = w-1-y, h-1-x
			case 8: // rotate 90 counter-clockwise
				sx, sy = w-1-y, x
			}
			copy(dst.Pix[dst.PixOffset(x, y):dst.PixOffset(x, y)+4], src.Pix[src.PixOffset(sx, sy):src.PixOffset(sx, sy)+4])
		}
	}

	return dst
}
