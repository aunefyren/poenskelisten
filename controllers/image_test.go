package controllers

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"image"
	"image/jpeg"
	"image/png"
	"io/fs"
	"math/rand"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logrusTest "github.com/sirupsen/logrus/hooks/test"
)

// imageTestSetup points the package-level image path vars at a temp directory
// and a real default avatar (the ones baked in at package init resolve relative
// to the package's source dir when running `go test`, which doesn't contain a
// web/assets tree), restoring the originals afterwards. Safe to call from
// multiple tests since each gets its own temp dir.
func imageTestSetup(t *testing.T) {
	t.Helper()

	origProfile, origWish, origDefault := profileImageDir, wishImageDir, defaultProfileImagePath
	t.Cleanup(func() {
		profileImageDir, wishImageDir, defaultProfileImagePath = origProfile, origWish, origDefault
	})

	dir := t.TempDir()
	profileImageDir = filepath.Join(dir, "profiles")
	wishImageDir = filepath.Join(dir, "wishes")

	repoDefault, err := filepath.Abs("../web/assets/user.svg")
	if err != nil {
		t.Fatalf("failed to resolve default avatar path: %v", err)
	}
	if _, err := os.Stat(repoDefault); err != nil {
		t.Fatalf("default avatar fixture missing at %s: %v", repoDefault, err)
	}
	defaultProfileImagePath = repoDefault
}

// imageTestRandomImageBytes builds a width x height image filled with random
// pixels (so JPEG/PNG compression can't shrink it well below its raw size) and
// encodes it, returning the raw (non-base64) encoded file bytes. Business logic
// in image.go rejects encoded images under 10000 bytes; callers that exercise
// that success path (e.g. storeUploadedImage) need to pick a width/height
// large enough to clear it - 200x200 comfortably does.
func imageTestRandomImageBytes(t *testing.T, width, height int, asPNG bool) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	r := rand.New(rand.NewSource(42))
	// Random noise per pixel, so JPEG/PNG compression can't shrink the output
	// well below its raw size (a solid-color image would compress to a few
	// hundred bytes and fail the >=10000-byte fixture requirement above).
	for i := range img.Pix {
		img.Pix[i] = byte(r.Intn(256))
	}
	// Alpha channel must stay opaque or encoders may special-case it.
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			o := img.PixOffset(x, y)
			img.Pix[o+3] = 255
		}
	}

	buf := new(bytes.Buffer)
	var err error
	if asPNG {
		err = png.Encode(buf, img)
	} else {
		err = jpeg.Encode(buf, img, &jpeg.Options{Quality: 90})
	}
	if err != nil {
		t.Fatalf("failed to encode test image: %v", err)
	}
	return buf.Bytes()
}

func imageTestDataURI(mime string, raw []byte) string {
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(raw)
}

// imageTestPrivateWishlist inserts a wishlist with Public explicitly false.
func imageTestPrivateWishlist(t *testing.T, ownerID uuid.UUID) models.Wishlist {
	t.Helper()
	now := time.Now()
	wishlist := models.Wishlist{
		Name:    "Private Wishlist " + uuid.NewString(),
		Enabled: true,
		OwnerID: ownerID,
		Date:    &now,
		Public:  boolPtr(false),
	}
	wishlist.ID = uuid.New()
	created, err := database.CreateWishlistInDB(wishlist)
	if err != nil {
		t.Fatalf("failed to create private wishlist: %v", err)
	}
	return created
}

// imageTestPublicWishlist inserts a wishlist with Public explicitly true. The
// `public` column defaults to false at the DB level, so a wishlist created via
// the shared createTestWishlist fixture (which leaves Public nil in Go) comes
// back from the DB as Public=false, i.e. private - callers that specifically
// need a publicly-viewable wishlist must use this fixture instead.
func imageTestPublicWishlist(t *testing.T, ownerID uuid.UUID) models.Wishlist {
	t.Helper()
	now := time.Now()
	wishlist := models.Wishlist{
		Name:    "Public Wishlist " + uuid.NewString(),
		Enabled: true,
		OwnerID: ownerID,
		Date:    &now,
		Public:  boolPtr(true),
	}
	wishlist.ID = uuid.New()
	created, err := database.CreateWishlistInDB(wishlist)
	if err != nil {
		t.Fatalf("failed to create public wishlist: %v", err)
	}
	return created
}

func getWithParams(handler gin.HandlerFunc, path string, params gin.Params, header map[string]string) (int, map[string]interface{}) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", path, nil)
	for k, v := range header {
		ctx.Request.Header.Set(k, v)
	}
	ctx.Params = params
	handler(ctx)

	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w.Code, body
}

// --- APIGetUserProfileImage ---

func TestGetUserProfileImageInvalidID(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)

	code, body := getWithParams(APIGetUserProfileImage, "/api/users/x/image", gin.Params{{Key: "user_id", Value: "not-a-uuid"}}, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, body)
	}
}

func TestGetUserProfileImageUnknownUser(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)

	unknown := uuid.New()
	code, _ := getWithParams(APIGetUserProfileImage, "/api/users/x/image", gin.Params{{Key: "user_id", Value: unknown.String()}}, nil)
	if code != 404 {
		t.Fatalf("status = %d, want 404 for a user that doesn't exist", code)
	}
}

func TestGetUserProfileImageDefaultWhenNoFileSaved(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	user := createTestUser(t)

	code, body := getWithParams(APIGetUserProfileImage, "/api/users/x/image", gin.Params{{Key: "user_id", Value: user.ID.String()}}, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, body)
	}
	if body["default"] != true {
		t.Errorf("default = %v, want true when no image has been uploaded", body["default"])
	}
	img, _ := body["image"].(string)
	if img == "" {
		t.Error("expected a non-empty base64 image even for the default avatar")
	}
}

func TestGetUserProfileImageSavedFile(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	user := createTestUser(t)

	raw := imageTestRandomImageBytes(t, 200, 200, false)
	if err := storeUploadedImage(profileImageDir, user.ID, imageTestDataURI("image/jpeg", raw)); err != nil {
		t.Fatalf("failed to seed profile image: %v", err)
	}

	code, body := getWithParams(APIGetUserProfileImage, "/api/users/x/image?thumbnail=true", gin.Params{{Key: "user_id", Value: user.ID.String()}}, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, body)
	}
	if body["default"] != false {
		t.Errorf("default = %v, want false once an image has been uploaded", body["default"])
	}
}

// --- APIGetWishImage ---

func TestGetWishImageInvalidID(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)

	code, _ := getWithParams(APIGetWishImage, "/api/wishes/x/image", gin.Params{{Key: "wish_id", Value: "not-a-uuid"}}, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400", code)
	}
}

