package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// wishCtx builds a gin test context for a request with an optional JSON body.
func wishCtx(method, path, body string) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	ctx.Request = httptest.NewRequest(method, path, reader)
	ctx.Request.Header.Set("Content-Type", "application/json")
	return w, ctx
}

// makeClaimable flips a wishlist's Claimable flag on and persists it.
func makeClaimable(t *testing.T, wishlist models.Wishlist) models.Wishlist {
	t.Helper()
	wishlist.Claimable = boolPtr(true)
	updated, err := database.UpdateWishlistInDB(wishlist)
	if err != nil {
		t.Fatalf("failed to make wishlist claimable: %v", err)
	}
	return updated
}

// addWishCollaborator adds userID as a collaborator on wishlistID.
func addWishCollaborator(t *testing.T, wishlistID, userID uuid.UUID) {
	t.Helper()
	collab := models.WishlistCollaborator{
		UserID:     userID,
		WishlistID: wishlistID,
		Enabled:    true,
	}
	collab.ID = uuid.New()
	if err := database.CreateWishlistCollaboratorInDB(collab); err != nil {
		t.Fatalf("failed to add wishlist collaborator: %v", err)
	}
}

func parseWishBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response body: %v; raw=%s", err, w.Body.String())
	}
	return body
}

// --- GetWishesFromWishlist ---

func TestGetWishesFromWishlist_Success(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("GET", "/api/wishes?wishlist="+wishlist.ID.String(), "")
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	GetWishesFromWishlist(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	body := parseWishBody(t, w)
	wishes, ok := body["wishes"].([]interface{})
	if !ok || len(wishes) != 1 {
		t.Errorf("wishes = %v, want 1 entry", body["wishes"])
	}
}

func TestGetWishesFromWishlist_GroupMemberSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	member := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, member.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)
	createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("GET", "/api/wishes?wishlist="+wishlist.ID.String(), "")
	ctx.Request.Header.Set("Authorization", authHeader(t, member.ID, false))
	GetWishesFromWishlist(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestGetWishesFromWishlist_MissingQuery(t *testing.T) {
	setupControllersDB(t)
	w, ctx := wishCtx("GET", "/api/wishes", "")
	GetWishesFromWishlist(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 without a wishlist query param", w.Code)
	}
}

func TestGetWishesFromWishlist_MissingAuth(t *testing.T) {
	setupControllersDB(t)
	w, ctx := wishCtx("GET", "/api/wishes?wishlist="+uuid.NewString(), "")
	GetWishesFromWishlist(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 without auth", w.Code)
	}
}

func TestGetWishesFromWishlist_InvalidWishlistID(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	w, ctx := wishCtx("GET", "/api/wishes?wishlist=not-a-uuid", "")
	ctx.Request.Header.Set("Authorization", authHeader(t, user.ID, false))
	GetWishesFromWishlist(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed wishlist id", w.Code)
	}
}

func TestGetWishesFromWishlist_NotMember(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	stranger := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	w, ctx := wishCtx("GET", "/api/wishes?wishlist="+wishlist.ID.String(), "")
	ctx.Request.Header.Set("Authorization", authHeader(t, stranger.ID, false))
	GetWishesFromWishlist(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a non-member", w.Code)
	}
}

func TestGetWishesFromWishlist_WishesQueryFailure(t *testing.T) {
	// Migrate without the Wish table so database.GetWishesFromWishlist fails
	// after ownership verification succeeds.
	setupControllersDB(t, &models.User{}, &models.Wishlist{}, &models.WishlistMembership{}, &models.GroupMembership{}, &models.Group{})
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	w, ctx := wishCtx("GET", "/api/wishes?wishlist="+wishlist.ID.String(), "")
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	GetWishesFromWishlist(ctx)
	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the wish table is unavailable; body=%s", w.Code, w.Body.String())
	}
}

func TestGetWishesFromWishlist_CollaboratorsQueryFailure(t *testing.T) {
	// Migrate without the WishlistCollaborator table so
	// database.GetWishlistCollaboratorsFromWishlist fails at the last step.
	setupControllersDB(t, &models.User{}, &models.Wishlist{}, &models.WishlistMembership{}, &models.GroupMembership{}, &models.Group{}, &models.Wish{})
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("GET", "/api/wishes?wishlist="+wishlist.ID.String(), "")
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	GetWishesFromWishlist(ctx)
	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the wishlist_collaborators table is unavailable; body=%s", w.Code, w.Body.String())
	}
}

// --- RegisterWish ---

func TestRegisterWish_Success(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	w, ctx := wishCtx("POST", "/api/wishes?wishlist="+wishlist.ID.String(), `{"name":"A nice gift","note":"","url":""}`)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterWish(ctx)

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
}

func TestRegisterWish_CollaboratorSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	collaborator := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	addWishCollaborator(t, wishlist.ID, collaborator.ID)

	w, ctx := wishCtx("POST", "/api/wishes?wishlist="+wishlist.ID.String(), `{"name":"Collab gift"}`)
	ctx.Request.Header.Set("Authorization", authHeader(t, collaborator.ID, false))
	RegisterWish(ctx)

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
}

func TestRegisterWish_MissingWishlistQuery(t *testing.T) {
	setupControllersDB(t)
	w, ctx := wishCtx("POST", "/api/wishes", `{"name":"A nice gift"}`)
	RegisterWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 without a wishlist query param", w.Code)
	}
}

func TestRegisterWish_BadJSON(t *testing.T) {
	setupControllersDB(t)
	w, ctx := wishCtx("POST", "/api/wishes?wishlist="+uuid.NewString(), `{not-json`)
	RegisterWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for malformed JSON", w.Code)
	}
}

func TestRegisterWish_InvalidWishlistID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	w, ctx := wishCtx("POST", "/api/wishes?wishlist=not-a-uuid", `{"name":"A nice gift"}`)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed wishlist id", w.Code)
	}
}

func TestRegisterWish_NotOwnerOrCollaborator(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	stranger := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	w, ctx := wishCtx("POST", "/api/wishes?wishlist="+wishlist.ID.String(), `{"name":"A nice gift"}`)
	ctx.Request.Header.Set("Authorization", authHeader(t, stranger.ID, false))
	RegisterWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a non-owner/collaborator", w.Code)
	}
}

func TestRegisterWish_NameTooShort(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	w, ctx := wishCtx("POST", "/api/wishes?wishlist="+wishlist.ID.String(), `{"name":"Hi"}`)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a too-short name", w.Code)
	}
}

func TestRegisterWish_DuplicateName(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	existing := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("POST", "/api/wishes?wishlist="+wishlist.ID.String(), `{"name":"`+existing.Name+`"}`)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a duplicate name; body=%s", w.Code, w.Body.String())
	}
}

func TestRegisterWish_InvalidURL(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	w, ctx := wishCtx("POST", "/api/wishes?wishlist="+wishlist.ID.String(), `{"name":"A nice gift","url":"http://"}`)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for an invalid URL; body=%s", w.Code, w.Body.String())
	}
}

func TestRegisterWish_CollaboratorCheckDBFailure(t *testing.T) {
	// Migrate without the WishlistCollaborator table so
	// database.VerifyWishlistCollaboratorToWishlist fails immediately.
	setupControllersDB(t, &models.User{}, &models.Wishlist{})
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	w, ctx := wishCtx("POST", "/api/wishes?wishlist="+wishlist.ID.String(), `{"name":"A nice gift"}`)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterWish(ctx)
	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the wishlist_collaborators table is unavailable; body=%s", w.Code, w.Body.String())
	}
}

func TestRegisterWish_UniqueNameCheckDBFailure(t *testing.T) {
	// Migrate without the Wish table so database.VerifyUniqueWishNameInWishlist
	// fails after every earlier check/validation has passed.
	setupControllersDB(t, &models.User{}, &models.Wishlist{}, &models.WishlistCollaborator{})
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	w, ctx := wishCtx("POST", "/api/wishes?wishlist="+wishlist.ID.String(), `{"name":"A nice gift"}`)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterWish(ctx)
	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the wish table is unavailable; body=%s", w.Code, w.Body.String())
	}
}

func TestRegisterWish_InvalidImageData(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	w, ctx := wishCtx("POST", "/api/wishes?wishlist="+wishlist.ID.String(), `{"name":"A nice gift","image_data":"not-valid-base64-image-data"}`)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterWish(ctx)
	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 for invalid image data; body=%s", w.Code, w.Body.String())
	}
}

// --- DeleteWish ---

