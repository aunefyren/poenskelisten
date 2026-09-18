package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// wlDo calls a handler directly with an optional JSON body, Authorization
// header and path params, mirroring the pattern in oauth_register_test.go
// and session_test.go.
func wlDo(handler gin.HandlerFunc, method, path, body, authHdr string, params gin.Params) (int, map[string]interface{}) {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(method, path, reader)
	if body != "" {
		ctx.Request.Header.Set("Content-Type", "application/json")
	}
	if authHdr != "" {
		ctx.Request.Header.Set("Authorization", authHdr)
	}
	if params != nil {
		ctx.Params = params
	}

	handler(ctx)

	var parsed map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &parsed)
	return w.Code, parsed
}

func wlFutureDate() string {
	return time.Now().Add(72 * time.Hour).UTC().Format("2006-01-02T15:04:05.000Z")
}

func wlPastDate() string {
	return time.Now().Add(-72 * time.Hour).UTC().Format("2006-01-02T15:04:05.000Z")
}

// --- RegisterWishlist ---

func TestRegisterWishlistSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	body := `{"name":"Birthday list","description":"stuff","expires":true,"date":"` + wlFutureDate() + `","claimable":true}`
	code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists", body, authHeader(t, owner.ID, false), nil)
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}
	if resp["message"] != "Wishlist created." {
		t.Errorf("message = %v", resp["message"])
	}
}

func TestRegisterWishlistRequiresAuth(t *testing.T) {
	setupControllersDB(t)

	body := `{"name":"Birthday list","expires":false}`
	code, _ := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists", body, "", nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400 without auth", code)
	}
}

func TestRegisterWishlistNameTooShort(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	body := `{"name":"abcd","expires":false}`
	code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists", body, authHeader(t, owner.ID, false), nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a short name; body=%v", code, resp)
	}
}

func TestRegisterWishlistDuplicateName(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	body := `{"name":"Same Name List","expires":false}`
	if code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists", body, authHeader(t, owner.ID, false), nil); code != 201 {
		t.Fatalf("first create: status = %d, want 201; body=%v", code, resp)
	}
	code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists", body, authHeader(t, owner.ID, false), nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a duplicate name; body=%v", code, resp)
	}
}

func TestRegisterWishlistNotMemberOfGroup(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	body := `{"name":"Unshared List","expires":false,"groups":["` + uuid.New().String() + `"]}`
	code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists", body, authHeader(t, owner.ID, false), nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400 when not a member of the referenced group; body=%v", code, resp)
	}
}

func TestRegisterWishlistPublicAndClaimableConflict(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	body := `{"name":"Public List","expires":false,"public":true,"claimable":true}`
	code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists", body, authHeader(t, owner.ID, false), nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400 for public+claimable; body=%v", code, resp)
	}
}

func TestRegisterWishlistPastDate(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	body := `{"name":"Past Date List","expires":true,"date":"` + wlPastDate() + `"}`
	code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists", body, authHeader(t, owner.ID, false), nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a past date; body=%v", code, resp)
	}
}

// --- DeleteWishlist ---

func TestDeleteWishlistSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	code, resp := wlDo(DeleteWishlist, "DELETE", "/api/auth/wishlists/"+wishlist.ID.String(), "", authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	if resp["message"] != "Wishlist deleted." {
		t.Errorf("message = %v", resp["message"])
	}
}

func TestDeleteWishlistNotOwner(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	other := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	code, resp := wlDo(DeleteWishlist, "DELETE", "/api/auth/wishlists/"+wishlist.ID.String(), "", authHeader(t, other.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a non-owner delete; body=%v", code, resp)
	}
}

func TestDeleteWishlistBadUUID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	code, resp := wlDo(DeleteWishlist, "DELETE", "/api/auth/wishlists/not-a-uuid", "", authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: "not-a-uuid"}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed wishlist_id; body=%v", code, resp)
	}
}