func TestGetWishImageNoWishlistForWish(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)

	unknown := uuid.New()
	code, body := getWithParams(APIGetWishImage, "/api/wishes/x/image", gin.Params{{Key: "wish_id", Value: unknown.String()}}, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, body)
	}
}

func TestGetWishImagePublicWishlistNoAuthNeeded(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	owner := createTestUser(t)
	wishlist := imageTestPublicWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	code, body := getWithParams(APIGetWishImage, "/api/wishes/x/image", gin.Params{{Key: "wish_id", Value: wish.ID.String()}}, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, body)
	}
	if body["image"] == "" || body["image"] == nil {
		t.Error("expected a base64 image (falls back to the default when none was uploaded)")
	}
}

func TestGetWishImagePrivateWishlistRequiresAuth(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	owner := createTestUser(t)
	wishlist := imageTestPrivateWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	code, body := getWithParams(APIGetWishImage, "/api/wishes/x/image", gin.Params{{Key: "wish_id", Value: wish.ID.String()}}, nil)
	if code != 401 {
		t.Fatalf("status = %d, want 401 without an Authorization header; body=%v", code, body)
	}
}

func TestGetWishImagePrivateWishlistWithAuth(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	owner := createTestUser(t)
	wishlist := imageTestPrivateWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	header := map[string]string{"Authorization": authHeader(t, owner.ID, false)}
	code, body := getWithParams(APIGetWishImage, "/api/wishes/x/image", gin.Params{{Key: "wish_id", Value: wish.ID.String()}}, header)
	if code != 200 {
		t.Fatalf("status = %d, want 200 with a valid token; body=%v", code, body)
	}
}

func TestGetWishImageSavedFile(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	owner := createTestUser(t)
	wishlist := imageTestPublicWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	raw := imageTestRandomImageBytes(t, 200, 200, true)
	if err := storeUploadedImage(wishImageDir, wish.ID, imageTestDataURI("image/png", raw)); err != nil {
		t.Fatalf("failed to seed wish image: %v", err)
	}

	code, body := getWithParams(APIGetWishImage, "/api/wishes/x/image", gin.Params{{Key: "wish_id", Value: wish.ID.String()}}, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, body)
	}
}

// --- pure helpers: encode/decode/resize/save/delete ---

func TestImageBytesToBase64MimeTypes(t *testing.T) {
	jpegBytes := imageTestRandomImageBytes(t, 50, 50, false)
	out := ImageBytesToBase64(jpegBytes)
	if !strings.HasPrefix(out, "data:image/jpeg;base64,") {
		t.Errorf("ImageBytesToBase64(jpeg) = %q", out[:min(30, len(out))])
	}

	pngBytes := imageTestRandomImageBytes(t, 50, 50, true)
	out = ImageBytesToBase64(pngBytes)
	if !strings.HasPrefix(out, "data:image/png;base64,") {
		t.Errorf("ImageBytesToBase64(png) = %q", out[:min(30, len(out))])
	}

	out = ImageBytesToBase64([]byte("not an image, just plain bytes"))
	if !strings.HasPrefix(out, "data:image/svg+xml;base64,") {
		t.Errorf("ImageBytesToBase64(unknown) should default to svg+xml, got %q", out[:min(40, len(out))])
	}
}

func TestBase64ToImageBytesRoundTrip(t *testing.T) {
	raw := imageTestRandomImageBytes(t, 50, 50, false)
	uri := imageTestDataURI("image/jpeg", raw)

	decoded, mime, err := Base64ToImageBytes(uri)
	if err != nil {
		t.Fatalf("Base64ToImageBytes failed: %v", err)
	}
	if mime != "image/jpeg" {
		t.Errorf("mime = %q, want image/jpeg", mime)
	}
	if !bytes.Equal(decoded, raw) {
		t.Error("decoded bytes do not match the original image bytes")
	}
}

func TestBase64ToImageBytesMissingMimeType(t *testing.T) {
	_, _, err := Base64ToImageBytes(base64.StdEncoding.EncodeToString([]byte("no-mime-prefix")))
	if err == nil {
		t.Error("expected an error when the string has no 'base64,' mime prefix")
	}
}

func TestBase64ToImageBytesInvalidBase64(t *testing.T) {
	_, _, err := Base64ToImageBytes("data:image/jpeg;base64,not-valid-base64!!!")
	if err == nil {
		t.Error("expected an error for invalid base64 content")
	}
}

func TestFitWithin(t *testing.T) {
	cases := []struct {
		name                  string
		width, height         int
		wantWidth, wantHeight int
	}{
		{"landscape", 2000, 1000, 1000, 500},
		{"portrait", 600, 3000, 200, 1000},
		{"already fits", 800, 600, 800, 600},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := fitWithin(image.NewRGBA(image.Rect(0, 0, c.width, c.height)), 1000, 1000).Bounds()
			if got.Dx() != c.wantWidth || got.Dy() != c.wantHeight {
				t.Errorf("fitWithin() = %dx%d, want %dx%d", got.Dx(), got.Dy(), c.wantWidth, c.wantHeight)
			}
		})
	}
}

func TestUpdateUserProfileImageSmallImageAccepted(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	user := createTestUser(t)

	raw := imageTestRandomImageBytes(t, 16, 16, true)
	if err := storeUploadedImage(profileImageDir, user.ID, imageTestDataURI("image/png", raw)); err != nil {
		t.Errorf("expected a small but valid image to be accepted, got: %v", err)
	}
}

func TestUpdateUserProfileImageGarbageRejected(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	user := createTestUser(t)

	if err := storeUploadedImage(profileImageDir, user.ID, imageTestDataURI("image/jpeg", []byte("short"))); err == nil {
		t.Error("expected an error for bytes that aren't a decodable image")
	}
}

func TestUpdateUserProfileImageInvalidMimeType(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	user := createTestUser(t)

	raw := imageTestRandomImageBytes(t, 200, 200, false)
	uri := imageTestDataURI("image/gif", raw)
	if err := storeUploadedImage(profileImageDir, user.ID, uri); err == nil {
		t.Error("expected an error for an unsupported mime type")
	}
}

func TestUpdateUserProfileImagePNGSuccess(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	user := createTestUser(t)

	raw := imageTestRandomImageBytes(t, 200, 200, true)
	if err := storeUploadedImage(profileImageDir, user.ID, imageTestDataURI("image/png", raw)); err != nil {
		t.Fatalf("storeUploadedImage failed for a valid png: %v", err)
	}
	if _, err := os.Stat(filepath.Join(profileImageDir, user.ID.String()+".jpg")); err != nil {
		t.Errorf("expected the profile image file to be saved to disk: %v", err)
	}
}