func TestDeleteWish_Success(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("DELETE", "/api/wishes/"+wish.ID.String(), "")
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	DeleteWish(ctx)

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteWish_InvalidID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	w, ctx := wishCtx("DELETE", "/api/wishes/not-a-uuid", "")
	ctx.Params = gin.Params{{Key: "wish_id", Value: "not-a-uuid"}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	DeleteWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed wish id", w.Code)
	}
}

func TestDeleteWish_NotOwner(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	stranger := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("DELETE", "/api/wishes/"+wish.ID.String(), "")
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, stranger.ID, false))
	DeleteWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a non-owner/collaborator", w.Code)
	}
}

// --- RegisterWishClaim / RemoveWishClaim ---

// claimSetup builds a claimable wishlist owned by owner, containing one wish
// (also owned by owner), with claimant made a member via a group.
func claimSetup(t *testing.T) (owner, claimant models.User, wishlist models.Wishlist, wish models.Wish) {
	t.Helper()
	owner = createTestUser(t)
	claimant = createTestUser(t)
	wishlist = createTestWishlist(t, owner.ID)
	wishlist = makeClaimable(t, wishlist)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, claimant.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)
	wish = createTestWish(t, owner.ID, wishlist.ID)
	return
}

func TestRegisterWishClaim_Success(t *testing.T) {
	setupControllersDB(t)
	_, claimant, _, wish := claimSetup(t)

	w, ctx := wishCtx("POST", "/api/wishes/"+wish.ID.String()+"/claim", `{}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, claimant.ID, false))
	RegisterWishClaim(ctx)

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
}

func TestRegisterWishClaim_SuccessWithWishlistID(t *testing.T) {
	setupControllersDB(t)
	_, claimant, wishlist, wish := claimSetup(t)

	w, ctx := wishCtx("POST", "/api/wishes/"+wish.ID.String()+"/claim", `{"wishlist_id":"`+wishlist.ID.String()+`"}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, claimant.ID, false))
	RegisterWishClaim(ctx)

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	body := parseWishBody(t, w)
	if _, ok := body["wishes"]; !ok {
		t.Error("expected the wishes list to be returned when wishlist_id is given")
	}
}

func TestRegisterWishClaim_BadJSON(t *testing.T) {
	setupControllersDB(t)
	w, ctx := wishCtx("POST", "/api/wishes/"+uuid.NewString()+"/claim", `{not-json`)
	RegisterWishClaim(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for malformed JSON", w.Code)
	}
}

func TestRegisterWishClaim_InvalidWishID(t *testing.T) {
	setupControllersDB(t)
	claimant := createTestUser(t)
	w, ctx := wishCtx("POST", "/api/wishes/not-a-uuid/claim", `{}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: "not-a-uuid"}}
	ctx.Request.Header.Set("Authorization", authHeader(t, claimant.ID, false))
	RegisterWishClaim(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed wish id", w.Code)
	}
}

func TestRegisterWishClaim_NotClaimable(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	claimant := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID) // Claimable defaults to false
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, claimant.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("POST", "/api/wishes/"+wish.ID.String()+"/claim", `{}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, claimant.ID, false))
	RegisterWishClaim(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a non-claimable wishlist", w.Code)
	}
}

func TestRegisterWishClaim_NotMember(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	stranger := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wishlist = makeClaimable(t, wishlist)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("POST", "/api/wishes/"+wish.ID.String()+"/claim", `{}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, stranger.ID, false))
	RegisterWishClaim(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a non-member", w.Code)
	}
}

func TestRegisterWishClaim_CollaboratorForbidden(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	collaborator := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wishlist = makeClaimable(t, wishlist)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, collaborator.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)
	addWishCollaborator(t, wishlist.ID, collaborator.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("POST", "/api/wishes/"+wish.ID.String()+"/claim", `{}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, collaborator.ID, false))
	RegisterWishClaim(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a collaborator trying to claim", w.Code)
	}
}

func TestRegisterWishClaim_OwnWish(t *testing.T) {
	setupControllersDB(t)
	owner, _, _, wish := claimSetup(t)

	w, ctx := wishCtx("POST", "/api/wishes/"+wish.ID.String()+"/claim", `{}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterWishClaim(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 when claiming your own wish", w.Code)
	}
}

func TestRegisterWishClaim_AlreadyClaimed(t *testing.T) {
	setupControllersDB(t)
	owner, claimant, wishlist, wish := claimSetup(t)

	secondClaimant := createTestUser(t)
	group2 := createTestGroup(t, owner.ID)
	addGroupMembership(t, group2.ID, secondClaimant.ID)
	addWishlistMembership(t, wishlist.ID, group2.ID)

	claim := models.WishClaim{UserID: claimant.ID, WishID: wish.ID, Enabled: true}
	claim.ID = uuid.New()
	if _, err := database.CreateWishClaimInDB(claim); err != nil {
		t.Fatalf("failed to seed claim: %v", err)
	}

	w, ctx := wishCtx("POST", "/api/wishes/"+wish.ID.String()+"/claim", `{}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, secondClaimant.ID, false))
	RegisterWishClaim(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for an already-claimed wish", w.Code)
	}
}

func TestRemoveWishClaim_Success(t *testing.T) {
	setupControllersDB(t)
	_, claimant, _, wish := claimSetup(t)

	claim := models.WishClaim{UserID: claimant.ID, WishID: wish.ID, Enabled: true}
	claim.ID = uuid.New()
	if _, err := database.CreateWishClaimInDB(claim); err != nil {
		t.Fatalf("failed to seed claim: %v", err)
	}

	w, ctx := wishCtx("POST", "/api/wishes/"+wish.ID.String()+"/unclaim", `{}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, claimant.ID, false))
	RemoveWishClaim(ctx)
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestRemoveWishClaim_NotClaimed(t *testing.T) {
	setupControllersDB(t)
	_, claimant, _, wish := claimSetup(t)

	w, ctx := wishCtx("POST", "/api/wishes/"+wish.ID.String()+"/unclaim", `{}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, claimant.ID, false))
	RemoveWishClaim(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 when nothing was claimed", w.Code)
	}
}

func TestRemoveWishClaim_CollaboratorForbidden(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	collaborator := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wishlist = makeClaimable(t, wishlist)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, collaborator.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)
	addWishCollaborator(t, wishlist.ID, collaborator.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("POST", "/api/wishes/"+wish.ID.String()+"/unclaim", `{}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, collaborator.ID, false))
	RemoveWishClaim(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a collaborator trying to unclaim", w.Code)
	}
}

func TestRemoveWishClaim_BadJSON(t *testing.T) {
	setupControllersDB(t)
	w, ctx := wishCtx("POST", "/api/wishes/"+uuid.NewString()+"/unclaim", `{not-json`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: uuid.NewString()}}
	RemoveWishClaim(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for malformed JSON", w.Code)
	}
}

func TestRemoveWishClaim_MissingAuth(t *testing.T) {
	setupControllersDB(t)
	w, ctx := wishCtx("POST", "/api/wishes/"+uuid.NewString()+"/unclaim", `{}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: uuid.NewString()}}
	RemoveWishClaim(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 without auth", w.Code)
	}
}

func TestRemoveWishClaim_InvalidWishID(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	w, ctx := wishCtx("POST", "/api/wishes/not-a-uuid/unclaim", `{}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: "not-a-uuid"}}
	ctx.Request.Header.Set("Authorization", authHeader(t, user.ID, false))
	RemoveWishClaim(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed wish id", w.Code)
	}
}

func TestRemoveWishClaim_OwnershipCheckDBFailure(t *testing.T) {
	// Migrate without the WishClaim table so
	// database.VerifyUserOwnershipToWishClaimByWish fails, after every
	// earlier ownership/membership check has passed.
	setupControllersDB(t, &models.User{}, &models.Wishlist{}, &models.WishlistMembership{}, &models.WishlistCollaborator{}, &models.Group{}, &models.GroupMembership{}, &models.Wish{})
	owner := createTestUser(t)
	claimant := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wishlist = makeClaimable(t, wishlist)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, claimant.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("POST", "/api/wishes/"+wish.ID.String()+"/unclaim", `{}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, claimant.ID, false))
	RemoveWishClaim(ctx)
	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the wish_claims table is unavailable; body=%s", w.Code, w.Body.String())
	}
}

// --- APIUpdateWish ---

func TestAPIUpdateWish_Success(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("PUT", "/api/wishes/"+wish.ID.String(), `{"name":"Updated name","note":"updated note"}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateWish(ctx)

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
}

func TestAPIUpdateWish_CollaboratorCheckDBFailure(t *testing.T) {
	// Migrate without the WishlistCollaborator table so
	// database.VerifyWishlistCollaboratorToWishlist fails, after the wish
	// lookup has already succeeded.
	setupControllersDB(t, &models.User{}, &models.Wishlist{}, &models.Wish{})
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("PUT", "/api/wishes/"+wish.ID.String(), `{"name":"Updated name"}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateWish(ctx)
	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the wishlist_collaborators table is unavailable; body=%s", w.Code, w.Body.String())
	}
}