func TestDeleteWishlistWithGroupQuery(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	wishlist := createTestWishlist(t, owner.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)

	code, resp := wlDo(DeleteWishlist, "DELETE", "/api/auth/wishlists/"+wishlist.ID.String()+"?group="+group.ID.String(), "", authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
}

// --- GetWishlist ---

func TestGetWishlistAsOwner(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	code, resp := wlDo(GetWishlist, "GET", "/api/auth/wishlists/"+wishlist.ID.String(), "", authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	if resp["message"] != "Wishlist retrieved." {
		t.Errorf("message = %v", resp["message"])
	}
}

func TestGetWishlistAsGroupMember(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	member := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, member.ID)
	wishlist := createTestWishlist(t, owner.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)

	code, resp := wlDo(GetWishlist, "GET", "/api/auth/wishlists/"+wishlist.ID.String(), "", authHeader(t, member.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 200 {
		t.Fatalf("status = %d, want 200 for a shared group member; body=%v", code, resp)
	}
}

func TestGetWishlistForbidden(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	stranger := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	code, resp := wlDo(GetWishlist, "GET", "/api/auth/wishlists/"+wishlist.ID.String(), "", authHeader(t, stranger.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for an unrelated user; body=%v", code, resp)
	}
}

func TestGetWishlistBadUUID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	code, _ := wlDo(GetWishlist, "GET", "/api/auth/wishlists/nope", "", authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: "nope"}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed wishlist_id", code)
	}
}

// --- GetWishlists ---

func TestGetWishlistsNoFilter(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	createTestWishlist(t, owner.ID)
	createTestWishlist(t, owner.ID)

	code, resp := wlDo(GetWishlists, "GET", "/api/auth/wishlists", "", authHeader(t, owner.ID, false), nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	list, ok := resp["wishlists"].([]interface{})
	if !ok || len(list) != 2 {
		t.Errorf("wishlists = %v, want 2 entries", resp["wishlists"])
	}
}

func TestGetWishlistsGroupFilter(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	member := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, member.ID)
	wishlist := createTestWishlist(t, owner.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)

	code, resp := wlDo(GetWishlists, "GET", "/api/auth/wishlists?group="+group.ID.String(), "", authHeader(t, member.ID, false), nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	list, ok := resp["wishlists"].([]interface{})
	if !ok || len(list) != 1 {
		t.Errorf("wishlists = %v, want 1 entry", resp["wishlists"])
	}
}

func TestGetWishlistsGroupFilterNotMember(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	stranger := createTestUser(t)
	group := createTestGroup(t, owner.ID)

	code, _ := wlDo(GetWishlists, "GET", "/api/auth/wishlists?group="+group.ID.String(), "", authHeader(t, stranger.ID, false), nil)
	// The handler reports non-membership as a 500 (an existing inconsistency
	// with the 400 used elsewhere for the same condition) - asserting the
	// actual behavior here, not the "should be" status.
	if code != 500 {
		t.Fatalf("status = %d, want 500 for a non-member group filter (see note above)", code)
	}
}

func TestGetWishlistsOwnedFilterExcludesGroupOnlyAccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	member := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, member.ID)
	wishlist := createTestWishlist(t, owner.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)

	code, resp := wlDo(GetWishlists, "GET", "/api/auth/wishlists?owned=true", "", authHeader(t, member.ID, false), nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	list, ok := resp["wishlists"].([]interface{})
	if !ok || len(list) != 0 {
		t.Errorf("wishlists = %v, want 0 entries (group access only, not owned/collab)", resp["wishlists"])
	}
}

func TestGetWishlistsExpiredFilter(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	expiredBody := `{"name":"Expired List","expires":false}`
	if code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists", expiredBody, authHeader(t, owner.ID, false), nil); code != 201 {
		t.Fatalf("setup: status = %d; body=%v", code, resp)
	}

	code, resp := wlDo(GetWishlists, "GET", "/api/auth/wishlists?expired=false", "", authHeader(t, owner.ID, false), nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	list, ok := resp["wishlists"].([]interface{})
	if !ok || len(list) != 1 {
		t.Errorf("wishlists = %v, want 1 non-expired entry", resp["wishlists"])
	}
}

func TestGetWishlistsTopLimit(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	for i := 0; i < 5; i++ {
		createTestWishlist(t, owner.ID)
	}

	code, resp := wlDo(GetWishlists, "GET", "/api/auth/wishlists?top=2", "", authHeader(t, owner.ID, false), nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	list, ok := resp["wishlists"].([]interface{})
	if !ok || len(list) != 2 {
		t.Errorf("wishlists = %v, want top=2 to limit to 2 entries", resp["wishlists"])
	}
}

// --- JoinWishlist ---

func TestJoinWishlistSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	group := createTestGroup(t, owner.ID)

	body := `{"groups":["` + group.ID.String() + `"]}`
	code, resp := wlDo(JoinWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/join", body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}
}

func TestJoinWishlistRequiresGroups(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	code, resp := wlDo(JoinWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/join", `{"groups":[]}`, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for an empty groups list; body=%v", code, resp)
	}
}

func TestJoinWishlistNotOwner(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	other := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	group := createTestGroup(t, owner.ID)

	body := `{"groups":["` + group.ID.String() + `"]}`
	code, resp := wlDo(JoinWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/join", body, authHeader(t, other.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a non-owner; body=%v", code, resp)
	}
}

func TestJoinWishlistAlreadyMember(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	group := createTestGroup(t, owner.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)

	body := `{"groups":["` + group.ID.String() + `"]}`
	code, resp := wlDo(JoinWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/join", body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for an existing membership; body=%v", code, resp)
	}
}