func TestCheckIfWishImageExists(t *testing.T) {
	imageTestSetup(t)
	wishID := uuid.New()

	exists, err := CheckIfWishImageExists(wishID)
	if err != nil || exists {
		t.Fatalf("exists = %v err = %v, want false/nil before any image is saved", exists, err)
	}

	raw := imageTestRandomImageBytes(t, 200, 200, false)
	if err := storeUploadedImage(wishImageDir, wishID, imageTestDataURI("image/jpeg", raw)); err != nil {
		t.Fatalf("failed to save wish image: %v", err)
	}

	exists, err = CheckIfWishImageExists(wishID)
	if err != nil || !exists {
		t.Fatalf("exists = %v err = %v, want true after saving", exists, err)
	}
}

func TestDeleteWishImage(t *testing.T) {
	imageTestSetup(t)
	wishID := uuid.New()

	// Deleting a non-existent image is a no-op, not an error.
	if err := DeleteWishImage(wishID); err != nil {
		t.Errorf("DeleteWishImage on a missing file returned an error: %v", err)
	}

	raw := imageTestRandomImageBytes(t, 200, 200, false)
	if err := storeUploadedImage(wishImageDir, wishID, imageTestDataURI("image/jpeg", raw)); err != nil {
		t.Fatalf("failed to save wish image: %v", err)
	}
	if err := DeleteWishImage(wishID); err != nil {
		t.Fatalf("DeleteWishImage failed: %v", err)
	}
	exists, err := CheckIfWishImageExists(wishID)
	if err != nil || exists {
		t.Errorf("exists = %v err = %v, want false after deletion", exists, err)
	}
}

func TestUpdateUserProfileImageTooLarge(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	user := createTestUser(t)

	huge := make([]byte, 10_000_001)
	uri := imageTestDataURI("image/jpeg", huge)
	if err := storeUploadedImage(profileImageDir, user.ID, uri); err == nil {
		t.Error("expected an error for an image over the maximum size")
	}
}

func TestSaveWishImageTooLarge(t *testing.T) {
	imageTestSetup(t)
	huge := make([]byte, 10_000_001)
	uri := imageTestDataURI("image/jpeg", huge)
	if err := storeUploadedImage(wishImageDir, uuid.New(), uri); err == nil {
		t.Error("expected an error for an image over the maximum size")
	}
}

func TestLoadDefaultProfileImageMissingFile(t *testing.T) {
	imageTestSetup(t)
	defaultProfileImagePath = filepath.Join(t.TempDir(), "does-not-exist.svg")

	if _, err := LoadDefaultProfileImage(); err == nil {
		t.Error("expected an error when the default avatar file is missing")
	}
}

func TestWriteFileAtomicDirectoryCollision(t *testing.T) {
	dir := t.TempDir()
	// Create a plain file where writeFileAtomic wants to MkdirAll a directory,
	// so the MkdirAll call fails.
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatalf("failed to seed blocking file: %v", err)
	}

	if err := writeFileAtomic(filepath.Join(blocker, "pic.jpg"), []byte("x")); err == nil {
		t.Error("expected an error when the target path collides with an existing file")
	}
}

func TestGetUserProfileImageResizeFailure(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	user := createTestUser(t)

	if err := os.MkdirAll(profileImageDir, 0755); err != nil {
		t.Fatalf("failed to create profile image dir: %v", err)
	}
	corrupt := filepath.Join(profileImageDir, user.ID.String()+".jpg")
	if err := os.WriteFile(corrupt, []byte("not a real jpeg"), 0644); err != nil {
		t.Fatalf("failed to seed corrupt image: %v", err)
	}

	code, body := getWithParams(APIGetUserProfileImage, "/api/users/x/image", gin.Params{{Key: "user_id", Value: user.ID.String()}}, nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500 when the saved file isn't a decodable image; body=%v", code, body)
	}
}

func TestGetWishImageResizeFailure(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	owner := createTestUser(t)
	wishlist := imageTestPublicWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	if err := os.MkdirAll(wishImageDir, 0755); err != nil {
		t.Fatalf("failed to create wish image dir: %v", err)
	}
	corrupt := filepath.Join(wishImageDir, wish.ID.String()+".jpg")
	if err := os.WriteFile(corrupt, []byte("not a real jpeg"), 0644); err != nil {
		t.Fatalf("failed to seed corrupt image: %v", err)
	}

	code, body := getWithParams(APIGetWishImage, "/api/wishes/x/image", gin.Params{{Key: "wish_id", Value: wish.ID.String()}}, nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500 when the saved file isn't a decodable image; body=%v", code, body)
	}
}

func TestDeleteWishImageRemoveFailure(t *testing.T) {
	imageTestSetup(t)
	wishID := uuid.New()

	raw := imageTestRandomImageBytes(t, 200, 200, false)
	if err := storeUploadedImage(wishImageDir, wishID, imageTestDataURI("image/jpeg", raw)); err != nil {
		t.Fatalf("failed to save wish image: %v", err)
	}

	// Strip write permission from the containing directory so the file is
	// still readable (CheckIfWishImageExists succeeds) but os.Remove fails.
	if err := os.Chmod(wishImageDir, 0555); err != nil {
		t.Fatalf("failed to chmod wish image dir: %v", err)
	}
	t.Cleanup(func() { os.Chmod(wishImageDir, 0755) })

	if err := DeleteWishImage(wishID); err == nil {
		t.Error("expected an error when the image file can't be removed")
	}
}

func TestWriteFileAtomicCreateFailure(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "pics")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}
	if err := os.Chmod(target, 0555); err != nil {
		t.Fatalf("failed to chmod target dir: %v", err)
	}
	t.Cleanup(func() { os.Chmod(target, 0755) })

	if err := writeFileAtomic(filepath.Join(target, "pic.jpg"), []byte("x")); err == nil {
		t.Error("expected an error when the target directory isn't writable")
	}
}

func TestImageHandlersDatabaseErrors(t *testing.T) {
	runDatabaseErrorCases(t, []dbErrorCase{
		{name: "APIGetUserProfileImage", handler: APIGetUserProfileImage, method: "GET", path: "/api/auth/users/00000000-0000-0000-0000-00000000000a/image", params: gin.Params{{Key: "user_id", Value: "00000000-0000-0000-0000-00000000000a"}}},
		{name: "APIGetWishImage", handler: APIGetWishImage, method: "GET", path: "/api/both/wishes/00000000-0000-0000-0000-00000000000a/image", params: gin.Params{{Key: "wish_id", Value: "00000000-0000-0000-0000-00000000000a"}}},
	})
}