func TestAPIUpdateWish_BadJSON(t *testing.T) {
	setupControllersDB(t)
	w, ctx := wishCtx("PUT", "/api/wishes/"+uuid.NewString(), `{not-json`)
	APIUpdateWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for malformed JSON", w.Code)
	}
}

func TestAPIUpdateWish_InvalidID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	w, ctx := wishCtx("PUT", "/api/wishes/not-a-uuid", `{"name":"Updated name"}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: "not-a-uuid"}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed wish id", w.Code)
	}
}

func TestAPIUpdateWish_NotFound(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	w, ctx := wishCtx("PUT", "/api/wishes/"+uuid.NewString(), `{"name":"Updated name"}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: uuid.NewString()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a wish that does not exist", w.Code)
	}
}

func TestAPIUpdateWish_NotOwner(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	stranger := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("PUT", "/api/wishes/"+wish.ID.String(), `{"name":"Updated name"}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, stranger.ID, false))
	APIUpdateWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a non-owner/collaborator", w.Code)
	}
}

func TestAPIUpdateWish_NameTooShort(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("PUT", "/api/wishes/"+wish.ID.String(), `{"name":"Hi"}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a too-short name", w.Code)
	}
}

func TestAPIUpdateWish_DuplicateName(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	other := createTestWish(t, owner.ID, wishlist.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("PUT", "/api/wishes/"+wish.ID.String(), `{"name":"`+other.Name+`"}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a duplicate name; body=%s", w.Code, w.Body.String())
	}
}

func TestAPIUpdateWish_InvalidURL(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("PUT", "/api/wishes/"+wish.ID.String(), `{"name":"`+wish.Name+`","url":"http://"}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for an invalid URL; body=%s", w.Code, w.Body.String())
	}
}

// --- APIGetWish ---

func TestAPIGetWish_Success(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("GET", "/api/wishes/"+wish.ID.String(), "")
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIGetWish(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestAPIGetWish_InvalidID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	w, ctx := wishCtx("GET", "/api/wishes/not-a-uuid", "")
	ctx.Params = gin.Params{{Key: "wish_id", Value: "not-a-uuid"}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIGetWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed wish id", w.Code)
	}
}

func TestAPIGetWish_NotMember(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	stranger := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("GET", "/api/wishes/"+wish.ID.String(), "")
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, stranger.ID, false))
	APIGetWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a non-member", w.Code)
	}
}

func TestAPIGetWish_UnknownWishlist(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	w, ctx := wishCtx("GET", "/api/wishes/"+uuid.NewString(), "")
	ctx.Params = gin.Params{{Key: "wish_id", Value: uuid.NewString()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIGetWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a wish with no matching wishlist; body=%s", w.Code, w.Body.String())
	}
}

func TestAPIGetWish_MembershipCheckDBFailure(t *testing.T) {
	// Migrate without the WishlistMembership/Group/GroupMembership tables so
	// database.VerifyUserMembershipToGroupMembershipToWishlist fails, after
	// GetWishlistIDFromWish and VerifyUserOwnershipToWishlist have already
	// succeeded (the wish exists and the caller isn't its owner).
	setupControllersDB(t, &models.User{}, &models.Wishlist{}, &models.Wish{})
	owner := createTestUser(t)
	stranger := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("GET", "/api/wishes/"+wish.ID.String(), "")
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, stranger.ID, false))
	APIGetWish(ctx)
	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the wishlist_memberships table is unavailable; body=%s", w.Code, w.Body.String())
	}
}

// --- parseRawURLFunction ---

func TestParseRawURLFunction(t *testing.T) {
	cases := []struct {
		name          string
		raw           string
		wantDomain    string
		wantSchemeSet bool
		wantDomainSet bool
	}{
		{"full URL with scheme", "https://example.com/path", "example.com", true, true},
		{"URL without scheme gets https prepended", "example.com/path", "example.com", false, true},
		{"bare word still resolves as a host", "notarealurl", "notarealurl", false, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			domain, scheme, err := parseRawURLFunction(c.raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.wantDomainSet && domain != c.wantDomain {
				t.Errorf("domain = %q, want %q", domain, c.wantDomain)
			}
			if c.wantSchemeSet && scheme == "" {
				t.Error("expected a non-empty scheme")
			}
		})
	}
}

// --- ConvertWishToWishObject branch coverage ---

func TestConvertWishToWishObject_WishlistOwnerClaimPurged(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	other := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wishlist = makeClaimable(t, wishlist)
	// Wish is owned by someone else on owner's wishlist (e.g. added by a collaborator).
	wish := createTestWish(t, other.ID, wishlist.ID)

	claim := models.WishClaim{UserID: other.ID, WishID: wish.ID, Enabled: true}
	claim.ID = uuid.New()
	if _, err := database.CreateWishClaimInDB(claim); err != nil {
		t.Fatalf("failed to seed claim: %v", err)
	}

	wishObject, err := ConvertWishToWishObject(wish, &owner.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wishObject.WishClaim) != 0 {
		t.Errorf("expected claim details purged for the wishlist owner, got %v", wishObject.WishClaim)
	}
}

func TestConvertWishToWishObject_HideClaimersAnonymizes(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	claimant := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wishlist = makeClaimable(t, wishlist)
	wishlist.HideClaimers = boolPtr(true)
	wishlist, err := database.UpdateWishlistInDB(wishlist)
	if err != nil {
		t.Fatalf("failed to set hide_claimers: %v", err)
	}
	wish := createTestWish(t, owner.ID, wishlist.ID)

	claim := models.WishClaim{UserID: claimant.ID, WishID: wish.ID, Enabled: true}
	claim.ID = uuid.New()
	if _, err := database.CreateWishClaimInDB(claim); err != nil {
		t.Fatalf("failed to seed claim: %v", err)
	}

	// A stranger (not owner) still sees the claim exists, but claimer identity
	// should be anonymized when hide_claimers is on.
	stranger := createTestUser(t)
	wishObject, err := ConvertWishToWishObject(wish, &stranger.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wishObject.WishClaim) != 1 {
		t.Fatalf("expected 1 claim, got %v", wishObject.WishClaim)
	}
	if wishObject.WishClaim[0].User.FirstName != "Hidden" {
		t.Errorf("FirstName = %q, want Hidden", wishObject.WishClaim[0].User.FirstName)
	}
}

func TestConvertWishToWishObject_WithCategory(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	category := createTestWishCategory(t, wishlist.ID, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)
	wish.CategoryID = &category.ID
	updated, err := database.UpdateWishInDB(wish)
	if err != nil {
		t.Fatalf("failed to assign category: %v", err)
	}

	wishObject, err := ConvertWishToWishObject(updated, &owner.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wishObject.Category == nil || wishObject.Category.Name != category.Name {
		t.Errorf("Category = %v, want %v", wishObject.Category, category.Name)
	}
}

// TestConvertWishToWishObject_DanglingWishlistIDErrors covers a wish whose
// WishlistID points at a wishlist that no longer exists. This used to panic
// (a nil-pointer dereference on *wishlist.Claimable), since
// database.GetWishlistByWishlistID returns a zero-value Wishlist with a nil
// Claimable and a false "found" bool rather than an error.
func TestConvertWishToWishObject_DanglingWishlistIDErrors(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	wish := models.Wish{Name: "Orphaned wish", Enabled: true, OwnerID: owner.ID, WishlistID: uuid.New()}
	wish.ID = uuid.New()
	if _, err := database.CreateWishInDB(wish); err != nil {
		t.Fatalf("failed to create wish: %v", err)
	}

	if _, err := ConvertWishToWishObject(wish, &owner.ID); err == nil {
		t.Error("expected an error for a wish with a dangling wishlist ID, got nil")
	}
}

func TestConvertWishesToWishObjects_SkipsUnconvertible(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	good := createTestWish(t, owner.ID, wishlist.ID)

	// A wish whose owner no longer exists can't be converted
	// (ConvertWishToWishObject needs to resolve the owner via GetUserInformation
	// first, which errors cleanly for an unknown user) - it should be skipped
	// rather than failing the whole batch.
	broken := models.Wish{Name: "Broken", Enabled: true, OwnerID: uuid.New(), WishlistID: wishlist.ID}
	broken.ID = uuid.New()
	if _, err := database.CreateWishInDB(broken); err != nil {
		t.Fatalf("failed to create broken wish: %v", err)
	}

	objects, err := ConvertWishesToWishObjects([]models.Wish{good, broken}, &owner.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(objects) != 1 || objects[0].Name != good.Name {
		t.Errorf("objects = %v, want exactly the one convertible wish", objects)
	}
}

// --- RegisterWish additional branches ---

func TestRegisterWish_InvalidNameCharacters(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	w, ctx := wishCtx("POST", "/api/wishes?wishlist="+wishlist.ID.String(), `{"name":"<script>bad</script>"}`)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for disallowed characters in the name", w.Code)
	}
}

func TestRegisterWish_InvalidNoteCharacters(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	w, ctx := wishCtx("POST", "/api/wishes?wishlist="+wishlist.ID.String(), `{"name":"A nice gift","note":"bad \"quote\""}`)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for disallowed characters in the note", w.Code)
	}
}

