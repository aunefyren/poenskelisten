package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/utilities"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/image/draw"
)

var profileImageDir, _ = filepath.Abs("./images/profiles")
var wishImageDir, _ = filepath.Abs("./images/wishes")
var defaultProfileImagePath, _ = filepath.Abs("./web/assets/user.svg")

const (
	maxImageWidth      = 1000
	maxImageHeight     = 1000
	maxThumbnailWidth  = 250
	maxThumbnailHeight = 250
	maxUploadBytes     = 10_000_000
	// Caps the decoded size of an upload. The byte limit alone doesn't bound
	// memory: a small, highly compressible PNG can declare enormous dimensions.
	// 50 MP covers any current phone camera's default output.
	maxUploadPixels = 50_000_000
	jpegQuality     = 85
)

// imageFilePath is the only place image paths are built. It takes a parsed
// UUID rather than the raw route param, so user input never reaches the
// filesystem and every textual form of a UUID maps to the same file.
func imageFilePath(dir string, id uuid.UUID, thumbnail bool) string {
	name := id.String()
	if thumbnail {
		name += "_thumbnail"
	}
	return filepath.Join(dir, name+".jpg")
}

func APIGetUserProfileImage(context *gin.Context) {
	thumbnail := context.Query("thumbnail") == "true"

	userID, err := uuid.Parse(context.Param("user_id"))
	if err != nil {
		logger.Log.Error("Failed to parse user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse user ID."})
		context.Abort()
		return
	}

	_, err = database.GetUserInformation(userID)
	if errors.Is(err, database.ErrUserNotFound) {
		context.JSON(http.StatusNotFound, gin.H{"error": "Failed to find user."})
		context.Abort()
		return
	} else if err != nil {
		logger.Log.Error("Failed to find user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find user."})
		context.Abort()
		return
	}

	imageBytes, found, err := loadStoredImage(profileImageDir, userID, thumbnail)
	if err != nil {
		logger.Log.Error("Failed to load profile image. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load profile image."})
		context.Abort()
		return
	} else if !found {
		imageBytes, err = LoadDefaultProfileImage()
		if err != nil {
			logger.Log.Error("Failed to load default profile image. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load default profile image."})
			context.Abort()
			return
		}
	}

	if notModified(context, imageBytes) {
		return
	}

	context.JSON(http.StatusOK, gin.H{"image": ImageBytesToBase64(imageBytes), "default": !found, "message": "Picture retrieved."})
}

func APIGetWishImage(context *gin.Context) {
	thumbnail := context.Query("thumbnail") == "true"

	wishID, err := uuid.Parse(context.Param("wish_id"))
	if err != nil {
		logger.Log.Error("Failed to parse wish ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wish ID."})
		context.Abort()
		return
	}

	wishlistFound, wishlist, err := database.GetWishlistByWishID(wishID)
	if err != nil {
		logger.Log.Error("Failed to get wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist."})
		context.Abort()
		return
	} else if !wishlistFound {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find wishlist for wish."})
		context.Abort()
		return
	}

	// A nil Public is treated as private, so the check fails closed.
	if wishlist.Public == nil || !*wishlist.Public {
		success, errorString, httpStatus := middlewares.AuthFunction(context, false)
		if !success {
			context.JSON(httpStatus, gin.H{"error": errorString})
			context.Abort()
			return
		}

		userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
		if err != nil {
			logger.Log.Error("Failed to parse header. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse header."})
			context.Abort()
			return
		}

		canView, err := userCanViewWishlist(userID, wishlist.ID)
		if err != nil {
			logger.Log.Error("Failed to verify wishlist access. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist access."})
			context.Abort()
			return
		} else if !canView {
			context.JSON(http.StatusForbidden, gin.H{"error": "You do not have access to this wishlist."})
			context.Abort()
			return
		}
	}

	imageBytes, found, err := loadStoredImage(wishImageDir, wishID, thumbnail)
	if err != nil {
		logger.Log.Error("Failed to load wish image. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load wish image."})
		context.Abort()
		return
	} else if !found {
		logger.Log.Warn("Failed to find wish image. Loading default.")
		imageBytes, err = LoadDefaultProfileImage()
		if err != nil {
			logger.Log.Error("Failed to load default profile image. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load default profile image."})
			context.Abort()
			return
		}
	}

	if notModified(context, imageBytes) {
		return
	}

	context.JSON(http.StatusOK, gin.H{"image": ImageBytesToBase64(imageBytes), "message": "Picture retrieved."})
}