// imageTestWithOrientation inserts a minimal big-endian EXIF APP1 segment
// carrying the given Orientation tag directly after the JPEG's SOI marker.
func imageTestWithOrientation(jpegBytes []byte, orientation byte) []byte {
	segment := []byte{
		0xFF, 0xE1, 0x00, 0x22, // APP1, length 34
		'E', 'x', 'i', 'f', 0x00, 0x00,
		'M', 'M', 0x00, 0x2A, 0x00, 0x00, 0x00, 0x08, // TIFF header, IFD0 at 8
		0x00, 0x01, // one entry
		0x01, 0x12, 0x00, 0x03, 0x00, 0x00, 0x00, 0x01, 0x00, orientation, 0x00, 0x00, // Orientation, SHORT
		0x00, 0x00, 0x00, 0x00, // no next IFD
	}
	out := append([]byte{}, jpegBytes[:2]...)
	out = append(out, segment...)
	return append(out, jpegBytes[2:]...)
}

func TestSaveWishImageAppliesOrientationAndStripsMetadata(t *testing.T) {
	imageTestSetup(t)
	wishID := uuid.New()

	raw := imageTestWithOrientation(imageTestRandomImageBytes(t, 300, 200, false), 6)
	if err := storeUploadedImage(wishImageDir, wishID, imageTestDataURI("image/jpeg", raw)); err != nil {
		t.Fatalf("storeUploadedImage failed: %v", err)
	}

	stored, err := os.ReadFile(filepath.Join(wishImageDir, wishID.String()+".jpg"))
	if err != nil {
		t.Fatalf("failed to read stored image: %v", err)
	}
	if bytes.Contains(stored, []byte("Exif")) {
		t.Error("stored image still contains an EXIF block")
	}
	config, err := jpeg.DecodeConfig(bytes.NewReader(stored))
	if err != nil {
		t.Fatalf("failed to decode stored image: %v", err)
	}
	if config.Width != 200 || config.Height != 300 {
		t.Errorf("stored image is %dx%d, want 200x300 (rotated upright)", config.Width, config.Height)
	}
}

// --- access control on private wishlists ---

func TestGetWishImagePrivateWishlistAccess(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	owner := createTestUser(t)
	wishlist := imageTestPrivateWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	groupMember := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, groupMember.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)

	collaborator := createTestUser(t)
	addWishCollaborator(t, wishlist.ID, collaborator.ID)

	outsider := createTestUser(t)

	cases := []struct {
		name   string
		userID uuid.UUID
		want   int
	}{
		{"owner", owner.ID, 200},
		{"group member", groupMember.ID, 200},
		{"collaborator", collaborator.ID, 200},
		{"unrelated user", outsider.ID, 403},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			header := map[string]string{"Authorization": authHeader(t, c.userID, false)}
			code, body := getWithParams(APIGetWishImage, "/api/wishes/x/image", gin.Params{{Key: "wish_id", Value: wish.ID.String()}}, header)
			if code != c.want {
				t.Errorf("status = %d, want %d; body=%v", code, c.want, body)
			}
		})
	}
}

// --- paths ---

func TestGetWishImageNonCanonicalUUIDServesSameFile(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	owner := createTestUser(t)
	wishlist := imageTestPublicWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	if err := storeUploadedImage(wishImageDir, wish.ID, imageTestDataURI("image/png", imageTestRandomImageBytes(t, 50, 50, true))); err != nil {
		t.Fatalf("failed to seed wish image: %v", err)
	}

	params := func(id string) gin.Params { return gin.Params{{Key: "wish_id", Value: id}} }
	_, canonical := getWithParams(APIGetWishImage, "/api/wishes/x/image", params(wish.ID.String()), nil)
	_, braced := getWithParams(APIGetWishImage, "/api/wishes/x/image", params("{"+wish.ID.String()+"}"), nil)
	if canonical["image"] == nil || canonical["image"] != braced["image"] {
		t.Error("a braced UUID should resolve to the same stored image as the canonical form")
	}
}

// --- upload processing ---

// imageTestPNGHeader returns the signature and IHDR chunk of a width x height
// 8-bit RGB PNG, with no pixel data: enough for its dimensions to be read,
// but decoding it fails.
func imageTestPNGHeader(width, height int) []byte {
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], uint32(width))
	binary.BigEndian.PutUint32(ihdr[4:8], uint32(height))
	ihdr[8], ihdr[9] = 8, 2 // 8-bit RGB
	chunk := append([]byte("IHDR"), ihdr...)
	header := []byte("\x89PNG\r\n\x1a\n")
	header = binary.BigEndian.AppendUint32(header, 13)
	header = append(header, chunk...)
	return binary.BigEndian.AppendUint32(header, crc32.ChecksumIEEE(chunk))
}

// imageTestPlainPNG encodes a uniform width x height grayscale PNG.
func imageTestPlainPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, image.NewGray(image.Rect(0, 0, width, height))); err != nil {
		t.Fatalf("failed to encode fixture: %v", err)
	}
	return buf.Bytes()
}

// imageTestPixelLimit sets max_image_pixels for the test.
func imageTestPixelLimit(t *testing.T, pixels int) {
	t.Helper()
	original := config.ConfigFile.MaxImagePixels
	t.Cleanup(func() { config.ConfigFile.MaxImagePixels = original })
	config.ConfigFile.MaxImagePixels = pixels
}

// storeUploadedImage decodes and stores an upload the way the handlers do.
func storeUploadedImage(dir string, id uuid.UUID, base64String string) error {
	img, err := decodeImageUpload(context.Background(), uuid.New(), base64String)
	if err != nil {
		return err
	}
	return storeImage(dir, id, img)
}

func TestSaveWishImageRejectsOversizedDimensions(t *testing.T) {
	imageTestSetup(t)

	err := storeUploadedImage(wishImageDir, uuid.New(), imageTestDataURI("image/png", imageTestPNGHeader(10000, 10000)))
	if err == nil || !strings.Contains(err.Error(), "dimensions") {
		t.Errorf("err = %v, want a dimensions error", err)
	}
}

// The pixel limit is checked on the header alone: an image over it is refused
// with the limit, never reaching the decoder, while one within it is decoded
// (and these fail to decode, having no pixel data).
func TestDecodeImageUploadChecksDefaultPixelLimitBeforeDecoding(t *testing.T) {
	imageTestPixelLimit(t, 0)

	for _, c := range []struct {
		width, height int
		refused       bool
	}{
		{4000, 3000, false},
		{6000, 4000, false},
		{6000, 4001, true},
		{10000, 10000, true},
	} {
		_, err := decodeImageUpload(context.Background(), uuid.New(), imageTestDataURI("image/png", imageTestPNGHeader(c.width, c.height)))
		refused := err != nil && strings.Contains(err.Error(), "at most 24 megapixels")
		if refused != c.refused {
			t.Errorf("%dx%d: err = %v, want refused by the pixel limit: %v", c.width, c.height, err, c.refused)
		}
		if err == nil {
			t.Errorf("%dx%d: decoded an image with no pixel data", c.width, c.height)
		}
	}
}