func TestRegisterWish_CategoryNameTooShort(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	w, ctx := wishCtx("POST", "/api/wishes?wishlist="+wishlist.ID.String(), `{"name":"A nice gift","category_name":"A"}`)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a too-short category name; body=%s", w.Code, w.Body.String())
	}
}

func TestRegisterWish_CategoryFromWrongWishlist(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	otherWishlist := createTestWishlist(t, owner.ID)
	category := createTestWishCategory(t, otherWishlist.ID, owner.ID)

	w, ctx := wishCtx("POST", "/api/wishes?wishlist="+wishlist.ID.String(), `{"name":"A nice gift","category_id":"`+category.ID.String()+`"}`)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a category belonging to a different wishlist", w.Code)
	}
}

func TestRegisterWish_WithExistingCategoryByID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	category := createTestWishCategory(t, wishlist.ID, owner.ID)

	w, ctx := wishCtx("POST", "/api/wishes?wishlist="+wishlist.ID.String(), `{"name":"A nice gift","category_id":"`+category.ID.String()+`"}`)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterWish(ctx)
	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
}

// --- DeleteWish additional branches ---

func TestDeleteWish_CollaboratorSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	collaborator := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	addWishCollaborator(t, wishlist.ID, collaborator.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("DELETE", "/api/wishes/"+wish.ID.String(), "")
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, collaborator.ID, false))
	DeleteWish(ctx)
	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteWish_UnknownID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	w, ctx := wishCtx("DELETE", "/api/wishes/"+uuid.NewString(), "")
	ctx.Params = gin.Params{{Key: "wish_id", Value: uuid.NewString()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	DeleteWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a nonexistent wish; body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteWish_CleansUpEmptyCategory(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	category := createTestWishCategory(t, wishlist.ID, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)
	wish.CategoryID = &category.ID
	wish, err := database.UpdateWishInDB(wish)
	if err != nil {
		t.Fatalf("failed to assign category: %v", err)
	}

	w, ctx := wishCtx("DELETE", "/api/wishes/"+wish.ID.String(), "")
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	DeleteWish(ctx)
	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}

	remaining, err := database.GetWishCategoryByID(category.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if remaining != nil {
		t.Errorf("expected the now-empty category to be cleaned up, got %v", remaining)
	}
}

func TestDeleteWish_ClaimableWithClaimNotifiesWithoutFailing(t *testing.T) {
	setupControllersDB(t)
	owner, claimant, _, wish := claimSetup(t)

	claim := models.WishClaim{UserID: claimant.ID, WishID: wish.ID, Enabled: true}
	claim.ID = uuid.New()
	if _, err := database.CreateWishClaimInDB(claim); err != nil {
		t.Fatalf("failed to seed claim: %v", err)
	}

	w, ctx := wishCtx("DELETE", "/api/wishes/"+wish.ID.String(), "")
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	DeleteWish(ctx)
	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteWish_CollaboratorCheckDBFailure(t *testing.T) {
	// Migrate without the WishlistCollaborator table. ConvertWishToWishObject
	// itself queries that table (to build the collaborators list) and fails
	// first, before the handler's own VerifyWishlistCollaboratorToWishlist
	// call would - either way it exercises a DB-failure 500 branch.
	setupControllersDB(t, &models.User{}, &models.Wishlist{}, &models.Wish{}, &models.WishClaim{}, &models.WishCategory{})
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("DELETE", "/api/wishes/"+wish.ID.String(), "")
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	DeleteWish(ctx)
	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the wishlist_collaborators table is unavailable; body=%s", w.Code, w.Body.String())
	}
}

// --- RegisterWishClaim / RemoveWishClaim additional branches ---

func TestRegisterWishClaim_WishNotFound(t *testing.T) {
	setupControllersDB(t)
	claimant := createTestUser(t)

	w, ctx := wishCtx("POST", "/api/wishes/"+uuid.NewString()+"/claim", `{}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: uuid.NewString()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, claimant.ID, false))
	RegisterWishClaim(ctx)
	if w.Code != 404 {
		t.Fatalf("status = %d, want 404 for a wish that doesn't exist", w.Code)
	}
}

func TestRegisterWishClaim_OwnerCanClaimOthersWish(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	contributor := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wishlist = makeClaimable(t, wishlist)
	// A wish added by someone else (e.g. via a group) on the owner's own wishlist.
	wish := createTestWish(t, contributor.ID, wishlist.ID)

	w, ctx := wishCtx("POST", "/api/wishes/"+wish.ID.String()+"/claim", `{}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterWishClaim(ctx)
	if w.Code != 201 {
		t.Fatalf("status = %d, want 201 (wishlist owner claiming another user's wish); body=%s", w.Code, w.Body.String())
	}
}

func TestRegisterWishClaim_DanglingWishlistID(t *testing.T) {
	// A wish whose WishlistID points at no real wishlist row (GetWishlistIDFromWish,
	// which only needs the Wish table, succeeds; GetWishlistByWishlistID then
	// reports "not found").
	setupControllersDB(t, &models.User{}, &models.Wish{})
	owner := createTestUser(t)
	claimant := createTestUser(t)
	wish := models.Wish{Name: "Gift", Enabled: true, OwnerID: owner.ID, WishlistID: uuid.New()}
	wish.ID = uuid.New()
	created, err := database.CreateWishInDB(wish)
	if err != nil {
		t.Fatalf("failed to create wish: %v", err)
	}

	w, ctx := wishCtx("POST", "/api/wishes/"+created.ID.String()+"/claim", `{}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: created.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, claimant.ID, false))
	RegisterWishClaim(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a wish with a dangling wishlist reference; body=%s", w.Code, w.Body.String())
	}
}

func TestRegisterWishClaim_ClaimStatusCheckDBFailure(t *testing.T) {
	// Migrate without the WishClaim table so database.VerifyWishIsClaimed
	// fails, after every earlier ownership/membership check has passed.
	setupControllersDB(t, &models.User{}, &models.Wishlist{}, &models.WishlistMembership{}, &models.WishlistCollaborator{}, &models.Group{}, &models.GroupMembership{}, &models.Wish{})
	owner := createTestUser(t)
	claimant := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wishlist = makeClaimable(t, wishlist)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, claimant.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("POST", "/api/wishes/"+wish.ID.String()+"/claim", `{}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, claimant.ID, false))
	RegisterWishClaim(ctx)
	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the wish_claims table is unavailable; body=%s", w.Code, w.Body.String())
	}
}

func TestRemoveWishClaim_WishNotFound(t *testing.T) {
	setupControllersDB(t)
	claimant := createTestUser(t)

	w, ctx := wishCtx("POST", "/api/wishes/"+uuid.NewString()+"/unclaim", `{}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: uuid.NewString()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, claimant.ID, false))
	RemoveWishClaim(ctx)
	if w.Code != 404 {
		t.Fatalf("status = %d, want 404 for a wish that doesn't exist", w.Code)
	}
}

func TestRemoveWishClaim_NotClaimableWishlist(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	member := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID) // Claimable defaults to false
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, member.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("POST", "/api/wishes/"+wish.ID.String()+"/unclaim", `{}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, member.ID, false))
	RemoveWishClaim(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a non-claimable wishlist", w.Code)
	}
}

func TestRemoveWishClaim_SuccessWithWishlistID(t *testing.T) {
	setupControllersDB(t)
	_, claimant, wishlist, wish := claimSetup(t)

	claim := models.WishClaim{UserID: claimant.ID, WishID: wish.ID, Enabled: true}
	claim.ID = uuid.New()
	if _, err := database.CreateWishClaimInDB(claim); err != nil {
		t.Fatalf("failed to seed claim: %v", err)
	}

	w, ctx := wishCtx("POST", "/api/wishes/"+wish.ID.String()+"/unclaim", `{"wishlist_id":"`+wishlist.ID.String()+`"}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, claimant.ID, false))
	RemoveWishClaim(ctx)
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	body := parseWishBody(t, w)
	if _, ok := body["wishes"]; !ok {
		t.Error("expected the wishes list to be returned when wishlist_id is given")
	}
}

// --- APIUpdateWish additional branches ---

func TestAPIUpdateWish_InvalidNoteCharacters(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("PUT", "/api/wishes/"+wish.ID.String(), `{"name":"`+wish.Name+`","note":"bad \"quote\""}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for disallowed characters in the note", w.Code)
	}
}