// userCanViewWishlist mirrors who can list a wishlist's wishes: its owner,
// members of a group it is shared with, and its collaborators.
func userCanViewWishlist(userID uuid.UUID, wishlistID uuid.UUID) (bool, error) {
	checks := []func() (bool, error){
		func() (bool, error) { return database.VerifyUserOwnershipToWishlist(userID, wishlistID) },
		func() (bool, error) {
			return database.VerifyUserMembershipToGroupMembershipToWishlist(userID, wishlistID)
		},
		func() (bool, error) { return database.VerifyWishlistCollaboratorToWishlist(wishlistID, userID) },
	}
	for _, check := range checks {
		allowed, err := check()
		if err != nil {
			return false, err
		} else if allowed {
			return true, nil
		}
	}
	return false, nil
}

// notModified sets caching headers for an image response and, if the client's
// cached copy is current, answers 304 and returns true. Images are served as
// JSON behind a bearer token, so the cache is private and must revalidate.
func notModified(context *gin.Context, imageBytes []byte) bool {
	sum := sha256.Sum256(imageBytes)
	etag := `"` + hex.EncodeToString(sum[:16]) + `"`
	context.Header("ETag", etag)
	context.Header("Cache-Control", "private, no-cache")

	if context.GetHeader("If-None-Match") == etag {
		context.AbortWithStatus(http.StatusNotModified)
		return true
	}
	return false
}

// loadStoredImage returns the stored JPEG for id at the requested size, with
// found=false if no image has been uploaded. Uploads are resized when stored,
// but images stored by older versions are full-resolution originals without
// a thumbnail; those are resized here once and written back, so each legacy
// file pays the cost a single time.
func loadStoredImage(dir string, id uuid.UUID, thumbnail bool) ([]byte, bool, error) {
	fullPath := imageFilePath(dir, id, false)

	if thumbnail {
		thumbnailBytes, err := os.ReadFile(imageFilePath(dir, id, true))
		if err == nil {
			return thumbnailBytes, true, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, false, err
		}
	}

	fullBytes, err := os.ReadFile(fullPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}

	config, _, err := image.DecodeConfig(bytes.NewReader(fullBytes))
	if err != nil {
		return nil, false, errors.New("Stored image is not decodable: " + err.Error())
	}
	oversized := config.Width > maxImageWidth || config.Height > maxImageHeight
	if !thumbnail && !oversized {
		return fullBytes, true, nil
	}

	original, _, err := image.Decode(bytes.NewReader(fullBytes))
	if err != nil {
		return nil, false, errors.New("Stored image is not decodable: " + err.Error())
	}

	// Write-back failures are only logged: the image is still served, and the
	// resize is simply redone on the next request.
	if oversized {
		fullBytes, err = encodeJPEG(fitWithin(original, maxImageWidth, maxImageHeight))
		if err != nil {
			return nil, false, err
		}
		if err := writeFileAtomic(fullPath, fullBytes); err != nil {
			logger.Log.Warn("Failed to write back resized legacy image. Error: " + err.Error())
		}
	}
	if !thumbnail {
		return fullBytes, true, nil
	}

	thumbnailBytes, err := encodeJPEG(fitWithin(original, maxThumbnailWidth, maxThumbnailHeight))
	if err != nil {
		return nil, false, err
	}
	if err := writeFileAtomic(imageFilePath(dir, id, true), thumbnailBytes); err != nil {
		logger.Log.Warn("Failed to write back legacy image thumbnail. Error: " + err.Error())
	}
	return thumbnailBytes, true, nil
}