func TestDecodeImageUploadUsesConfiguredPixelLimit(t *testing.T) {
	imageTestPixelLimit(t, 1_500_000)

	img, err := decodeImageUpload(context.Background(), uuid.New(), imageTestDataURI("image/png", imageTestPlainPNG(t, 1500, 1000)))
	if err != nil {
		t.Fatalf("1500x1000 refused with a 1.5 megapixel limit: %v", err)
	}
	if bounds := img.Bounds(); bounds.Dx() != 1000 || bounds.Dy() != 667 {
		t.Errorf("decoded image is %dx%d, want it scaled down to 1000x667", bounds.Dx(), bounds.Dy())
	}

	_, err = decodeImageUpload(context.Background(), uuid.New(), imageTestDataURI("image/png", imageTestPlainPNG(t, 1501, 1000)))
	if err == nil || !strings.Contains(err.Error(), "at most 1.5 megapixels") {
		t.Errorf("1501x1000 err = %v, want the 1.5 megapixel limit", err)
	}
}

// imageTestHoldDecodeBudget takes the whole decode budget until the test ends
// or the returned function is called.
func imageTestHoldDecodeBudget(t *testing.T) (release func()) {
	t.Helper()
	if err := decodeBudget.Acquire(context.Background(), worstCaseConcurrentDecodes*decodeSharesPerImage); err != nil {
		t.Fatal(err)
	}
	var once sync.Once
	release = func() {
		once.Do(func() { decodeBudget.Release(worstCaseConcurrentDecodes * decodeSharesPerImage) })
	}
	t.Cleanup(release)
	return release
}

// imageTestWaitsForBudget runs fn while the decode budget is taken, checks
// that it waits, then frees the budget and checks that it completes.
func imageTestWaitsForBudget(t *testing.T, fn func()) {
	t.Helper()
	release := imageTestHoldDecodeBudget(t)
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
		t.Fatal("decoded an image while the decode budget was taken")
	case <-time.After(100 * time.Millisecond):
	}
	release()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("decoding did not resume once the budget was free")
	}
}

func TestDecodeImageUploadWaitsForDecodeBudget(t *testing.T) {
	uri := imageTestDataURI("image/png", imageTestPlainPNG(t, 50, 50))
	imageTestWaitsForBudget(t, func() {
		if _, err := decodeImageUpload(context.Background(), uuid.New(), uri); err != nil {
			t.Errorf("decodeImageUpload: %v", err)
		}
	})
}

// An upload stops waiting for budget when its request ends, and is refused as
// busy rather than decoded later for nobody.
func TestDecodeImageUploadGivesUpWhenRequestEnds(t *testing.T) {
	imageTestHoldDecodeBudget(t)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := decodeImageUpload(ctx, uuid.New(), imageTestDataURI("image/png", imageTestPlainPNG(t, 50, 50)))
	if !errors.Is(err, errImageBusy) {
		t.Errorf("err = %v, want errImageBusy", err)
	}
}

// One account decodes one upload at a time; others are not held up by it.
func TestDecodeImageUploadOneAtATimePerAccount(t *testing.T) {
	userID := uuid.New()
	imageUploadsInProgress.Store(userID, struct{}{})
	t.Cleanup(func() { imageUploadsInProgress.Delete(userID) })
	uri := imageTestDataURI("image/png", imageTestPlainPNG(t, 50, 50))

	if _, err := decodeImageUpload(context.Background(), userID, uri); !errors.Is(err, errImageUploadInProgress) {
		t.Errorf("second upload of an account: err = %v, want errImageUploadInProgress", err)
	}
	if _, err := decodeImageUpload(context.Background(), uuid.New(), uri); err != nil {
		t.Errorf("upload of another account: %v", err)
	}

	imageUploadsInProgress.Delete(userID)
	if _, err := decodeImageUpload(context.Background(), userID, uri); err != nil {
		t.Errorf("upload once the account's previous one finished: %v", err)
	}
	if _, busy := imageUploadsInProgress.Load(userID); busy {
		t.Error("the account is still marked as uploading after its upload finished")
	}
}

func TestRespondImageUploadError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, c := range []struct {
		err    error
		status int
		body   string
	}{
		{fmt.Errorf("%w (%w)", errImageBusy, context.DeadlineExceeded), 503, errImageBusy.Error()},
		{errImageUploadInProgress, 429, errImageUploadInProgress.Error()},
		{errImageTooCostly, 400, errImageTooCostly.Error()},
		{errors.New("Image is too large."), 400, "Image is too large."},
	} {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		respondImageUploadError(ctx, c.err)
		var body map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		if w.Code != c.status || body["error"] != c.body || !ctx.IsAborted() {
			t.Errorf("%v: got %d %q (aborted %v), want %d %q", c.err, w.Code, body["error"], ctx.IsAborted(), c.status, c.body)
		}
	}
}

