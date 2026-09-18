package controllers

import (
	"aunefyren/poenskelisten/database"
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