// --- RemoveFromWishlist ---

func TestRemoveFromWishlistSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	group := createTestGroup(t, owner.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)

	body := `{"group_id":"` + group.ID.String() + `"}`
	code, resp := wlDo(RemoveFromWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/leave", body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}
}

func TestRemoveFromWishlistFinalListDBFailure(t *testing.T) {
	// Migrate without the WishlistCollaborator table so, after the membership
	// is successfully deleted, the final GetWishlistObjects call (used when no
	// ?group= query param is given) fails while re-listing the user's wishlists.
	setupControllersDB(t, &models.User{}, &models.Wishlist{}, &models.WishlistMembership{}, &models.Group{}, &models.GroupMembership{})
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	group := createTestGroup(t, owner.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)

	body := `{"group_id":"` + group.ID.String() + `"}`
	code, resp := wlDo(RemoveFromWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/leave", body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 500 {
		t.Fatalf("status = %d, want 500 when the wishlist_collaborators table is unavailable; body=%v", code, resp)
	}
}

func TestRemoveFromWishlistNoMembership(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	group := createTestGroup(t, owner.ID)

	body := `{"group_id":"` + group.ID.String() + `"}`
	code, resp := wlDo(RemoveFromWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/leave", body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 when the membership doesn't exist; body=%v", code, resp)
	}
}

func TestRemoveFromWishlistNotOwner(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	other := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	group := createTestGroup(t, owner.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)

	body := `{"group_id":"` + group.ID.String() + `"}`
	code, resp := wlDo(RemoveFromWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/leave", body, authHeader(t, other.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a non-owner; body=%v", code, resp)
	}
}

// --- APIUpdateWishlist ---

func TestAPIUpdateWishlistAsOwner(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	body := `{"name":"Renamed List","expires":false}`
	code, resp := wlDo(APIUpdateWishlist, "PUT", "/api/auth/wishlists/"+wishlist.ID.String(), body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}
}

func TestAPIUpdateWishlistForbidden(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	stranger := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	body := `{"name":"Renamed List","expires":false}`
	code, resp := wlDo(APIUpdateWishlist, "PUT", "/api/auth/wishlists/"+wishlist.ID.String(), body, authHeader(t, stranger.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a non-owner/non-collaborator; body=%v", code, resp)
	}
}

func TestAPIUpdateWishlistNameTooShort(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	body := `{"name":"ab","expires":false}`
	code, resp := wlDo(APIUpdateWishlist, "PUT", "/api/auth/wishlists/"+wishlist.ID.String(), body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a too-short name; body=%v", code, resp)
	}
}

func TestAPIUpdateWishlistPublicAndClaimableConflict(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	body := `{"name":"` + wishlist.Name + `","expires":false,"public":true,"claimable":true}`
	code, resp := wlDo(APIUpdateWishlist, "PUT", "/api/auth/wishlists/"+wishlist.ID.String(), body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for public+claimable; body=%v", code, resp)
	}
}

// --- APICollaborateWishlist / APIUnCollaborateWishlist ---

func TestAPICollaborateWishlistSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	collaborator := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	body := `{"users":["` + collaborator.ID.String() + `"]}`
	code, resp := wlDo(APICollaborateWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/collaborators", body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}
}

func TestAPICollaborateWishlistRequiresUsers(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	code, resp := wlDo(APICollaborateWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/collaborators", `{"users":[]}`, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for an empty users list; body=%v", code, resp)
	}
}