// imageTestWriteLegacyFile stores raw as id's full-size image, as an older
// version would have.
func imageTestWriteLegacyFile(t *testing.T, id uuid.UUID, raw []byte) {
	t.Helper()
	if err := os.MkdirAll(wishImageDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(imageFilePath(wishImageDir, id, false), raw, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadStoredImageWaitsForDecodeBudget(t *testing.T) {
	imageTestSetup(t)
	id := uuid.New()
	imageTestWriteLegacyFile(t, id, imageTestPlainPNG(t, 1200, 800))
	imageTestWaitsForBudget(t, func() {
		if _, found, err := loadStoredImage(context.Background(), wishImageDir, id, true); err != nil || !found {
			t.Errorf("loadStoredImage = (%v, %v), want the thumbnail", found, err)
		}
	})
}

// A legacy image that would need resizing is served as stored, without being
// resized or written back, when the decoder stays busy for the whole request.
func TestLoadStoredImageServesAsStoredWhileBusy(t *testing.T) {
	imageTestSetup(t)
	id := uuid.New()
	stored := imageTestPlainPNG(t, 1200, 800)
	imageTestWriteLegacyFile(t, id, stored)
	imageTestHoldDecodeBudget(t)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	served, found, err := loadStoredImage(ctx, wishImageDir, id, true)
	if err != nil || !found || !bytes.Equal(served, stored) {
		t.Errorf("loadStoredImage = (%d bytes, %v, %v), want the stored file", len(served), found, err)
	}
	if _, err := os.Stat(imageFilePath(wishImageDir, id, true)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("thumbnail written while busy: stat err = %v", err)
	}
}

// A legacy image within the pixel limit whose decode would cost more than one
// upload may (here a very tall, narrow one) is served as stored too.
func TestLoadStoredImageServesCostlyFileAsStored(t *testing.T) {
	imageTestSetup(t)
	imageTestPixelLimit(t, 1_000_000)
	id := uuid.New()
	stored := imageTestPlainPNG(t, 1, 1_000_000)
	imageTestWriteLegacyFile(t, id, stored)

	served, found, err := loadStoredImage(context.Background(), wishImageDir, id, true)
	if err != nil || !found || !bytes.Equal(served, stored) {
		t.Errorf("loadStoredImage = (%d bytes, %v, %v), want the stored file", len(served), found, err)
	}
}

// A stored file over the pixel limit is served as stored, thumbnail or not,
// rather than decoded to resize it.
func TestLoadStoredImageServesFileOverPixelLimitAsStored(t *testing.T) {
	imageTestSetup(t)
	imageTestPixelLimit(t, 0)
	id := uuid.New()
	stored := imageTestPNGHeader(6000, 4001)
	if err := os.MkdirAll(wishImageDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(imageFilePath(wishImageDir, id, false), stored, 0644); err != nil {
		t.Fatal(err)
	}

	for _, thumbnail := range []bool{false, true} {
		served, found, err := loadStoredImage(context.Background(), wishImageDir, id, thumbnail)
		if err != nil || !found || !bytes.Equal(served, stored) {
			t.Errorf("thumbnail=%v: loadStoredImage = (%d bytes, %v, %v), want the stored file", thumbnail, len(served), found, err)
		}
	}
}

func TestSaveWishImageFlattensTransparencyOnWhite(t *testing.T) {
	imageTestSetup(t)
	wishID := uuid.New()

	buf := new(bytes.Buffer)
	if err := png.Encode(buf, image.NewNRGBA(image.Rect(0, 0, 20, 20))); err != nil { // fully transparent
		t.Fatalf("failed to encode fixture: %v", err)
	}
	if err := storeUploadedImage(wishImageDir, wishID, imageTestDataURI("image/png", buf.Bytes())); err != nil {
		t.Fatalf("storeUploadedImage failed: %v", err)
	}

	stored, err := os.ReadFile(imageFilePath(wishImageDir, wishID, false))
	if err != nil {
		t.Fatalf("failed to read stored image: %v", err)
	}
	img, err := jpeg.Decode(bytes.NewReader(stored))
	if err != nil {
		t.Fatalf("failed to decode stored image: %v", err)
	}
	r, g, b, _ := img.At(10, 10).RGBA()
	if r>>8 < 250 || g>>8 < 250 || b>>8 < 250 {
		t.Errorf("transparent pixel stored as (%d,%d,%d), want white", r>>8, g>>8, b>>8)
	}
}

func TestSaveWishImageStoresResizedVariants(t *testing.T) {
	imageTestSetup(t)
	wishID := uuid.New()

	raw := imageTestRandomImageBytes(t, 1600, 800, false)
	if err := storeUploadedImage(wishImageDir, wishID, imageTestDataURI("image/jpeg", raw)); err != nil {
		t.Fatalf("storeUploadedImage failed: %v", err)
	}

	for _, c := range []struct {
		thumbnail             bool
		wantWidth, wantHeight int
	}{{false, 1000, 500}, {true, 250, 125}} {
		stored, err := os.ReadFile(imageFilePath(wishImageDir, wishID, c.thumbnail))
		if err != nil {
			t.Fatalf("thumbnail=%v: failed to read stored image: %v", c.thumbnail, err)
		}
		config, err := jpeg.DecodeConfig(bytes.NewReader(stored))
		if err != nil {
			t.Fatalf("thumbnail=%v: failed to decode stored image: %v", c.thumbnail, err)
		}
		if config.Width != c.wantWidth || config.Height != c.wantHeight {
			t.Errorf("thumbnail=%v: stored %dx%d, want %dx%d", c.thumbnail, config.Width, config.Height, c.wantWidth, c.wantHeight)
		}
	}
}

// --- serving ---

func TestLoadStoredImageMigratesLegacyFullResolutionFile(t *testing.T) {
	imageTestSetup(t)
	wishID := uuid.New()

	// Older versions stored the decoded original as-is, with no thumbnail.
	if err := os.MkdirAll(wishImageDir, 0755); err != nil {
		t.Fatalf("failed to create image dir: %v", err)
	}
	legacy := imageTestRandomImageBytes(t, 1500, 1200, false)
	if err := os.WriteFile(imageFilePath(wishImageDir, wishID, false), legacy, 0644); err != nil {
		t.Fatalf("failed to seed legacy image: %v", err)
	}

	thumbnailBytes, found, err := loadStoredImage(context.Background(), wishImageDir, wishID, true)
	if err != nil || !found {
		t.Fatalf("found = %v err = %v, want the legacy image to be served", found, err)
	}
	if config, _ := jpeg.DecodeConfig(bytes.NewReader(thumbnailBytes)); config.Width > 250 || config.Height > 250 {
		t.Errorf("served thumbnail is %dx%d, want within 250x250", config.Width, config.Height)
	}

	for _, thumbnail := range []bool{false, true} {
		stored, err := os.ReadFile(imageFilePath(wishImageDir, wishID, thumbnail))
		if err != nil {
			t.Fatalf("thumbnail=%v: expected a written-back file: %v", thumbnail, err)
		}
		config, _ := jpeg.DecodeConfig(bytes.NewReader(stored))
		if config.Width > 1000 || config.Height > 1000 {
			t.Errorf("thumbnail=%v: written-back file is %dx%d, want within 1000x1000", thumbnail, config.Width, config.Height)
		}
	}
}

func TestGetWishImageConditionalRequest(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	owner := createTestUser(t)
	wishlist := imageTestPublicWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	request := func(ifNoneMatch string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = httptest.NewRequest("GET", "/api/wishes/x/image", nil)
		if ifNoneMatch != "" {
			ctx.Request.Header.Set("If-None-Match", ifNoneMatch)
		}
		ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
		APIGetWishImage(ctx)
		return w
	}

	first := request("")
	etag := first.Header().Get("ETag")
	if first.Code != 200 || etag == "" {
		t.Fatalf("status = %d etag = %q, want 200 with an ETag", first.Code, etag)
	}
	if cached := request(etag); cached.Code != 304 || cached.Body.Len() != 0 {
		t.Errorf("status = %d body = %d bytes, want an empty 304 for a matching ETag", cached.Code, cached.Body.Len())
	}
	if stale := request(`"stale"`); stale.Code != 200 {
		t.Errorf("status = %d, want 200 for a non-matching ETag", stale.Code)
	}
}

// --- cleanup on delete ---

func TestDeleteWishRemovesImage(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)
	if err := storeUploadedImage(wishImageDir, wish.ID, imageTestDataURI("image/png", imageTestRandomImageBytes(t, 50, 50, true))); err != nil {
		t.Fatalf("failed to seed wish image: %v", err)
	}

	w, ctx := wishCtx("DELETE", "/api/wishes/"+wish.ID.String(), "")
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	DeleteWish(ctx)
	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}

	imageTestAssertNoFiles(t, wishImageDir, wish.ID)
}

func TestDeleteWishlistRemovesWishImages(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wishes := []models.Wish{createTestWish(t, owner.ID, wishlist.ID), createTestWish(t, owner.ID, wishlist.ID)}
	for _, wish := range wishes {
		if err := storeUploadedImage(wishImageDir, wish.ID, imageTestDataURI("image/png", imageTestRandomImageBytes(t, 50, 50, true))); err != nil {
			t.Fatalf("failed to seed wish image: %v", err)
		}
	}

	code, resp := wlDo(DeleteWishlist, "DELETE", "/api/auth/wishlists/"+wishlist.ID.String(), "", authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}

	for _, wish := range wishes {
		imageTestAssertNoFiles(t, wishImageDir, wish.ID)
	}
}

func TestAPIDeleteUserRemovesProfileImage(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	admin := createTestUser(t)
	target := createTestUser(t)
	if err := storeUploadedImage(profileImageDir, target.ID, imageTestDataURI("image/png", imageTestRandomImageBytes(t, 50, 50, true))); err != nil {
		t.Fatalf("failed to seed profile image: %v", err)
	}

	header := map[string]string{"Authorization": authHeader(t, admin.ID, true)}
	params := gin.Params{{Key: "user_id", Value: target.ID.String()}}
	code, resp, _ := doRequest(APIDeleteUser, "DELETE", "/api/admin/users/"+target.ID.String(), "", header, params)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}

	imageTestAssertNoFiles(t, profileImageDir, target.ID)
}

func imageTestAssertNoFiles(t *testing.T, dir string, id uuid.UUID) {
	t.Helper()
	for _, thumbnail := range []bool{false, true} {
		if _, err := os.Stat(imageFilePath(dir, id, thumbnail)); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("thumbnail=%v: expected image file to be deleted, stat err = %v", thumbnail, err)
		}
	}
}

