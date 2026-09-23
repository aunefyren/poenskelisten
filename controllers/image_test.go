package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
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
	if err := UpdateUserProfileImage(user.ID, imageTestDataURI("image/png", raw)); err != nil {
		t.Errorf("expected a small but valid image to be accepted, got: %v", err)
	}
}

func TestUpdateUserProfileImageGarbageRejected(t *testing.T) {
	setupControllersDB(t)
	imageTestSetup(t)
	user := createTestUser(t)

	if err := UpdateUserProfileImage(user.ID, imageTestDataURI("image/jpeg", []byte("short"))); err == nil {
		t.Error("expected an error for bytes that aren't a decodable image")
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
	if err := SaveWishImage(wishID, imageTestDataURI("image/jpeg", raw)); err != nil {
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
	if err := SaveWishImage(wishID, imageTestDataURI("image/jpeg", raw)); err != nil {
		t.Fatalf("SaveWishImage failed: %v", err)
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

	if err := SaveWishImage(wish.ID, imageTestDataURI("image/png", imageTestRandomImageBytes(t, 50, 50, true))); err != nil {
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

func TestSaveWishImageRejectsOversizedDimensions(t *testing.T) {
	imageTestSetup(t)

	// A PNG header declaring 10000x10000 (100 MP). Only the header is needed:
	// the upload must be rejected before any pixel data is decoded.
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], 10000)
	binary.BigEndian.PutUint32(ihdr[4:8], 10000)
	ihdr[8], ihdr[9] = 8, 2 // 8-bit RGB
	chunk := append([]byte("IHDR"), ihdr...)
	header := []byte("\x89PNG\r\n\x1a\n")
	header = binary.BigEndian.AppendUint32(header, 13)
	header = append(header, chunk...)
	header = binary.BigEndian.AppendUint32(header, crc32.ChecksumIEEE(chunk))

	err := SaveWishImage(uuid.New(), imageTestDataURI("image/png", header))
	if err == nil || !strings.Contains(err.Error(), "dimensions") {
		t.Errorf("err = %v, want a dimensions error", err)
	}
}

func TestSaveWishImageFlattensTransparencyOnWhite(t *testing.T) {
	imageTestSetup(t)
	wishID := uuid.New()

	buf := new(bytes.Buffer)
	if err := png.Encode(buf, image.NewNRGBA(image.Rect(0, 0, 20, 20))); err != nil { // fully transparent
		t.Fatalf("failed to encode fixture: %v", err)
	}
	if err := SaveWishImage(wishID, imageTestDataURI("image/png", buf.Bytes())); err != nil {
		t.Fatalf("SaveWishImage failed: %v", err)
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
	if err := SaveWishImage(wishID, imageTestDataURI("image/jpeg", raw)); err != nil {
		t.Fatalf("SaveWishImage failed: %v", err)
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

	thumbnailBytes, found, err := loadStoredImage(wishImageDir, wishID, true)
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
	if err := SaveWishImage(wish.ID, imageTestDataURI("image/png", imageTestRandomImageBytes(t, 50, 50, true))); err != nil {
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
		if err := SaveWishImage(wish.ID, imageTestDataURI("image/png", imageTestRandomImageBytes(t, 50, 50, true))); err != nil {
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
	if err := UpdateUserProfileImage(target.ID, imageTestDataURI("image/png", imageTestRandomImageBytes(t, 50, 50, true))); err != nil {
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