func TestAPICollaborateWishlistNotOwner(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	other := createTestUser(t)
	collaborator := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	body := `{"users":["` + collaborator.ID.String() + `"]}`
	code, resp := wlDo(APICollaborateWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/collaborators", body, authHeader(t, other.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 401 {
		t.Fatalf("status = %d, want 401 for a non-owner; body=%v", code, resp)
	}
}

func TestAPICollaborateWishlistSelf(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	body := `{"users":["` + owner.ID.String() + `"]}`
	code, resp := wlDo(APICollaborateWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/collaborators", body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 when the owner adds themselves; body=%v", code, resp)
	}
}

func TestAPIUnCollaborateWishlistSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	collaborator := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	addBody := `{"users":["` + collaborator.ID.String() + `"]}`
	if code, resp := wlDo(APICollaborateWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/collaborators", addBody, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}}); code != 201 {
		t.Fatalf("setup: status = %d; body=%v", code, resp)
	}

	removeBody := `{"user_id":"` + collaborator.ID.String() + `"}`
	code, resp := wlDo(APIUnCollaborateWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/collaborators/remove", removeBody, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}
}

func TestAPIUnCollaborateWishlistDoesNotExist(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	stranger := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	body := `{"user_id":"` + stranger.ID.String() + `"}`
	code, resp := wlDo(APIUnCollaborateWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/collaborators/remove", body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 when no collaboration exists; body=%v", code, resp)
	}
}

// --- GetPublicWishlist ---

func TestGetPublicWishlistSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	createBody := `{"name":"Public List","expires":false,"public":true,"claimable":false}`
	if code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists", createBody, authHeader(t, owner.ID, false), nil); code != 201 {
		t.Fatalf("setup: status = %d, want 201; body=%v", code, resp)
	}

	owned, err := database.GetOwnedWishlists(owner.ID)
	if err != nil || len(owned) != 1 {
		t.Fatalf("failed to look up the created wishlist: %v (found %d)", err, len(owned))
	}
	hash := owned[0].PublicHash.String()

	code, resp := wlDo(GetPublicWishlist, "GET", "/api/open/wishlists/public/"+hash, "", "",
		gin.Params{{Key: "wishlist_hash", Value: hash}})
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
}

func TestGetPublicWishlistNotFound(t *testing.T) {
	setupControllersDB(t)

	code, resp := wlDo(GetPublicWishlist, "GET", "/api/open/wishlists/public/"+uuid.New().String(), "", "",
		gin.Params{{Key: "wishlist_hash", Value: uuid.New().String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for an unknown hash; body=%v", code, resp)
	}
}

func TestGetPublicWishlistBadUUID(t *testing.T) {
	setupControllersDB(t)

	code, _ := wlDo(GetPublicWishlist, "GET", "/api/open/wishlists/public/not-a-uuid", "", "",
		gin.Params{{Key: "wishlist_hash", Value: "not-a-uuid"}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed hash", code)
	}
}

// --- RegisterWishlist additional branches ---

func TestRegisterWishlistBadJSON(t *testing.T) {
	setupControllersDB(t)
	code, _ := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists", `{not-json`, "", nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400 for malformed JSON", code)
	}
}

func TestRegisterWishlistInvalidDescriptionCharacters(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	body := `{"name":"A nice list","description":"bad \"quote\"","expires":false}`
	code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists", body, authHeader(t, owner.ID, false), nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400 for disallowed characters in the description; body=%v", code, resp)
	}
}

func TestRegisterWishlistBadDateFormat(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	body := `{"name":"A nice list","expires":true,"date":"not-a-date"}`
	code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists", body, authHeader(t, owner.ID, false), nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed date; body=%v", code, resp)
	}
}

func TestRegisterWishlistWithGroupContextID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)

	body := `{"name":"A nice list","expires":false}`
	code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists?groupContextID="+group.ID.String(), body, authHeader(t, owner.ID, false), nil)
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}
}

func TestRegisterWishlistBadGroupContextID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	body := `{"name":"A nice list","expires":false}`
	code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists?groupContextID=not-a-uuid", body, authHeader(t, owner.ID, false), nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed groupContextID; body=%v", code, resp)
	}
}