func TestAPIUpdateWish_PriceChange(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("PUT", "/api/wishes/"+wish.ID.String(), `{"name":"`+wish.Name+`","price":42.5}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateWish(ctx)
	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}

	updated, err := database.GetWishByWishID(wish.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Price == nil || *updated.Price != 42.5 {
		t.Errorf("Price = %v, want 42.5", updated.Price)
	}
}

func TestAPIUpdateWish_ClearURL(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)
	wish.URL = "https://example.com"
	wish, err := database.UpdateWishInDB(wish)
	if err != nil {
		t.Fatalf("failed to seed a URL: %v", err)
	}

	w, ctx := wishCtx("PUT", "/api/wishes/"+wish.ID.String(), `{"name":"`+wish.Name+`","url":""}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateWish(ctx)
	if w.Code != 201 {
		t.Fatalf("status = %d, want 201 (clearing the URL should not require it to parse); body=%s", w.Code, w.Body.String())
	}
}

func TestAPIUpdateWish_CollaboratorSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	collaborator := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	addWishCollaborator(t, wishlist.ID, collaborator.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("PUT", "/api/wishes/"+wish.ID.String(), `{"name":"Updated by collaborator"}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, collaborator.ID, false))
	APIUpdateWish(ctx)
	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
}

func TestAPIUpdateWish_CategoryNameTooShort(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w, ctx := wishCtx("PUT", "/api/wishes/"+wish.ID.String(), `{"name":"`+wish.Name+`","category_name":"A"}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateWish(ctx)
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a too-short category name", w.Code)
	}
}

func TestAPIUpdateWish_MovingCategoryCleansUpEmptyOne(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	oldCategory := createTestWishCategory(t, wishlist.ID, owner.ID)
	newCategory := createTestWishCategory(t, wishlist.ID, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)
	wish.CategoryID = &oldCategory.ID
	wish, err := database.UpdateWishInDB(wish)
	if err != nil {
		t.Fatalf("failed to assign the original category: %v", err)
	}

	w, ctx := wishCtx("PUT", "/api/wishes/"+wish.ID.String(), `{"name":"`+wish.Name+`","category_id":"`+newCategory.ID.String()+`"}`)
	ctx.Params = gin.Params{{Key: "wish_id", Value: wish.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateWish(ctx)
	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}

	remaining, err := database.GetWishCategoryByID(oldCategory.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if remaining != nil {
		t.Errorf("expected the now-empty old category to be cleaned up, got %v", remaining)
	}
}

func TestWishHandlersDatabaseErrors(t *testing.T) {
	wishParam := gin.Params{{Key: "wish_id", Value: "00000000-0000-0000-0000-00000000000a"}}
	runDatabaseErrorCases(t, []dbErrorCase{
		{name: "GetWishesFromWishlist", handler: GetWishesFromWishlist, method: "GET", path: "/api/auth/wishes?wishlist=00000000-0000-0000-0000-00000000000a"},
		{name: "RegisterWish", handler: RegisterWish, method: "POST", path: "/api/auth/wishes?wishlist=00000000-0000-0000-0000-00000000000a", body: `{"name":"Wish","note":"Note"}`},
		{name: "DeleteWish", handler: DeleteWish, method: "DELETE", path: "/api/auth/wishes/00000000-0000-0000-0000-00000000000a", params: wishParam},
		{name: "RegisterWishClaim", handler: RegisterWishClaim, method: "POST", path: "/api/auth/wishes/00000000-0000-0000-0000-00000000000a/claim", body: `{"wishlist_id":"00000000-0000-0000-0000-00000000000a"}`, params: wishParam},
		{name: "RemoveWishClaim", handler: RemoveWishClaim, method: "POST", path: "/api/auth/wishes/00000000-0000-0000-0000-00000000000a/unclaim", body: `{"wishlist_id":"00000000-0000-0000-0000-00000000000a"}`, params: wishParam},
		{name: "APIUpdateWish", handler: APIUpdateWish, method: "POST", path: "/api/auth/wishes/00000000-0000-0000-0000-00000000000a", body: `{"name":"Wish","note":"Note"}`, params: wishParam},
		{name: "APIGetWish", handler: APIGetWish, method: "GET", path: "/api/auth/wishes/00000000-0000-0000-0000-00000000000a", params: wishParam},
	})
}

func TestWishHandlersRequireAuth(t *testing.T) {
	wishParam := gin.Params{{Key: "wish_id", Value: "00000000-0000-0000-0000-00000000000a"}}
	runUnauthenticatedCases(t, []dbErrorCase{
		{name: "RegisterWish", handler: RegisterWish, method: "POST", path: "/api/auth/wishes?wishlist=00000000-0000-0000-0000-00000000000a", body: `{"name":"Wish"}`},
		{name: "DeleteWish", handler: DeleteWish, method: "DELETE", path: "/api/auth/wishes/00000000-0000-0000-0000-00000000000a", params: wishParam},
		{name: "RegisterWishClaim", handler: RegisterWishClaim, method: "POST", path: "/api/auth/wishes/00000000-0000-0000-0000-00000000000a/claim", body: `{"wishlist_id":"00000000-0000-0000-0000-00000000000a"}`, params: wishParam},
		{name: "APIUpdateWish", handler: APIUpdateWish, method: "POST", path: "/api/auth/wishes/00000000-0000-0000-0000-00000000000a", body: `{"name":"Wish"}`, params: wishParam},
		{name: "APIGetWish", handler: APIGetWish, method: "GET", path: "/api/auth/wishes/00000000-0000-0000-0000-00000000000a", params: wishParam},
	})
}

// --- Deeper branches: injected DB failures, sorting, images ---

// wishTestReq is one handler call, built by a fixture before any failure is
// injected so the fixture's own inserts aren't affected.
type wishTestReq struct {
	handler gin.HandlerFunc
	method  string
	path    string
	body    string
	params  gin.Params
	userID  uuid.UUID
}

// wishTestSend runs req as req.userID and returns the recorder.
func wishTestSend(t *testing.T, req wishTestReq, header string) *httptest.ResponseRecorder {
	t.Helper()
	path := req.path
	if path == "" {
		path = "/"
	}
	w, ctx := wishCtx(req.method, path, req.body)
	ctx.Params = req.params
	ctx.Request.Header.Set("Authorization", header)
	req.handler(ctx)
	return w
}

// wishTestFailCase injects a failure into one GORM operation and expects the
// handler to stop at the branch that reports it.
type wishTestFailCase struct {
	name    string
	fixture func(t *testing.T) wishTestReq
	op      string
	table   string
	skip    int
	want    int
	wantErr string
}

func wishTestRunFailures(t *testing.T, cases []wishTestFailCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			req := c.fixture(t)
			header := authHeader(t, req.userID, false)
			failDBOperation(t, c.op, c.table, c.skip)

			w := wishTestSend(t, req, header)
			if w.Code != c.want {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, c.want, w.Body.String())
			}
			if got := parseWishBody(t, w)["error"]; got != c.wantErr {
				t.Errorf("error = %v, want %q", got, c.wantErr)
			}
		})
	}
}

// wishTestImageDir points wishImageDir at a fresh temp directory for the
// test, restoring the original afterwards.
func wishTestImageDir(t *testing.T) string {
	t.Helper()
	orig := wishImageDir
	t.Cleanup(func() { wishImageDir = orig })
	wishImageDir = filepath.Join(t.TempDir(), "wishes")
	return wishImageDir
}