func CheckIfWishImageExists(wishID uuid.UUID) (bool, error) {
	_, err := os.Stat(imageFilePath(wishImageDir, wishID, false))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func ImageBytesToBase64(image []byte) string {
	var base64Encoding string

	switch http.DetectContentType(image) {
	case "image/jpeg":
		base64Encoding = "data:image/jpeg;base64,"
	case "image/png":
		base64Encoding = "data:image/png;base64,"
	default:
		base64Encoding = "data:image/svg+xml;base64,"
	}

	return base64Encoding + base64.StdEncoding.EncodeToString(image)
}

func Base64ToImageBytes(base64String string) ([]byte, string, error) {
	b64DataArray := strings.Split(base64String, "base64,")
	if len(b64DataArray) != 2 {
		return nil, "", errors.New("Base64 string does not contain mime type.")
	}

	mimeType := strings.Replace(b64DataArray[0], "data:", "", -1)
	mimeType = strings.Replace(mimeType, ";", "", -1)

	imageBytes, err := base64.StdEncoding.DecodeString(b64DataArray[1])
	if err != nil {
		logger.Log.Error("Failed to convert Base64 string to byte array. Returning. Error: " + err.Error())
		return nil, "", errors.New("Invalid Base64 string.")
	}

	return imageBytes, mimeType, nil
}

func LoadDefaultProfileImage() ([]byte, error) {
	imageBytes, err := os.ReadFile(defaultProfileImagePath)
	if err != nil {
		logger.Log.Error("Failed to load default profile image. Error: " + err.Error() + ". Returning.")
		return nil, errors.New("Failed to load default profile image.")
	}

	return imageBytes, nil
}

// fitWithin scales img down to fit inside maxWidth x maxHeight, preserving
// aspect ratio. Images that already fit are returned unchanged.
func fitWithin(img image.Image, maxWidth int, maxHeight int) image.Image {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= maxWidth && height <= maxHeight {
		return img
	}

	scale := min(float64(maxWidth)/float64(width), float64(maxHeight)/float64(height))
	newWidth := max(1, int(float64(width)*scale+0.5))
	newHeight := max(1, int(float64(height)*scale+0.5))

	resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.CatmullRom.Scale(resized, resized.Bounds(), img, bounds, draw.Src, nil)
	return resized
}

// flattenOnWhite composites transparent images onto white. JPEG has no alpha
// channel, and encoding a transparent PNG as-is turns its transparent areas black.
func flattenOnWhite(img image.Image) image.Image {
	if opaque, ok := img.(interface{ Opaque() bool }); ok && opaque.Opaque() {
		return img
	}

	bounds := img.Bounds()
	flattened := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(flattened, flattened.Bounds(), image.White, image.Point{}, draw.Src)
	draw.Draw(flattened, flattened.Bounds(), img, bounds.Min, draw.Over)
	return flattened
}

// decodeUploadedImage decodes an uploaded JPEG/PNG and bakes any EXIF
// orientation into the pixels. The re-encode in encodeJPEG writes no
// metadata, which is what keeps EXIF (GPS, camera serial, timestamps) out of
// stored and served images - but it also drops the orientation tag, so it has
// to be applied here or phone photos end up sideways.
func decodeUploadedImage(imageBytes []byte, mimeType string) (image.Image, error) {
	var decodeConfig func([]byte) (image.Config, error)
	var decode func([]byte) (image.Image, error)

	switch mimeType {
	case "image/jpeg":
		decodeConfig = func(b []byte) (image.Config, error) { return jpeg.DecodeConfig(bytes.NewReader(b)) }
		decode = func(b []byte) (image.Image, error) { return jpeg.Decode(bytes.NewReader(b)) }
	case "image/png":
		decodeConfig = func(b []byte) (image.Config, error) { return png.DecodeConfig(bytes.NewReader(b)) }
		decode = func(b []byte) (image.Image, error) { return png.Decode(bytes.NewReader(b)) }
	default:
		logger.Log.Error("Invalid mime type for image. Type: " + mimeType)
		return nil, errors.New("Invalid image type.")
	}

	config, err := decodeConfig(imageBytes)
	if err != nil {
		logger.Log.Error("Failed to read image header. Returning. Error: " + err.Error())
		return nil, errors.New("Failed to create image from byte array.")
	}
	if int64(config.Width)*int64(config.Height) > maxUploadPixels {
		return nil, errors.New("Image dimensions are too large.")
	}

	imageObject, err := decode(imageBytes)
	if err != nil {
		logger.Log.Error("Failed to create image from byte array. Returning. Error: " + err.Error())
		return nil, errors.New("Failed to create image from byte array.")
	}

	if mimeType == "image/jpeg" {
		imageObject = utilities.ApplyOrientation(imageObject, utilities.JPEGOrientation(imageBytes))
	}

	return flattenOnWhite(imageObject), nil
}

// storeUploadedImage validates a base64 data-URI upload and stores it as a
// full-size and a thumbnail JPEG, so serving never has to resize.
func storeUploadedImage(dir string, id uuid.UUID, base64String string) error {
	imageBytes, mimeType, err := Base64ToImageBytes(base64String)
	if err != nil {
		logger.Log.Error("Failed to convert Base64 String to bytes. Error: " + err.Error())
		return errors.New("Invalid Base64 string.")
	}

	if len(imageBytes) > maxUploadBytes {
		return errors.New("Image is too large.")
	}

	imageObject, err := decodeUploadedImage(imageBytes, mimeType)
	if err != nil {
		return err
	}

	err = writeImageVariants(dir, id, imageObject)
	if err != nil {
		logger.Log.Error("Failed to save image to disk. Returning. Error: " + err.Error())
		return errors.New("Failed to save image to disk.")
	}

	return nil
}

func writeImageVariants(dir string, id uuid.UUID, img image.Image) error {
	fullBytes, err := encodeJPEG(fitWithin(img, maxImageWidth, maxImageHeight))
	if err != nil {
		return err
	}
	thumbnailBytes, err := encodeJPEG(fitWithin(img, maxThumbnailWidth, maxThumbnailHeight))
	if err != nil {
		return err
	}
	if err := writeFileAtomic(imageFilePath(dir, id, false), fullBytes); err != nil {
		return err
	}
	return writeFileAtomic(imageFilePath(dir, id, true), thumbnailBytes)
}

func encodeJPEG(img image.Image) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, errors.New("Failed to encode image: " + err.Error())
	}
	return buf.Bytes(), nil
}