// --- DeleteWishlist additional branches ---

func TestDeleteWishlistGroupQueryBadUUID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	code, resp := wlDo(DeleteWishlist, "DELETE", "/api/auth/wishlists/"+wishlist.ID.String()+"?group=not-a-uuid", "", authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed group id; body=%v", code, resp)
	}
}

func TestDeleteWishlistGroupQueryNotMember(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	group := createTestGroup(t, owner.ID) // owner not added as a member

	code, resp := wlDo(DeleteWishlist, "DELETE", "/api/auth/wishlists/"+wishlist.ID.String()+"?group="+group.ID.String(), "", authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a group the caller isn't a member of; body=%v", code, resp)
	}
}

// --- GetWishlist additional branches ---

func TestGetWishlistRequiresAuth(t *testing.T) {
	setupControllersDB(t)
	wishlist := createTestWishlist(t, createTestUser(t).ID)

	code, _ := wlDo(GetWishlist, "GET", "/api/auth/wishlists/"+wishlist.ID.String(), "", "",
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 without auth", code)
	}
}

// --- GetWishlists additional branches ---

func TestGetWishlistsRequiresAuth(t *testing.T) {
	setupControllersDB(t)
	code, _ := wlDo(GetWishlists, "GET", "/api/auth/wishlists", "", "", nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400 without auth", code)
	}
}

func TestGetWishlistsGroupFilterBadUUID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	code, _ := wlDo(GetWishlists, "GET", "/api/auth/wishlists?group=not-a-uuid", "", authHeader(t, owner.ID, false), nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed group id", code)
	}
}

// TestGetWishlistsTopLimitOffByOne pins down the documented bug in docs/wip.md:
// with exactly top+1 items, the filter fails to truncate at all.
func TestGetWishlistsTopLimitOffByOne(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	for i := 0; i < 3; i++ {
		createTestWishlist(t, owner.ID)
	}

	code, resp := wlDo(GetWishlists, "GET", "/api/auth/wishlists?top=2", "", authHeader(t, owner.ID, false), nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	list, ok := resp["wishlists"].([]interface{})
	if !ok || len(list) != 3 {
		t.Fatalf("wishlists = %v, want all 3 (documented off-by-one bug: top=2 should truncate to 2 but doesn't at exactly top+1 items)", resp["wishlists"])
	}
}

func TestGetWishlistsNotAMemberOfGroupFilter(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	member := createTestUser(t)
	sharedGroup := createTestGroup(t, owner.ID)
	otherGroup := createTestGroup(t, owner.ID)
	addGroupMembership(t, sharedGroup.ID, member.ID)
	addGroupMembership(t, otherGroup.ID, member.ID)

	sharedWishlist := createTestWishlist(t, owner.ID)
	addWishlistMembership(t, sharedWishlist.ID, sharedGroup.ID)
	otherWishlist := createTestWishlist(t, owner.ID)
	addWishlistMembership(t, otherWishlist.ID, otherGroup.ID)

	code, resp := wlDo(GetWishlists, "GET", "/api/auth/wishlists?notAMemberOfGroupID="+sharedGroup.ID.String(), "", authHeader(t, member.ID, false), nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	list, ok := resp["wishlists"].([]interface{})
	if !ok || len(list) != 1 {
		t.Errorf("wishlists = %v, want exactly the wishlist not shared via sharedGroup", resp["wishlists"])
	}
}

func TestGetWishlistsNotAMemberOfGroupFilterBadUUID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	code, _ := wlDo(GetWishlists, "GET", "/api/auth/wishlists?notAMemberOfGroupID=not-a-uuid", "", authHeader(t, owner.ID, false), nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed notAMemberOfGroupID", code)
	}
}

// --- JoinWishlist additional branches ---

func TestJoinWishlistBadJSON(t *testing.T) {
	setupControllersDB(t)
	code, _ := wlDo(JoinWishlist, "POST", "/api/auth/wishlists/"+uuid.NewString()+"/join", `{not-json`, "",
		gin.Params{{Key: "wishlist_id", Value: uuid.NewString()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for malformed JSON", code)
	}
}

func TestJoinWishlistBadUUID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)

	body := `{"groups":["` + group.ID.String() + `"]}`
	code, resp := wlDo(JoinWishlist, "POST", "/api/auth/wishlists/not-a-uuid/join", body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: "not-a-uuid"}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed wishlist id; body=%v", code, resp)
	}
}