// wishTestBlockImageDelete makes deleting wishID's image fail: a non-empty
// directory sits where the full-size image file would be, so os.Remove
// returns ENOTEMPTY rather than "not exist".
func wishTestBlockImageDelete(t *testing.T, wishID uuid.UUID) {
	t.Helper()
	dir := wishTestImageDir(t)
	blocker := imageFilePath(dir, wishID, false)
	if err := os.MkdirAll(blocker, 0755); err != nil {
		t.Fatalf("failed to create blocking dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(blocker, "keep"), []byte("x"), 0644); err != nil {
		t.Fatalf("failed to populate blocking dir: %v", err)
	}
}

// wishTestSeedClaim inserts an enabled claim by userID on wishID.
func wishTestSeedClaim(t *testing.T, wishID, userID uuid.UUID) {
	t.Helper()
	claim := models.WishClaim{UserID: userID, WishID: wishID, Enabled: true}
	claim.ID = uuid.New()
	if _, err := database.CreateWishClaimInDB(claim); err != nil {
		t.Fatalf("failed to seed claim: %v", err)
	}
}

// wishTestWishNames returns the "name" of each entry in body["wishes"].
func wishTestWishNames(t *testing.T, body map[string]interface{}) []string {
	t.Helper()
	list, ok := body["wishes"].([]interface{})
	if !ok {
		t.Fatalf("wishes = %v, want a list", body["wishes"])
	}
	names := []string{}
	for _, entry := range list {
		names = append(names, entry.(map[string]interface{})["name"].(string))
	}
	return names
}

// wishTestCreateNamedWish inserts a wish with an explicit name and creation
// time, so tests can assert the handlers' newest-first ordering.
func wishTestCreateNamedWish(t *testing.T, ownerID, wishlistID uuid.UUID, name string, created time.Time) models.Wish {
	t.Helper()
	wish := models.Wish{Name: name, Enabled: true, OwnerID: ownerID, WishlistID: wishlistID}
	wish.ID = uuid.New()
	wish.CreatedAt = created
	wish.UpdatedAt = created
	createdWish, err := database.CreateWishInDB(wish)
	if err != nil {
		t.Fatalf("failed to create wish: %v", err)
	}
	return createdWish
}

func wishTestWishEnabled(t *testing.T, wishID uuid.UUID) bool {
	t.Helper()
	wish, err := database.GetWishByWishID(wishID)
	if err != nil {
		t.Fatalf("failed to look up wish: %v", err)
	}
	return wish != nil
}

func TestGetWishesFromWishlist_InjectedFailures(t *testing.T) {
	fixture := func(t *testing.T) wishTestReq {
		owner := createTestUser(t)
		wishlist := createTestWishlist(t, owner.ID)
		return wishTestReq{handler: GetWishesFromWishlist, method: "GET", path: "/api/wishes?wishlist=" + wishlist.ID.String(), userID: owner.ID}
	}
	wishTestRunFailures(t, []wishTestFailCase{
		{name: "membership check", fixture: fixture, op: "query", table: "wishlist_memberships", want: 500, wantErr: "Failed to verify membership of group."},
		// First wishlists query is the ownership check, the second the owner lookup.
		{name: "owner lookup", fixture: fixture, op: "query", table: "wishlists", skip: 1, want: 500, wantErr: "Failed to get wishlist owner."},
	})
}

func TestGetWishesFromWishlist_CollaboratorsAndNewestFirst(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	collaborator := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	addWishCollaborator(t, wishlist.ID, collaborator.ID)
	now := time.Now()
	wishTestCreateNamedWish(t, owner.ID, wishlist.ID, "Older wish", now.Add(-time.Hour))
	wishTestCreateNamedWish(t, owner.ID, wishlist.ID, "Newer wish", now)

	w := wishTestSend(t, wishTestReq{handler: GetWishesFromWishlist, method: "GET", path: "/api/wishes?wishlist=" + wishlist.ID.String()}, authHeader(t, owner.ID, false))
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	body := parseWishBody(t, w)
	collabs, _ := body["collaborators"].([]interface{})
	if len(collabs) != 1 || collabs[0] != collaborator.ID.String() {
		t.Errorf("collaborators = %v, want [%s]", body["collaborators"], collaborator.ID)
	}
	if names := wishTestWishNames(t, body); len(names) != 2 || names[0] != "Newer wish" {
		t.Errorf("wishes = %v, want newest first", names)
	}
}

func TestConvertWishToWishObject_WishlistQueryFails(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)
	failDBOperation(t, "query", "wishlists", 0)

	if _, err := ConvertWishToWishObject(wish, nil); err == nil {
		t.Fatal("expected an error when the wishlist lookup fails")
	}
}

func TestConvertWishToWishObject_WishlistOwnerMissing(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlistOwner := createTestUser(t)
	wishlist := createTestWishlist(t, wishlistOwner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)
	// The wish owner is the first users lookup, the wishlist owner the second.
	failDBOperation(t, "query", "users", 1)

	if _, err := ConvertWishToWishObject(wish, nil); err == nil {
		t.Fatal("expected an error when the wishlist owner lookup fails")
	}
}

func TestConvertWishToWishObject_ImageStatErrorTreatedAsNoImage(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)

	// A regular file where the image directory should be makes os.Stat fail
	// with ENOTDIR, which isn't "not exist".
	orig := wishImageDir
	t.Cleanup(func() { wishImageDir = orig })
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	wishImageDir = file

	object, err := ConvertWishToWishObject(wish, nil)
	if err != nil {
		t.Fatalf("ConvertWishToWishObject: %v", err)
	}
	if object.Image {
		t.Error("Image = true, want false when the image can't be checked")
	}
}

func TestConvertWishToWishObject_CategoryLookupFailureSkipsCategory(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	category := createTestWishCategory(t, wishlist.ID, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)
	wish.CategoryID = &category.ID
	failDBOperation(t, "query", "wish_categories", 0)

	object, err := ConvertWishToWishObject(wish, nil)
	if err != nil {
		t.Fatalf("ConvertWishToWishObject: %v", err)
	}
	if object.Category != nil {
		t.Errorf("Category = %+v, want nil when the lookup fails", object.Category)
	}
}

func TestRegisterWish_InjectedFailures(t *testing.T) {
	fixture := func(body string) func(t *testing.T) wishTestReq {
		return func(t *testing.T) wishTestReq {
			owner := createTestUser(t)
			wishlist := createTestWishlist(t, owner.ID)
			return wishTestReq{handler: RegisterWish, method: "POST", path: "/api/wishes?wishlist=" + wishlist.ID.String(), body: body, userID: owner.ID}
		}
	}
	plain := fixture(`{"name":"A nice gift"}`)
	wishTestRunFailures(t, []wishTestFailCase{
		{name: "ownership check", fixture: plain, op: "query", table: "wishlists", want: 500, wantErr: "Failed to verify ownership of wishlist."},
		{name: "category resolution", fixture: fixture(`{"name":"A nice gift","category_name":"Books"}`), op: "query", table: "wish_categories", want: 500, wantErr: "Failed to resolve wish category."},
		{name: "create", fixture: plain, op: "create", table: "wishes", want: 500, wantErr: "Failed to create wishlist."},
		// The first wishes query is the unique-name check.
		{name: "list wishes", fixture: plain, op: "query", table: "wishes", skip: 1, want: 500, wantErr: "Failed to get wishes from database."},
	})
}

func TestRegisterWish_InvalidURLCharacters(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	w := wishTestSend(t, wishTestReq{handler: RegisterWish, method: "POST", path: "/api/wishes?wishlist=" + wishlist.ID.String(), body: `{"name":"A nice gift","url":"https://example.com/<b>"}`}, authHeader(t, owner.ID, false))
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	if _, wishes, _ := database.GetWishesFromWishlist(wishlist.ID); len(wishes) != 0 {
		t.Errorf("expected no wish to be created, got %d", len(wishes))
	}
}

func TestRegisterWish_ReturnsWishesNewestFirst(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wishTestCreateNamedWish(t, owner.ID, wishlist.ID, "Existing wish", time.Now().Add(-time.Hour))

	w := wishTestSend(t, wishTestReq{handler: RegisterWish, method: "POST", path: "/api/wishes?wishlist=" + wishlist.ID.String(), body: `{"name":"Brand new wish"}`}, authHeader(t, owner.ID, false))
	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if names := wishTestWishNames(t, parseWishBody(t, w)); len(names) != 2 || names[0] != "Brand new wish" {
		t.Errorf("wishes = %v, want the new wish first", names)
	}
}

func TestDeleteWish_InjectedFailures(t *testing.T) {
	fixture := func(t *testing.T) wishTestReq {
		owner := createTestUser(t)
		wishlist := createTestWishlist(t, owner.ID)
		wish := createTestWish(t, owner.ID, wishlist.ID)
		return wishTestReq{handler: DeleteWish, method: "DELETE", path: "/api/wishes/" + wish.ID.String(), params: gin.Params{{Key: "wish_id", Value: wish.ID.String()}}, userID: owner.ID}
	}
	wishTestRunFailures(t, []wishTestFailCase{
		// ConvertWishToWishObject makes the first collaborator and wishlist
		// queries; the handler's own checks are the second.
		{name: "collaborator check", fixture: fixture, op: "query", table: "wishlist_collaborators", skip: 1, want: 500, wantErr: "Failed to verify wishlist collaborator status."},
		{name: "ownership check", fixture: fixture, op: "query", table: "wishlists", skip: 1, want: 500, wantErr: "Failed to verify ownership of wishlist."},
		{name: "disable wish", fixture: fixture, op: "update", table: "wishes", want: 500, wantErr: "Failed to delete wish."},
		{name: "list wishes", fixture: fixture, op: "query", table: "wishes", skip: 1, want: 500, wantErr: "Failed to get wishes from database."},
	})
}