// writeFileAtomic writes to a temp file and renames it into place, so a failed
// write can't leave a truncated image and readers never see a partial one.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.New("Failed to create directory for image: " + err.Error())
	}

	tempFile, err := os.CreateTemp(dir, ".upload-*.jpg")
	if err != nil {
		return errors.New("Failed to create file for image: " + err.Error())
	}
	// Harmless after a successful rename; cleans up on every failure path.
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return errors.New("Failed to write image: " + err.Error())
	}
	if err := tempFile.Close(); err != nil {
		return errors.New("Failed to write image: " + err.Error())
	}
	// CreateTemp uses 0600; keep the permissions os.Create used to give.
	if err := os.Chmod(tempFile.Name(), 0644); err != nil {
		return errors.New("Failed to set image permissions: " + err.Error())
	}
	if err := os.Rename(tempFile.Name(), path); err != nil {
		return errors.New("Failed to move image into place: " + err.Error())
	}

	return nil
}

// deleteImageFiles removes both stored sizes of an image. Missing files are
// not an error, so it is safe to call for records that never had an image.
func deleteImageFiles(dir string, id uuid.UUID) error {
	for _, thumbnail := range []bool{false, true} {
		err := os.Remove(imageFilePath(dir, id, thumbnail))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}

func UpdateUserProfileImage(userID uuid.UUID, base64String string) error {
	return storeUploadedImage(profileImageDir, userID, base64String)
}

func DeleteUserProfileImage(userID uuid.UUID) error {
	err := deleteImageFiles(profileImageDir, userID)
	if err != nil {
		logger.Log.Error("Failed to delete profile image. Error: " + err.Error())
		return errors.New("Failed to delete profile image.")
	}
	return nil
}

func SaveWishImage(wishID uuid.UUID, base64String string) error {
	return storeUploadedImage(wishImageDir, wishID, base64String)
}

func DeleteWishImage(wishID uuid.UUID) error {
	err := deleteImageFiles(wishImageDir, wishID)
	if err != nil {
		logger.Log.Error("Failed to delete requested wish image. Error: " + err.Error())
		return errors.New("Failed to delete requested wish image.")
	}
	return nil
}
