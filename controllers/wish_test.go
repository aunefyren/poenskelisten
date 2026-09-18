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