func TestDeleteWish_ImageDeleteFailureIsNotFatal(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)
	wishTestBlockImageDelete(t, wish.ID)

	w := wishTestSend(t, wishTestReq{handler: DeleteWish, method: "DELETE", params: gin.Params{{Key: "wish_id", Value: wish.ID.String()}}}, authHeader(t, owner.ID, false))
	if w.Code != 201 {
		t.Fatalf("status = %d, want 201 even though the image couldn't be removed; body=%s", w.Code, w.Body.String())
	}
	if wishTestWishEnabled(t, wish.ID) {
		t.Error("expected the wish to be disabled")
	}
}

func TestDeleteWish_ReturnsRemainingWishesNewestFirst(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	now := time.Now()
	wishTestCreateNamedWish(t, owner.ID, wishlist.ID, "Older wish", now.Add(-2*time.Hour))
	wishTestCreateNamedWish(t, owner.ID, wishlist.ID, "Newer wish", now.Add(-time.Hour))
	doomed := createTestWish(t, owner.ID, wishlist.ID)

	w := wishTestSend(t, wishTestReq{handler: DeleteWish, method: "DELETE", params: gin.Params{{Key: "wish_id", Value: doomed.ID.String()}}}, authHeader(t, owner.ID, false))
	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	names := wishTestWishNames(t, parseWishBody(t, w))
	if len(names) != 2 || names[0] != "Newer wish" || names[1] != "Older wish" {
		t.Errorf("wishes = %v, want [Newer wish Older wish]", names)
	}
}

// The claimant notification runs after the response is written, so its
// failures are only logged: the delete must still succeed.
func TestDeleteWish_ClaimNotificationFailuresAreNotFatal(t *testing.T) {
	cases := []struct {
		name  string
		op    string
		table string
		skip  int
	}{
		// wishlists: ConvertWishToWishObject, ownership check, then GetWishlist.
		{name: "wishlist lookup", op: "query", table: "wishlists", skip: 2},
		{name: "wishlist conversion", op: "query", table: "groups", skip: 0},
		// users: wish owner, claimant, wishlist owner, the wishlist
		// conversion's owner lookup, then the claimant's full record.
		{name: "claimant lookup", op: "query", table: "users", skip: 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			owner, claimant, _, wish := claimSetup(t)
			wishTestSeedClaim(t, wish.ID, claimant.ID)
			header := authHeader(t, owner.ID, false)
			failDBOperation(t, c.op, c.table, c.skip)

			w := wishTestSend(t, wishTestReq{handler: DeleteWish, method: "DELETE", params: gin.Params{{Key: "wish_id", Value: wish.ID.String()}}}, header)
			if w.Code != 201 {
				t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
			}
			if msg := parseWishBody(t, w)["message"]; msg != "Wish deleted." {
				t.Errorf("message = %v, want %q", msg, "Wish deleted.")
			}
		})
	}
}

func TestParseRawURLFunction_Unparseable(t *testing.T) {
	domain, scheme, _ := parseRawURLFunction("%zz")
	if domain != "" || scheme != "" {
		t.Errorf("parseRawURLFunction(%%zz) = (%q, %q), want empty domain and scheme", domain, scheme)
	}
}

func TestRegisterWishClaim_InjectedFailures(t *testing.T) {
	fixture := func(t *testing.T) wishTestReq {
		_, claimant, wishlist, wish := claimSetup(t)
		return wishTestReq{handler: RegisterWishClaim, method: "POST", body: `{"wishlist_id":"` + wishlist.ID.String() + `"}`, params: gin.Params{{Key: "wish_id", Value: wish.ID.String()}}, userID: claimant.ID}
	}
	wishTestRunFailures(t, []wishTestFailCase{
		{name: "wishlist lookup", fixture: fixture, op: "query", table: "wishlists", want: 500, wantErr: "Failed to get wishlist object."},
		{name: "ownership check", fixture: fixture, op: "query", table: "wishlists", skip: 1, want: 500, wantErr: "Failed to verify ownership of wishlist."},
		{name: "membership check", fixture: fixture, op: "query", table: "wishlist_memberships", want: 500, wantErr: "Failed to verify membership to wishlist."},
		{name: "collaborator check", fixture: fixture, op: "query", table: "wishlist_collaborators", want: 500, wantErr: "Failed to verify wishlist collaborator status."},
		// The first wishes query resolves the wishlist ID.
		{name: "wish ownership check", fixture: fixture, op: "query", table: "wishes", skip: 1, want: 500, wantErr: "Failed to verify ownership of wishlist."},
		{name: "create claim", fixture: fixture, op: "create", table: "wish_claims", want: 500, wantErr: "Failed to create claim."},
		{name: "list wishes", fixture: fixture, op: "query", table: "wishes", skip: 2, want: 500, wantErr: "Failed to get wishes from database."},
	})
}

func TestRegisterWishClaim_ReturnsWishesNewestFirst(t *testing.T) {
	setupControllersDB(t)
	owner, claimant, wishlist, wish := claimSetup(t)
	wishTestCreateNamedWish(t, owner.ID, wishlist.ID, "Older wish", time.Now().Add(-time.Hour))

	w := wishTestSend(t, wishTestReq{handler: RegisterWishClaim, method: "POST", body: `{"wishlist_id":"` + wishlist.ID.String() + `"}`, params: gin.Params{{Key: "wish_id", Value: wish.ID.String()}}}, authHeader(t, claimant.ID, false))
	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	names := wishTestWishNames(t, parseWishBody(t, w))
	if len(names) != 2 || names[0] != wish.Name {
		t.Errorf("wishes = %v, want %q first", names, wish.Name)
	}
	if claimed, _ := database.VerifyWishIsClaimed(wish.ID); !claimed {
		t.Error("expected the wish to be claimed")
	}
}

func TestRemoveWishClaim_InjectedFailures(t *testing.T) {
	fixture := func(t *testing.T) wishTestReq {
		_, claimant, wishlist, wish := claimSetup(t)
		wishTestSeedClaim(t, wish.ID, claimant.ID)
		return wishTestReq{handler: RemoveWishClaim, method: "POST", body: `{"wishlist_id":"` + wishlist.ID.String() + `"}`, params: gin.Params{{Key: "wish_id", Value: wish.ID.String()}}, userID: claimant.ID}
	}
	wishTestRunFailures(t, []wishTestFailCase{
		{name: "wishlist lookup", fixture: fixture, op: "query", table: "wishlists", want: 500, wantErr: "Failed to get wishlist object."},
		{name: "ownership check", fixture: fixture, op: "query", table: "wishlists", skip: 1, want: 500, wantErr: "Failed to verify ownership of wishlist."},
		{name: "membership check", fixture: fixture, op: "query", table: "wishlist_memberships", want: 500, wantErr: "Failed to verify membership to wishlist."},
		{name: "collaborator check", fixture: fixture, op: "query", table: "wishlist_collaborators", want: 500, wantErr: "Failed to verify wishlist collaborator status."},
		{name: "delete claim", fixture: fixture, op: "update", table: "wish_claims", want: 500, wantErr: "Failed to delete claim."},
		{name: "list wishes", fixture: fixture, op: "query", table: "wishes", skip: 1, want: 500, wantErr: "Failed to get wishes from database."},
	})
}

func TestRemoveWishClaim_DanglingWishlistID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	claimant := createTestUser(t)
	wish := models.Wish{Name: "Gift", Enabled: true, OwnerID: owner.ID, WishlistID: uuid.New()}
	wish.ID = uuid.New()
	if _, err := database.CreateWishInDB(wish); err != nil {
		t.Fatalf("failed to create wish: %v", err)
	}

	w := wishTestSend(t, wishTestReq{handler: RemoveWishClaim, method: "POST", body: `{}`, params: gin.Params{{Key: "wish_id", Value: wish.ID.String()}}}, authHeader(t, claimant.ID, false))
	if w.Code != 404 {
		t.Fatalf("status = %d, want 404 for a wish with a dangling wishlist reference; body=%s", w.Code, w.Body.String())
	}
}

