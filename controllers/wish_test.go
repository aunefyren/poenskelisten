package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
