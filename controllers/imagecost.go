package controllers

import (
	"aunefyren/poenskelisten/config"
	"context"
	"errors"
	"fmt"
	"image"
	"math"
	"runtime"
	"time"

	"golang.org/x/sync/semaphore"
)

// Decoding an image takes memory in proportion to its pixel count, but how
// much per pixel depends on its format: a progressive JPEG keeps every
// coefficient of every component until its last scan, and the resize to the
// stored size needs scratch space in proportion to the source's height. So
// each decode is charged its estimated cost against a shared budget, rather
// than a fixed number of decodes being allowed at once.
const (
	// An image at the per-image cap (maxDecodeCost) takes this many shares of
	// the budget; cheaper images take proportionally fewer.
	decodeSharesPerImage = 1000
	// The budget holds this many images at the per-image cap.
	worstCaseConcurrentDecodes = 2
	// A decode taking at least this many shares collects garbage when done.
	decodeGCMinShares = decodeSharesPerImage / 4
	// How long an upload waits for budget before it is refused, so requests
	// (and the bodies they hold) can't pile up behind a busy decoder.
	decodeWaitTimeout = 30 * time.Second
	// golang.org/x/image/draw's kernel scaler allocates one [4]float64 per
	// destination column per source row.
	resizeScratchBytesPerEntry = 32
	// Covers what doesn't depend on the source size: the 1000x1000 scaled
	// copy and the orientation and flattening copies (4 MB each at most) and
	// the decoders' buffers.
	decodeOverheadBytes = 32 << 20
	// Upper bounds per pixel, for a header the estimate can't read.
	maxJPEGDecodeBytesPerPixel = 24
	maxPNGDecodeBytesPerPixel  = 16
	// What the per-image cap allows per pixel at the pixel limit: the most a
	// baseline JPEG or a non-interlaced PNG takes (a 16-bit PNG).
	referenceDecodeBytesPerPixel = 8
)

var decodeBudget = semaphore.NewWeighted(worstCaseConcurrentDecodes * decodeSharesPerImage)

var (
	errImageBusy             = errors.New("The server is busy processing other images. Try again in a moment.")
	errImageUploadInProgress = errors.New("Another image upload of yours is still being processed. Try again when it finishes.")
	errImageTooCostly        = errors.New("Image needs too much memory to process. Use a smaller image, or save it as a standard (non-progressive) JPEG or a non-interlaced PNG.")
)

// fittedSize is the size fitWithin scales a width x height image to.
func fittedSize(width, height, maxWidth, maxHeight int) (int, int) {
	if width <= maxWidth && height <= maxHeight {
		return width, height
	}
	scale := min(float64(maxWidth)/float64(width), float64(maxHeight)/float64(height))
	return max(1, int(float64(width)*scale+0.5)), max(1, int(float64(height)*scale+0.5))
}

// resizeScratchBytes is the scratch space fitWithin needs to scale a
// width x height image down to the stored size. It grows with the source's
// height, so a very tall, narrow image needs far more than its pixel count
// suggests.
func resizeScratchBytes(width, height int) int64 {
	if width <= maxImageWidth && height <= maxImageHeight {
		return 0
	}
	fittedWidth, _ := fittedSize(width, height, maxImageWidth, maxImageHeight)
	return int64(fittedWidth) * int64(height) * resizeScratchBytesPerEntry
}

// decodeCost estimates the most memory decoding an image ("jpeg" or "png")
// and scaling it down to the stored size takes, in bytes.
func decodeCost(imageBytes []byte, format string, header image.Config) int64 {
	var perPixel float64
	switch format {
	case "jpeg":
		perPixel = jpegDecodeBytesPerPixel(imageBytes)
	case "png":
		perPixel = pngDecodeBytesPerPixel(imageBytes)
	default:
		perPixel = maxJPEGDecodeBytesPerPixel
	}
	pixels := float64(header.Width) * float64(header.Height)
	return int64(math.Ceil(pixels*perPixel)) + resizeScratchBytes(header.Width, header.Height) + decodeOverheadBytes
}

// maxDecodeCost is the most memory one decode may take: what a 16-bit PNG at
// the pixel limit takes in its costliest (square) shape. The pixel limit is
// then the only limit for baseline JPEGs and non-interlaced PNGs, while
// progressive JPEGs and interlaced PNGs, which take more per pixel, are
// refused at fewer pixels.
func maxDecodeCost() int64 {
	pixels := config.MaxImagePixels()
	side := int(math.Sqrt(float64(pixels)))
	return pixels*referenceDecodeBytesPerPixel + resizeScratchBytes(side, side) + decodeOverheadBytes
}

// acquireDecodeBudget waits until cost fits in the decode budget, or until
// ctx ends or decodeWaitTimeout passes, which it reports as errImageBusy.
func acquireDecodeBudget(ctx context.Context, cost int64) (release func(), err error) {
	shares := min(decodeSharesPerImage, max(1, (cost*decodeSharesPerImage+maxDecodeCost()-1)/maxDecodeCost()))

	waitCtx, cancel := context.WithTimeout(ctx, decodeWaitTimeout)
	defer cancel()
	if err := decodeBudget.Acquire(waitCtx, shares); err != nil {
		return nil, fmt.Errorf("%w (%w)", errImageBusy, err)
	}
	return func() {
		// The full-size image is garbage by now, but the collector may not
		// reclaim it before the next decode allocates its own, which can let
		// the heap grow well past the budget. Collecting first keeps the heap
		// close to what the budget allows; a small decode isn't worth it.
		if shares >= decodeGCMinShares {
			runtime.GC()
		}
		decodeBudget.Release(shares)
	}, nil
}