func TestJoinWishlistUnknownGroup(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	body := `{"groups":["` + uuid.NewString() + `"]}`
	code, resp := wlDo(JoinWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/join", body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 500 {
		t.Fatalf("status = %d, want 500 when the group doesn't exist; body=%v", code, resp)
	}
}

// --- RemoveFromWishlist additional branches ---

func TestRemoveFromWishlistBadJSON(t *testing.T) {
	setupControllersDB(t)
	code, _ := wlDo(RemoveFromWishlist, "POST", "/api/auth/wishlists/"+uuid.NewString()+"/leave", `{not-json`, "",
		gin.Params{{Key: "wishlist_id", Value: uuid.NewString()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for malformed JSON", code)
	}
}

func TestRemoveFromWishlistRequiresAuth(t *testing.T) {
	setupControllersDB(t)
	wishlist := createTestWishlist(t, createTestUser(t).ID)
	body := `{"group_id":"` + uuid.NewString() + `"}`

	code, _ := wlDo(RemoveFromWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/leave", body, "",
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 without auth", code)
	}
}

func TestRemoveFromWishlistBadUUID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	body := `{"group_id":"` + uuid.NewString() + `"}`

	code, resp := wlDo(RemoveFromWishlist, "POST", "/api/auth/wishlists/not-a-uuid/leave", body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: "not-a-uuid"}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed wishlist id; body=%v", code, resp)
	}
}

func TestRemoveFromWishlistWithGroupQuery(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	wishlist := createTestWishlist(t, owner.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)

	// Leaving removes the only membership, so use a second membership to keep
	// the wishlist "found" for the subsequent group-scoped listing.
	otherGroup := createTestGroup(t, owner.ID)
	addGroupMembership(t, otherGroup.ID, owner.ID)
	addWishlistMembership(t, wishlist.ID, otherGroup.ID)

	body := `{"group_id":"` + group.ID.String() + `"}`
	code, resp := wlDo(RemoveFromWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/leave?group="+otherGroup.ID.String(), body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}
}

// --- APIUpdateWishlist additional branches ---

func TestAPIUpdateWishlistBadJSON(t *testing.T) {
	setupControllersDB(t)
	code, _ := wlDo(APIUpdateWishlist, "PUT", "/api/auth/wishlists/"+uuid.NewString(), `{not-json`, "",
		gin.Params{{Key: "wishlist_id", Value: uuid.NewString()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for malformed JSON", code)
	}
}

func TestAPIUpdateWishlistBadUUID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	body := `{"name":"Renamed","expires":false}`

	code, resp := wlDo(APIUpdateWishlist, "PUT", "/api/auth/wishlists/not-a-uuid", body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: "not-a-uuid"}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed wishlist id; body=%v", code, resp)
	}
}

func TestAPIUpdateWishlistRequiresAuth(t *testing.T) {
	setupControllersDB(t)
	wishlist := createTestWishlist(t, createTestUser(t).ID)
	body := `{"name":"Renamed","expires":false}`

	code, _ := wlDo(APIUpdateWishlist, "PUT", "/api/auth/wishlists/"+wishlist.ID.String(), body, "",
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 without auth", code)
	}
}

func TestAPIUpdateWishlistInvalidDescriptionCharacters(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	body := `{"name":"` + wishlist.Name + `","description":"bad \"quote\"","expires":false}`
	code, resp := wlDo(APIUpdateWishlist, "PUT", "/api/auth/wishlists/"+wishlist.ID.String(), body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for disallowed characters in the description; body=%v", code, resp)
	}
}

func TestAPIUpdateWishlistBadDateFormat(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	body := `{"name":"` + wishlist.Name + `","expires":true,"date":"not-a-date"}`
	code, resp := wlDo(APIUpdateWishlist, "PUT", "/api/auth/wishlists/"+wishlist.ID.String(), body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed date; body=%v", code, resp)
	}
}

func TestAPIUpdateWishlistCollaboratorSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	collaborator := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	addWishCollaborator(t, wishlist.ID, collaborator.ID)

	body := `{"name":"Updated by collaborator","expires":false}`
	code, resp := wlDo(APIUpdateWishlist, "PUT", "/api/auth/wishlists/"+wishlist.ID.String(), body, authHeader(t, collaborator.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}
}

// --- APICollaborateWishlist additional branches ---

func TestAPICollaborateWishlistBadJSON(t *testing.T) {
	setupControllersDB(t)
	code, _ := wlDo(APICollaborateWishlist, "POST", "/api/auth/wishlists/"+uuid.NewString()+"/collaborators", `{not-json`, "",
		gin.Params{{Key: "wishlist_id", Value: uuid.NewString()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for malformed JSON", code)
	}
}

func TestAPICollaborateWishlistBadUUID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	collaborator := createTestUser(t)
	body := `{"users":["` + collaborator.ID.String() + `"]}`

	code, resp := wlDo(APICollaborateWishlist, "POST", "/api/auth/wishlists/not-a-uuid/collaborators", body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: "not-a-uuid"}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed wishlist id; body=%v", code, resp)
	}
}

func TestAPICollaborateWishlistUnknownUser(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	body := `{"users":["` + uuid.NewString() + `"]}`

	code, resp := wlDo(APICollaborateWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/collaborators", body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for an unknown user; body=%v", code, resp)
	}
}