func TestRemoveWishClaim_NotMember(t *testing.T) {
	setupControllersDB(t)
	_, _, _, wish := claimSetup(t)
	stranger := createTestUser(t)

	w := wishTestSend(t, wishTestReq{handler: RemoveWishClaim, method: "POST", body: `{}`, params: gin.Params{{Key: "wish_id", Value: wish.ID.String()}}}, authHeader(t, stranger.ID, false))
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a non-member; body=%s", w.Code, w.Body.String())
	}
}

func TestRemoveWishClaim_ReturnsWishesNewestFirst(t *testing.T) {
	setupControllersDB(t)
	owner, claimant, wishlist, wish := claimSetup(t)
	wishTestCreateNamedWish(t, owner.ID, wishlist.ID, "Older wish", time.Now().Add(-time.Hour))
	wishTestSeedClaim(t, wish.ID, claimant.ID)

	w := wishTestSend(t, wishTestReq{handler: RemoveWishClaim, method: "POST", body: `{"wishlist_id":"` + wishlist.ID.String() + `"}`, params: gin.Params{{Key: "wish_id", Value: wish.ID.String()}}}, authHeader(t, claimant.ID, false))
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	names := wishTestWishNames(t, parseWishBody(t, w))
	if len(names) != 2 || names[0] != wish.Name {
		t.Errorf("wishes = %v, want %q first", names, wish.Name)
	}
	if claimed, _ := database.VerifyWishIsClaimed(wish.ID); claimed {
		t.Error("expected the claim to be removed")
	}
}

// wishTestUpdateFixture builds an APIUpdateWish call by the wish's owner;
// body is formatted with the wish's current name so it only changes what the
// test intends to.
func wishTestUpdateFixture(bodyFormat string) func(t *testing.T) wishTestReq {
	return func(t *testing.T) wishTestReq {
		owner := createTestUser(t)
		wishlist := createTestWishlist(t, owner.ID)
		wish := createTestWish(t, owner.ID, wishlist.ID)
		return wishTestReq{handler: APIUpdateWish, method: "POST", body: fmt.Sprintf(bodyFormat, wish.Name), params: gin.Params{{Key: "wish_id", Value: wish.ID.String()}}, userID: owner.ID}
	}
}

func TestAPIUpdateWish_InjectedFailures(t *testing.T) {
	same := wishTestUpdateFixture(`{"name":%q}`)
	wishTestRunFailures(t, []wishTestFailCase{
		{name: "ownership check", fixture: same, op: "query", table: "wishlists", want: 500, wantErr: "Failed to verify ownership of wishlist."},
		// The first wishes query loads the original wish.
		{name: "unique name check", fixture: wishTestUpdateFixture(`{"name":"Renamed %s"}`), op: "query", table: "wishes", skip: 1, want: 500, wantErr: "Failed to verify wish name."},
		{name: "category resolution", fixture: wishTestUpdateFixture(`{"name":%q,"category_name":"Books"}`), op: "query", table: "wish_categories", want: 500, wantErr: "Failed to resolve wish category."},
		{name: "save wish", fixture: same, op: "update", table: "wishes", want: 500, wantErr: "Failed to update wish in database."},
		{name: "convert wish", fixture: same, op: "query", table: "users", want: 500, wantErr: "Failed to convert wish to wish object."},
		{name: "list wishes", fixture: same, op: "query", table: "wishes", skip: 1, want: 500, wantErr: "Failed to get wishes from database."},
	})
}

func TestAPIUpdateWish_InvalidCharacters(t *testing.T) {
	cases := []struct {
		name       string
		bodyFormat string
	}{
		{name: "name", bodyFormat: `{"name":"Bad <name> %s"}`},
		{name: "url", bodyFormat: `{"name":%q,"url":"https://example.com/<b>"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			req := wishTestUpdateFixture(c.bodyFormat)(t)
			w := wishTestSend(t, req, authHeader(t, req.userID, false))
			if w.Code != 400 {
				t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestAPIUpdateWish_InvalidImageData(t *testing.T) {
	setupControllersDB(t)
	wishTestImageDir(t)
	req := wishTestUpdateFixture(`{"name":%q,"image_data":"not-valid-base64-image-data"}`)(t)

	w := wishTestSend(t, req, authHeader(t, req.userID, false))
	if w.Code != 500 {
		t.Fatalf("status = %d, want 500; body=%s", w.Code, w.Body.String())
	}
	if got := parseWishBody(t, w)["error"]; got != "Failed to save wish image." {
		t.Errorf("error = %v, want %q", got, "Failed to save wish image.")
	}
}

func TestAPIUpdateWish_ImageDelete(t *testing.T) {
	setupControllersDB(t)
	dir := wishTestImageDir(t)
	req := wishTestUpdateFixture(`{"name":%q,"image_delete":true}`)(t)
	wishID := uuid.MustParse(req.params[0].Value)
	for _, thumbnail := range []bool{false, true} {
		path := imageFilePath(dir, wishID, thumbnail)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create image dir: %v", err)
		}
		if err := os.WriteFile(path, []byte("jpeg"), 0644); err != nil {
			t.Fatalf("failed to seed image: %v", err)
		}
	}

	w := wishTestSend(t, req, authHeader(t, req.userID, false))
	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if exists, _ := CheckIfWishImageExists(wishID); exists {
		t.Error("expected the wish image to be removed")
	}
}

func TestAPIUpdateWish_ImageDeleteFailure(t *testing.T) {
	setupControllersDB(t)
	req := wishTestUpdateFixture(`{"name":%q,"image_delete":true}`)(t)
	wishTestBlockImageDelete(t, uuid.MustParse(req.params[0].Value))

	w := wishTestSend(t, req, authHeader(t, req.userID, false))
	if w.Code != 500 {
		t.Fatalf("status = %d, want 500; body=%s", w.Code, w.Body.String())
	}
}

func TestAPIUpdateWish_ReturnsWishesNewestFirst(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wishTestCreateNamedWish(t, owner.ID, wishlist.ID, "Older wish", time.Now().Add(-time.Hour))
	wish := createTestWish(t, owner.ID, wishlist.ID)

	w := wishTestSend(t, wishTestReq{handler: APIUpdateWish, method: "POST", body: `{"name":"Updated wish name"}`, params: gin.Params{{Key: "wish_id", Value: wish.ID.String()}}}, authHeader(t, owner.ID, false))
	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	names := wishTestWishNames(t, parseWishBody(t, w))
	if len(names) != 2 || names[0] != "Updated wish name" {
		t.Errorf("wishes = %v, want the updated wish first", names)
	}
}

func TestAPIGetWish_InjectedFailures(t *testing.T) {
	fixture := func(t *testing.T) wishTestReq {
		owner := createTestUser(t)
		wishlist := createTestWishlist(t, owner.ID)
		wish := createTestWish(t, owner.ID, wishlist.ID)
		return wishTestReq{handler: APIGetWish, method: "GET", params: gin.Params{{Key: "wish_id", Value: wish.ID.String()}}, userID: owner.ID}
	}
	wishTestRunFailures(t, []wishTestFailCase{
		{name: "ownership check", fixture: fixture, op: "query", table: "wishlists", want: 500, wantErr: "Failed to verify wishlist ownership."},
		// The first wishes query resolves the wishlist ID.
		{name: "wish lookup", fixture: fixture, op: "query", table: "wishes", skip: 1, want: 500, wantErr: "Failed to get wish from database."},
		{name: "convert wish", fixture: fixture, op: "query", table: "users", want: 500, wantErr: "Failed to convert wish to wish object."},
	})
}

func TestAPIGetWish_WishDisappearsMidRequest(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createTestWish(t, owner.ID, wishlist.ID)
	header := authHeader(t, owner.ID, false)

	// Disable the wish just before the ownership check - after the handler
	// has resolved its wishlist, but before it loads the wish itself.
	fired := false
	err := database.Instance.Callback().Query().Before("gorm:query").Register("test:wish:disable:"+uuid.NewString(), func(db *gorm.DB) {
		if fired || db.Statement.Table != "wishlists" {
			return
		}
		fired = true
		if err := database.Instance.Exec("UPDATE wishes SET enabled = ? WHERE id = ?", false, wish.ID).Error; err != nil {
			t.Errorf("failed to disable wish: %v", err)
		}
	})
	if err != nil {
		t.Fatalf("failed to register callback: %v", err)
	}

	w := wishTestSend(t, wishTestReq{handler: APIGetWish, method: "GET", params: gin.Params{{Key: "wish_id", Value: wish.ID.String()}}}, header)
	if w.Code != 404 {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
	if got := parseWishBody(t, w)["error"]; got != "Failed to find wish in the database." {
		t.Errorf("error = %v, want %q", got, "Failed to find wish in the database.")
	}
}
