package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"
	"image/png"
	"math/rand"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// imageTestSetup points the package-level image path vars at a temp directory
// and a real default avatar (the ones baked in at package init resolve relative
// to the package's source dir when running `go test`, which doesn't contain a
// web/assets tree), restoring the originals afterwards. Safe to call from
// multiple tests since each gets its own temp dir.
func imageTestSetup(t *testing.T) {
	t.Helper()

	origProfile, origWish, origDefault := profile_image_path, wish_image_path, default_profile_image_path
	t.Cleanup(func() {
		profile_image_path, wish_image_path, default_profile_image_path = origProfile, origWish, origDefault
	})

	dir := t.TempDir()
	profile_image_path = filepath.Join(dir, "profiles")
	wish_image_path = filepath.Join(dir, "wishes")

	repoDefault, err := filepath.Abs("../web/assets/user.svg")
	if err != nil {
		t.Fatalf("failed to resolve default avatar path: %v", err)
	}
	if _, err := os.Stat(repoDefault); err != nil {
		t.Fatalf("default avatar fixture missing at %s: %v", repoDefault, err)
	}
	default_profile_image_path = repoDefault
}

// imageTestRandomImageBytes builds a width x height image filled with random
// pixels (so JPEG/PNG compression can't shrink it well below its raw size) and
// encodes it, returning the raw (non-base64) encoded file bytes. Business logic
// in image.go rejects encoded images under 10000 bytes; callers that exercise
// that success path (e.g. UpdateUserProfileImage/SaveWishImage) need to pick a
// width/height large enough to clear it - 200x200 comfortably does.
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
	if code != 500 {
		t.Fatalf("status = %d, want 500 for a user that doesn't exist", code)
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
	if err := UpdateUserProfileImage(user.ID, imageTestDataURI("image/jpeg", raw)); err != nil {
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
	if err := SaveWishImage(wish.ID, imageTestDataURI("image/png", raw)); err != nil {
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
	out, err := ImageBytesToBase64(jpegBytes)
	if err != nil || out[:len("data:image/jpeg;base64,")] != "data:image/jpeg;base64," {
		t.Errorf("ImageBytesToBase64(jpeg) = %q, err=%v", out[:min(30, len(out))], err)
	}

	pngBytes := imageTestRandomImageBytes(t, 50, 50, true)
	out, err = ImageBytesToBase64(pngBytes)
	if err != nil || out[:len("data:image/png;base64,")] != "data:image/png;base64," {
		t.Errorf("ImageBytesToBase64(png) = %q, err=%v", out[:min(30, len(out))], err)
	}

	out, err = ImageBytesToBase64([]byte("not an image, just plain bytes"))
	if err != nil || out[:len("data:image/svg+xml;base64,")] != "data:image/svg+xml;base64," {
		t.Errorf("ImageBytesToBase64(unknown) should default to svg+xml, got %q err=%v", out[:min(40, len(out))], err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

func TestResizeImageInvalidBytes(t *testing.T) {
	_, err := ResizeImage(100, 100, []byte("not an image"))
	if err == nil {
		t.Error("expected an error resizing non-image bytes")
	}
}

func TestResizeImageValid(t *testing.T) {
	raw := imageTestRandomImageBytes(t, 400, 400, false)
	resized, err := ResizeImage(100, 100, raw)
	if err != nil {
		t.Fatalf("ResizeImage failed: %v", err)
	}
	img, _, err := image.Decode(bytes.NewReader(resized))
	if err != nil {
		t.Fatalf("resized output is not a decodable image: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() > 100 || bounds.Dy() > 100 {
		t.Errorf("resized image is %dx%d, want within 100x100", bounds.Dx(), bounds.Dy())
	}
}

func TestUpdateUserProfileImageTooSmall(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	user := createTestUser(t)

	tiny := imageTestDataURI("image/jpeg", []byte("short"))
	if err := UpdateUserProfileImage(user.ID, tiny); err == nil {
		t.Error("expected an error for an image under the minimum size")
	}
}

func TestUpdateUserProfileImageInvalidMimeType(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	user := createTestUser(t)

	raw := imageTestRandomImageBytes(t, 200, 200, false)
	uri := imageTestDataURI("image/gif", raw)
	if err := UpdateUserProfileImage(user.ID, uri); err == nil {
		t.Error("expected an error for an unsupported mime type")
	}
}

func TestUpdateUserProfileImagePNGSuccess(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	user := createTestUser(t)

	raw := imageTestRandomImageBytes(t, 200, 200, true)
	if err := UpdateUserProfileImage(user.ID, imageTestDataURI("image/png", raw)); err != nil {
		t.Fatalf("UpdateUserProfileImage failed for a valid png: %v", err)
	}
	if _, err := os.Stat(filepath.Join(profile_image_path, user.ID.String()+".jpg")); err != nil {
		t.Errorf("expected the profile image file to be saved to disk: %v", err)
	}
}

func TestLoadImageFileMissing(t *testing.T) {
	if _, err := LoadImageFile("/does/not/exist.jpg"); err == nil {
		t.Error("expected an error loading a missing file")
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
	if err := SaveWishImage(wishID, imageTestDataURI("image/jpeg", raw)); err != nil {
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
	if err := SaveWishImage(wishID, imageTestDataURI("image/jpeg", raw)); err != nil {
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
	if err := UpdateUserProfileImage(user.ID, uri); err == nil {
		t.Error("expected an error for an image over the maximum size")
	}
}

func TestSaveWishImageTooLarge(t *testing.T) {
	imageTestSetup(t)
	huge := make([]byte, 10_000_001)
	uri := imageTestDataURI("image/jpeg", huge)
	if err := SaveWishImage(uuid.New(), uri); err == nil {
		t.Error("expected an error for an image over the maximum size")
	}
}

func TestLoadDefaultProfileImageMissingFile(t *testing.T) {
	imageTestSetup(t)
	default_profile_image_path = filepath.Join(t.TempDir(), "does-not-exist.svg")

	if _, err := LoadDefaultProfileImage(); err == nil {
		t.Error("expected an error when the default avatar file is missing")
	}
}

func TestSaveImageFileDirectoryCollision(t *testing.T) {
	dir := t.TempDir()
	// Create a plain file where SaveImageFile wants to MkdirAll a directory,
	// so the MkdirAll call fails.
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatalf("failed to seed blocking file: %v", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	if err := SaveImageFile(blocker, "pic.jpg", img); err == nil {
		t.Error("expected an error when the target path collides with an existing file")
	}
}

func TestGetUserProfileImageResizeFailure(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	user := createTestUser(t)

	if err := os.MkdirAll(profile_image_path, 0755); err != nil {
		t.Fatalf("failed to create profile image dir: %v", err)
	}
	corrupt := filepath.Join(profile_image_path, user.ID.String()+".jpg")
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

	if err := os.MkdirAll(wish_image_path, 0755); err != nil {
		t.Fatalf("failed to create wish image dir: %v", err)
	}
	corrupt := filepath.Join(wish_image_path, wish.ID.String()+".jpg")
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
	if err := SaveWishImage(wishID, imageTestDataURI("image/jpeg", raw)); err != nil {
		t.Fatalf("failed to save wish image: %v", err)
	}

	// Strip write permission from the containing directory so the file is
	// still readable (CheckIfWishImageExists succeeds) but os.Remove fails.
	if err := os.Chmod(wish_image_path, 0555); err != nil {
		t.Fatalf("failed to chmod wish image dir: %v", err)
	}
	t.Cleanup(func() { os.Chmod(wish_image_path, 0755) })

	if err := DeleteWishImage(wishID); err == nil {
		t.Error("expected an error when the image file can't be removed")
	}
}

func TestSaveImageFileCreateFailure(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "pics")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}
	if err := os.Chmod(target, 0555); err != nil {
		t.Fatalf("failed to chmod target dir: %v", err)
	}
	t.Cleanup(func() { os.Chmod(target, 0755) })

	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	if err := SaveImageFile(target, "pic.jpg", img); err == nil {
		t.Error("expected an error when the target directory isn't writable")
	}
}

func TestImageHandlersDatabaseErrors(t *testing.T) {
	runDatabaseErrorCases(t, []dbErrorCase{
		{name: "APIGetUserProfileImage", handler: APIGetUserProfileImage, method: "GET", path: "/api/auth/users/00000000-0000-0000-0000-00000000000a/image", params: gin.Params{{Key: "user_id", Value: "00000000-0000-0000-0000-00000000000a"}}},
		{name: "APIGetWishImage", handler: APIGetWishImage, method: "GET", path: "/api/both/wishes/00000000-0000-0000-0000-00000000000a/image", params: gin.Params{{Key: "wish_id", Value: "00000000-0000-0000-0000-00000000000a"}}},
	})
}