func TestAPICollaborateWishlistAlreadyCollaborating(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	collaborator := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	addWishCollaborator(t, wishlist.ID, collaborator.ID)

	body := `{"users":["` + collaborator.ID.String() + `"]}`
	code, resp := wlDo(APICollaborateWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/collaborators", body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for an existing collaboration; body=%v", code, resp)
	}
}

// --- APIUnCollaborateWishlist additional branches ---

func TestAPIUnCollaborateWishlistBadJSON(t *testing.T) {
	setupControllersDB(t)
	code, _ := wlDo(APIUnCollaborateWishlist, "POST", "/api/auth/wishlists/"+uuid.NewString()+"/collaborators/remove", `{not-json`, "",
		gin.Params{{Key: "wishlist_id", Value: uuid.NewString()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for malformed JSON", code)
	}
}

func TestAPIUnCollaborateWishlistBadUUID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	body := `{"user_id":"` + uuid.NewString() + `"}`

	code, resp := wlDo(APIUnCollaborateWishlist, "POST", "/api/auth/wishlists/not-a-uuid/collaborators/remove", body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: "not-a-uuid"}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed wishlist id; body=%v", code, resp)
	}
}

func TestAPIUnCollaborateWishlistNotOwner(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	other := createTestUser(t)
	collaborator := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	addWishCollaborator(t, wishlist.ID, collaborator.ID)

	body := `{"user_id":"` + collaborator.ID.String() + `"}`
	code, resp := wlDo(APIUnCollaborateWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/collaborators/remove", body, authHeader(t, other.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 401 {
		t.Fatalf("status = %d, want 401 for a non-owner; body=%v", code, resp)
	}
}

// --- ConvertWishlistToWishlistObject / GetWishlistObjects branch coverage ---

func TestGetWishlistObjectsIncludesCollaborations(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	collaborator := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	addWishCollaborator(t, wishlist.ID, collaborator.ID)

	objects, err := GetWishlistObjects(collaborator.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(objects) != 1 {
		t.Fatalf("objects = %v, want exactly the one wishlist the caller collaborates on", objects)
	}
}

func TestGetAllWishlistObjectsDeduplicatesAcrossAccessPaths(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	member := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, member.ID)
	wishlist := createTestWishlist(t, owner.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)
	// Also make the member a direct collaborator on the same wishlist, so it
	// would appear via two access paths if not deduplicated.
	addWishCollaborator(t, wishlist.ID, member.ID)

	objects, err := GetAllWishlistObjects(member.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(objects) != 1 {
		t.Errorf("objects = %v, want exactly 1 (deduplicated across group membership + collaboration)", objects)
	}
}

func TestRegisterWishlistSharedWithGroupSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)

	body := `{"name":"Shared List","expires":false,"groups":["` + group.ID.String() + `"]}`
	code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists", body, authHeader(t, owner.ID, false), nil)
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}

	memberships, err := database.GetWishlistsFromGroup(group.ID)
	if err != nil {
		t.Fatalf("failed to list group wishlists: %v", err)
	}
	if len(memberships) != 1 {
		t.Errorf("group wishlist memberships = %v, want 1", memberships)
	}
}