// --- error paths ---

// imageTestReadOnlyDir makes dir read-only for the rest of the test, so files
// in it stay readable but nothing new can be created or removed.
func imageTestReadOnlyDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.Chmod(dir, 0555); err != nil {
		t.Fatalf("failed to chmod %s: %v", dir, err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0755) })
}

// imageTestSeedFile writes raw bytes at path, creating its directory.
func imageTestSeedFile(t *testing.T, path string, raw []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create image dir: %v", err)
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatalf("failed to seed image: %v", err)
	}
}

func TestGetUserProfileImageDefaultMissing(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	defaultProfileImagePath = filepath.Join(t.TempDir(), "missing.svg")
	user := createTestUser(t)

	code, body := getWithParams(APIGetUserProfileImage, "/api/users/x/image", gin.Params{{Key: "user_id", Value: user.ID.String()}}, nil)
	if code != 500 || body["error"] != "Failed to load default profile image." {
		t.Errorf("status = %d body=%v, want 500 for a missing default avatar", code, body)
	}
}

func TestGetUserProfileImageConditionalRequest(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	user := createTestUser(t)

	request := func(ifNoneMatch string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = httptest.NewRequest("GET", "/api/users/x/image", nil)
		if ifNoneMatch != "" {
			ctx.Request.Header.Set("If-None-Match", ifNoneMatch)
		}
		ctx.Params = gin.Params{{Key: "user_id", Value: user.ID.String()}}
		APIGetUserProfileImage(ctx)
		return w
	}

	first := request("")
	etag := first.Header().Get("ETag")
	if first.Code != 200 || etag == "" {
		t.Fatalf("status = %d etag = %q, want 200 with an ETag", first.Code, etag)
	}
	if cached := request(etag); cached.Code != 304 || cached.Body.Len() != 0 {
		t.Errorf("status = %d body = %d bytes, want an empty 304 for a matching ETag", cached.Code, cached.Body.Len())
	}
}

func TestGetWishImageDefaultMissing(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	defaultProfileImagePath = filepath.Join(t.TempDir(), "missing.svg")
	owner := createTestUser(t)
	wishlist := imageTestPublicWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	code, body := getWithParams(APIGetWishImage, "/api/wishes/x/image", gin.Params{{Key: "wish_id", Value: wish.ID.String()}}, nil)
	if code != 500 || body["error"] != "Failed to load default profile image." {
		t.Errorf("status = %d body=%v, want 500 for a missing default image", code, body)
	}
}

func TestGetWishImageAccessCheckFails(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	owner := createTestUser(t)
	viewer := createTestUser(t)
	wishlist := imageTestPrivateWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)
	header := map[string]string{"Authorization": authHeader(t, viewer.ID, false)}
	// The viewer is neither owner nor group member, so the check reaches the
	// collaborator lookup, which is the one made to fail.
	failDBOperation(t, "query", "wishlist_collaborators", 0)

	code, body := getWithParams(APIGetWishImage, "/api/wishes/x/image", gin.Params{{Key: "wish_id", Value: wish.ID.String()}}, header)
	if code != 500 || body["error"] != "Failed to verify wishlist access." {
		t.Errorf("status = %d body=%v, want 500 when the access check errors", code, body)
	}
}

func TestLoadStoredImageReadErrors(t *testing.T) {
	cases := []struct {
		name      string
		dirPath   func(dir string, id uuid.UUID) string
		thumbnail bool
	}{
		{"thumbnail path is a directory", func(dir string, id uuid.UUID) string { return imageFilePath(dir, id, true) }, true},
		{"full path is a directory", func(dir string, id uuid.UUID) string { return imageFilePath(dir, id, false) }, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			id := uuid.New()
			if err := os.MkdirAll(c.dirPath(dir, id), 0755); err != nil {
				t.Fatalf("failed to create blocking directory: %v", err)
			}

			data, found, err := loadStoredImage(context.Background(), dir, id, c.thumbnail)
			if err == nil || found || data != nil {
				t.Errorf("loadStoredImage = (%d bytes, %v, %v), want a read error", len(data), found, err)
			}
		})
	}
}