// jpegDecodeBytesPerPixel mirrors what image/jpeg allocates: the decoded image
// (one byte per sample: gray, or YCbCr at its chroma subsampling plus a black
// channel for a 4-component image), a 4-byte-per-pixel RGBA or CMYK copy when
// the image is converted (a 4-component image always is, a 3-component one
// when it holds RGB rather than YCbCr), and for a progressive image 4 bytes of
// coefficients per sample, kept until the last scan.
func jpegDecodeBytesPerPixel(imageBytes []byte) float64 {
	header, ok := readJPEGHeader(imageBytes)
	if !ok || header.sampling[0][0]*header.sampling[0][1] == 0 {
		return maxJPEGDecodeBytesPerPixel
	}

	samples := 1.0 // image/jpeg ignores a grayscale image's nominal sampling.
	if len(header.sampling) > 1 {
		samples = 0
		for _, hv := range header.sampling {
			samples += float64(hv[0]*hv[1]) / float64(header.sampling[0][0]*header.sampling[0][1])
		}
	}
	perPixel := samples
	if len(header.sampling) == 4 || (len(header.sampling) == 3 && header.mayBeRGB()) {
		perPixel += 4
	}
	if header.progressive {
		perPixel += 4 * samples
	}
	return perPixel
}

// jpegHeader is what decides how much memory image/jpeg needs for an image.
type jpegHeader struct {
	progressive bool
	sampling    [][2]int // each component's (horizontal, vertical) factors
	ids         []byte   // each component's identifier
	// Read from the markers between the frame header and the first scan;
	// complete is false if the file ended before that scan.
	jfif, adobe    bool
	adobeTransform byte
	complete       bool
}

// mayBeRGB reports whether image/jpeg could treat a 3-component image as RGB
// (its isRGB), and so convert it into an RGBA copy.
func (h jpegHeader) mayBeRGB() bool {
	switch {
	case !h.complete:
		return true
	case h.jfif:
		return false
	case h.adobe && h.adobeTransform == 0:
		return true
	}
	return h.ids[0] == 'R' && h.ids[1] == 'G' && h.ids[2] == 'B'
}

// readJPEGHeader walks a JPEG's markers up to its first scan, the way
// image/jpeg does. ok is false for a file it can't follow, or without a frame
// header, which the caller charges the worst case.
func readJPEGHeader(b []byte) (header jpegHeader, ok bool) {
	if len(b) < 2 || b[0] != 0xFF || b[1] != 0xD8 {
		return jpegHeader{}, false
	}
	frameFound := false
	i := 2
	for {
		if i >= len(b) {
			return header, frameFound
		}
		if b[i] != 0xFF {
			return jpegHeader{}, false
		}
		// Any number of 0xFF fill bytes may precede a marker.
		for i < len(b) && b[i] == 0xFF {
			i++
		}
		if i >= len(b) {
			return header, frameFound
		}
		marker := b[i]
		i++
		switch {
		case marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7): // no length
			continue
		case marker == 0xDA: // the first scan: everything that matters is known
			header.complete = frameFound
			return header, frameFound
		case marker == 0xD8 || marker == 0xD9:
			return jpegHeader{}, false
		}

		if i+2 > len(b) {
			return header, frameFound
		}
		length := int(b[i])<<8 | int(b[i+1])
		if length < 2 {
			return jpegHeader{}, false
		}
		if i+length > len(b) {
			return header, frameFound
		}
		segment := b[i+2 : i+length]
		i += length

		switch marker {
		case 0xC0, 0xC1, 0xC2:
			if frameFound || len(segment) < 6 {
				return jpegHeader{}, false
			}
			components := int(segment[5])
			if components == 0 || len(segment) < 6+3*components {
				return jpegHeader{}, false
			}
			for c := range components {
				header.ids = append(header.ids, segment[6+3*c])
				hv := segment[7+3*c]
				header.sampling = append(header.sampling, [2]int{int(hv >> 4), int(hv & 0x0f)})
			}
			header.progressive = marker == 0xC2
			frameFound = true
		case 0xE0:
			if len(segment) >= 5 && string(segment[:5]) == "JFIF\x00" {
				header.jfif = true
			}
		case 0xEE:
			if len(segment) >= 12 && string(segment[:5]) == "Adobe" {
				header.adobe = true
				header.adobeTransform = segment[11]
			}
		}
	}
}

// pngDecodeBytesPerPixel mirrors image/png: 1 byte per pixel for a paletted
// image, 8 for 16-bit samples, 4 otherwise. A grayscale or RGB image counts
// as decoded with transparency (into NRGBA), which a tRNS chunk causes but
// DecodeConfig doesn't report. An interlaced image is decoded pass by pass
// into separate images before they are merged, so it counts twice.
func pngDecodeBytesPerPixel(imageBytes []byte) float64 {
	// The IHDR chunk comes first: after the 8-byte signature, its length and
	// type, then width and height, then bit depth, color type, compression,
	// filter and interlace method.
	if len(imageBytes) < 29 || string(imageBytes[12:16]) != "IHDR" {
		return maxPNGDecodeBytesPerPixel
	}
	depth, colorType, interlace := imageBytes[24], imageBytes[25], imageBytes[28]

	perPixel := 4.0
	switch {
	case colorType == 3:
		perPixel = 1
	case depth == 16:
		perPixel = 8
	}
	if interlace != 0 {
		perPixel *= 2
	}
	return perPixel
}