func TestLoadStoredImageTruncatedFile(t *testing.T) {
	dir := t.TempDir()
	id := uuid.New()
	full := imageTestRandomImageBytes(t, 300, 300, false)
	// Keep the header (so DecodeConfig passes) but cut off the scan data.
	imageTestSeedFile(t, imageFilePath(dir, id, false), full[:len(full)/2])

	_, found, err := loadStoredImage(context.Background(), dir, id, true)
	if err == nil || found {
		t.Errorf("found = %v err = %v, want a decode error for a truncated image", found, err)
	}
}

func TestLoadStoredImageWriteBackFailuresStillServe(t *testing.T) {
	cases := []struct {
		name          string
		width, height int
		thumbnail     bool
		maxDimension  int
	}{
		{"oversized legacy full image", 1500, 1200, false, 1000},
		{"legacy image without thumbnail", 300, 300, true, 250},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			id := uuid.New()
			original := imageTestRandomImageBytes(t, c.width, c.height, false)
			imageTestSeedFile(t, imageFilePath(dir, id, false), original)
			imageTestReadOnlyDir(t, dir)

			data, found, err := loadStoredImage(context.Background(), dir, id, c.thumbnail)
			if err != nil || !found {
				t.Fatalf("found = %v err = %v, want the image served despite the failed write-back", found, err)
			}
			config, err := jpeg.DecodeConfig(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("served bytes aren't a JPEG: %v", err)
			}
			if config.Width > c.maxDimension || config.Height > c.maxDimension {
				t.Errorf("served image is %dx%d, want within %d", config.Width, config.Height, c.maxDimension)
			}

			// Nothing could be written back, so the original stays as it was.
			stored, _ := os.ReadFile(imageFilePath(dir, id, false))
			if !bytes.Equal(stored, original) {
				t.Error("stored full image changed despite a read-only directory")
			}
			if _, err := os.Stat(imageFilePath(dir, id, true)); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("thumbnail stat err = %v, want not-exist", err)
			}
		})
	}
}

func TestCheckIfWishImageExistsStatError(t *testing.T) {
	imageTestSetup(t)
	// A regular file where the directory should be makes Stat fail with
	// ENOTDIR rather than not-exist.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatalf("failed to seed blocking file: %v", err)
	}
	wishImageDir = blocker

	exists, err := CheckIfWishImageExists(uuid.New())
	if err == nil || exists {
		t.Errorf("CheckIfWishImageExists = (%v, %v), want (false, error)", exists, err)
	}
}

func TestDecodeUploadedImageTruncated(t *testing.T) {
	cases := []struct {
		mime  string
		asPNG bool
	}{
		{"image/jpeg", false},
		{"image/png", true},
	}
	for _, c := range cases {
		t.Run(c.mime, func(t *testing.T) {
			raw := imageTestRandomImageBytes(t, 200, 200, c.asPNG)
			img, err := decodeUploadedImage(context.Background(), raw[:len(raw)/2], c.mime)
			if err == nil || img != nil || err.Error() != "Failed to create image from byte array." {
				t.Errorf("decodeUploadedImage = (%v, %v), want the decode error", img, err)
			}
		})
	}
}

// The MIME type comes straight from the caller's data URI, so it is logged
// quoted: a CRLF in it must not reach the log as a line break
// (GHSA-pmhf-h957-p2v3).
func TestDecodeUploadedImageQuotesInvalidMimeType(t *testing.T) {
	hook := logrusTest.NewLocal(logger.Log)
	t.Cleanup(func() { logger.Log.ReplaceHooks(make(logrus.LevelHooks)) })

	payload := "data:image/png\r\n[INFO]: 2026-11-03 14:03:16 - User admin successfully authenticated from 127.0.0.1;base64," +
		base64.StdEncoding.EncodeToString(imageTestRandomImageBytes(t, 1, 1, true))
	err := storeUploadedImage(t.TempDir(), uuid.New(), payload)
	if err == nil || err.Error() != "Invalid image type." {
		t.Fatalf("storeUploadedImage error = %v, want the invalid type error", err)
	}

	entry := hook.LastEntry()
	if entry == nil {
		t.Fatal("expected the invalid mime type to be logged")
	}
	if strings.ContainsAny(entry.Message, "\r\n") {
		t.Errorf("logged message contains a raw line break: %q", entry.Message)
	}
	if !strings.Contains(entry.Message, `"image/png\r\n[INFO]`) {
		t.Errorf("logged message = %q, want the mime type quoted", entry.Message)
	}
}

func TestSaveWishImageWriteFails(t *testing.T) {
	imageTestSetup(t)
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatalf("failed to seed blocking file: %v", err)
	}
	wishImageDir = blocker

	raw := imageTestRandomImageBytes(t, 200, 200, false)
	err := storeUploadedImage(wishImageDir, uuid.New(), imageTestDataURI("image/jpeg", raw))
	if err == nil || err.Error() != "Failed to save image to disk." {
		t.Errorf("err = %v, want 'Failed to save image to disk.'", err)
	}
}

func TestEncodeJPEGTooLarge(t *testing.T) {
	// JPEG can't encode a dimension of 65536 or more.
	img := image.NewGray(image.Rect(0, 0, 1<<16, 1))
	if data, err := encodeJPEG(img); err == nil || data != nil {
		t.Errorf("encodeJPEG = (%d bytes, %v), want an error", len(data), err)
	}
}

func TestWriteFileAtomicRenameFailure(t *testing.T) {
	dir := t.TempDir()
	// A non-empty directory at the target path can't be replaced by rename.
	target := filepath.Join(dir, "pic.jpg")
	imageTestSeedFile(t, filepath.Join(target, "inner"), []byte("x"))

	if err := writeFileAtomic(target, []byte("data")); err == nil {
		t.Fatal("expected an error when the target is a non-empty directory")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to list dir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("dir has %d entries, want only the blocking directory (temp file cleaned up)", len(entries))
	}
}

func TestDeleteUserProfileImageRemoveFailure(t *testing.T) {
	imageTestSetup(t)
	userID := uuid.New()
	raw := imageTestRandomImageBytes(t, 200, 200, false)
	if err := storeUploadedImage(profileImageDir, userID, imageTestDataURI("image/jpeg", raw)); err != nil {
		t.Fatalf("failed to save profile image: %v", err)
	}
	imageTestReadOnlyDir(t, profileImageDir)

	if err := DeleteUserProfileImage(userID); err == nil || err.Error() != "Failed to delete profile image." {
		t.Errorf("err = %v, want 'Failed to delete profile image.'", err)
	}
	if _, err := os.Stat(imageFilePath(profileImageDir, userID, false)); err != nil {
		t.Errorf("profile image should still exist: %v", err)
	}
}
