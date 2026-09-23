package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"encoding/json"
	"gorm.io/gorm"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a non-member group filter", code)
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

// A ?group= the caller isn't a member of is a caller mistake, so it's a 400 -
// this used to be reported as a 500.
func TestRemoveFromWishlistNotMemberOfQueriedGroup(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	group := createTestGroup(t, owner.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)

	body := `{"group_id":"` + group.ID.String() + `"}`
	code, resp := wlDo(RemoveFromWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/leave?group="+group.ID.String(), body, authHeader(t, owner.ID, false),
		gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}})
	if code != 400 {
		t.Fatalf("status = %d, want 400 for a group the caller isn't a member of; body=%v", code, resp)
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

// TestGetWishlistsTopLimitExactlyOneOver covers the boundary case where the
// result count is exactly top+1: the ?top= filter must still truncate.
func TestGetWishlistsTopLimitExactlyOneOver(t *testing.T) {
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
	if !ok || len(list) != 2 {
		t.Fatalf("wishlists = %v, want exactly 2 (top=2 should truncate)", resp["wishlists"])
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

	// Editing as a collaborator must not hand the wishlist to the collaborator.
	updated, err := database.GetWishlist(wishlist.ID)
	if err != nil {
		t.Fatalf("failed to reload wishlist: %v", err)
	}
	if updated.Name != "Updated by collaborator" {
		t.Errorf("name = %q, want the collaborator's edit to be saved", updated.Name)
	}
	if updated.OwnerID != owner.ID {
		t.Errorf("owner = %v, want the original owner %v", updated.OwnerID, owner.ID)
	}
}

// Names are unique per owner, so a collaborator can't rename a wishlist to
// clash with another of the owner's wishlists...
func TestAPIUpdateWishlistCollaboratorDuplicateOfOwnersName(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	collaborator := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	ownersOther := createTestWishlist(t, owner.ID)
	addWishCollaborator(t, wishlist.ID, collaborator.ID)

	body := `{"name":"` + ownersOther.Name + `"}`
	code, resp := wlDo(APIUpdateWishlist, "PUT", "/api/auth/wishlists/"+wishlist.ID.String(), body, authHeader(t, collaborator.ID, false), wishlistTestParams(wishlist.ID))
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
}

// ...but may reuse a name the collaborator has on their own profile.
func TestAPIUpdateWishlistCollaboratorMayReuseOwnName(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	collaborator := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	collaboratorsOwn := createTestWishlist(t, collaborator.ID)
	addWishCollaborator(t, wishlist.ID, collaborator.ID)

	body := `{"name":"` + collaboratorsOwn.Name + `"}`
	code, resp := wlDo(APIUpdateWishlist, "PUT", "/api/auth/wishlists/"+wishlist.ID.String(), body, authHeader(t, collaborator.ID, false), wishlistTestParams(wishlist.ID))
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}
}

func TestAPIUpdateWishlistKeepsPublicLinkOnEdit(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := wishlistTestSave(t, createTestWishlist(t, owner.ID), func(w *models.Wishlist) {
		w.Public = boolPtr(true)
		w.PublicHash = uuid.New()
	})

	body := `{"name":"` + wishlist.Name + `","description":"Only the description changed","public":true}`
	code, resp := wlDo(APIUpdateWishlist, "PUT", "/api/auth/wishlists/"+wishlist.ID.String(), body, authHeader(t, owner.ID, false), wishlistTestParams(wishlist.ID))
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}

	updated, err := database.GetWishlist(wishlist.ID)
	if err != nil {
		t.Fatalf("failed to reload wishlist: %v", err)
	}
	if updated.PublicHash != wishlist.PublicHash {
		t.Errorf("public hash changed from %v to %v; an ordinary edit must keep shared links working", wishlist.PublicHash, updated.PublicHash)
	}
}

func TestAPIUpdateWishlistRepublishingIssuesNewPublicLink(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := wishlistTestSave(t, createTestWishlist(t, owner.ID), func(w *models.Wishlist) {
		w.Public = boolPtr(false)
		w.PublicHash = uuid.New()
	})

	body := `{"name":"` + wishlist.Name + `","public":true}`
	code, resp := wlDo(APIUpdateWishlist, "PUT", "/api/auth/wishlists/"+wishlist.ID.String(), body, authHeader(t, owner.ID, false), wishlistTestParams(wishlist.ID))
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}

	updated, err := database.GetWishlist(wishlist.ID)
	if err != nil {
		t.Fatalf("failed to reload wishlist: %v", err)
	}
	if updated.PublicHash == wishlist.PublicHash {
		t.Error("public hash unchanged; re-publishing a private wishlist must revoke the old link")
	}
	if updated.Public == nil || !*updated.Public {
		t.Error("wishlist should now be public")
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

func TestWishlistHandlersDatabaseErrors(t *testing.T) {
	wishlistParam := gin.Params{{Key: "wishlist_id", Value: "00000000-0000-0000-0000-00000000000a"}}
	runDatabaseErrorCases(t, []dbErrorCase{
		{name: "RegisterWishlist", handler: RegisterWishlist, method: "POST", path: "/api/auth/wishlists", body: `{"name":"Birthday list","description":"Desc","date":"` + wlFutureDate() + `"}`},
		{name: "DeleteWishlist", handler: DeleteWishlist, method: "DELETE", path: "/api/auth/wishlists/00000000-0000-0000-0000-00000000000a", params: wishlistParam},
		{name: "GetWishlist", handler: GetWishlist, method: "GET", path: "/api/auth/wishlists/00000000-0000-0000-0000-00000000000a", params: wishlistParam},
		{name: "GetWishlists", handler: GetWishlists, method: "GET", path: "/api/auth/wishlists"},
		{name: "GetWishlists/group", handler: GetWishlists, method: "GET", path: "/api/auth/wishlists?group=00000000-0000-0000-0000-00000000000a"},
		{name: "APICollaborateWishlist", handler: APICollaborateWishlist, method: "POST", path: "/api/auth/wishlists/00000000-0000-0000-0000-00000000000a/collaborate", body: `{"users":["00000000-0000-0000-0000-00000000000a"]}`, params: wishlistParam},
		{name: "GetWishlists/notAMemberOfGroupID", handler: GetWishlists, method: "GET", path: "/api/auth/wishlists?notAMemberOfGroupID=00000000-0000-0000-0000-00000000000a"},
		{name: "JoinWishlist", handler: JoinWishlist, method: "POST", path: "/api/auth/wishlists/00000000-0000-0000-0000-00000000000a/join", body: `{"groups":["00000000-0000-0000-0000-00000000000a"]}`, params: wishlistParam},
		{name: "RemoveFromWishlist", handler: RemoveFromWishlist, method: "POST", path: "/api/auth/wishlists/00000000-0000-0000-0000-00000000000a/remove", body: `{"group_id":"00000000-0000-0000-0000-00000000000a"}`, params: wishlistParam},
		{name: "APIUpdateWishlist", handler: APIUpdateWishlist, method: "POST", path: "/api/auth/wishlists/00000000-0000-0000-0000-00000000000a", body: `{"name":"Birthday list","description":"Desc","date":"` + wlFutureDate() + `"}`, params: wishlistParam},
		{name: "APIUnCollaborateWishlist", handler: APIUnCollaborateWishlist, method: "POST", path: "/api/auth/wishlists/00000000-0000-0000-0000-00000000000a/un-collaborate", body: `{"user_id":"00000000-0000-0000-0000-00000000000a"}`, params: wishlistParam},
		{name: "GetPublicWishlist", handler: GetPublicWishlist, method: "GET", path: "/api/open/wishlists/public/00000000-0000-0000-0000-00000000000a", params: gin.Params{{Key: "wishlist_hash", Value: "00000000-0000-0000-0000-00000000000a"}}},
	})
}

func TestWishlistHandlersRequireAuth(t *testing.T) {
	wishlistParam := gin.Params{{Key: "wishlist_id", Value: "00000000-0000-0000-0000-00000000000a"}}
	runUnauthenticatedCases(t, []dbErrorCase{
		{name: "DeleteWishlist", handler: DeleteWishlist, method: "DELETE", path: "/api/auth/wishlists/00000000-0000-0000-0000-00000000000a", params: wishlistParam},
		{name: "JoinWishlist", handler: JoinWishlist, method: "POST", path: "/api/auth/wishlists/00000000-0000-0000-0000-00000000000a/join", body: `{"groups":["00000000-0000-0000-0000-00000000000a"]}`, params: wishlistParam},
		{name: "APICollaborateWishlist", handler: APICollaborateWishlist, method: "POST", path: "/api/auth/wishlists/00000000-0000-0000-0000-00000000000a/collaborate", body: `{"users":["00000000-0000-0000-0000-00000000000a"]}`, params: wishlistParam},
		{name: "APIUnCollaborateWishlist", handler: APIUnCollaborateWishlist, method: "POST", path: "/api/auth/wishlists/00000000-0000-0000-0000-00000000000a/un-collaborate", body: `{"user_id":"00000000-0000-0000-0000-00000000000a"}`, params: wishlistParam},
	})
}

// --- deeper-branch coverage: helpers ---

func wishlistTestParams(id uuid.UUID) gin.Params {
	return gin.Params{{Key: "wishlist_id", Value: id.String()}}
}

// wishlistTestSave applies mutate to wishlist and persists it, for fixtures
// createTestWishlist can't express (expired, public, ...).
func wishlistTestSave(t *testing.T, wishlist models.Wishlist, mutate func(*models.Wishlist)) models.Wishlist {
	t.Helper()
	mutate(&wishlist)
	updated, err := database.UpdateWishlistInDB(wishlist)
	if err != nil {
		t.Fatalf("failed to update wishlist: %v", err)
	}
	return updated
}

// wishlistTestList pulls resp["wishlists"] out as a slice, failing if absent.
func wishlistTestList(t *testing.T, resp map[string]interface{}) []interface{} {
	t.Helper()
	list, ok := resp["wishlists"].([]interface{})
	if !ok {
		t.Fatalf("wishlists = %v, want a list", resp["wishlists"])
	}
	return list
}

// wishlistTestIDs returns the "id" of every wishlist object in list.
func wishlistTestIDs(list []interface{}) map[string]bool {
	ids := map[string]bool{}
	for _, item := range list {
		if obj, ok := item.(map[string]interface{}); ok {
			ids[obj["id"].(string)] = true
		}
	}
	return ids
}

// --- RegisterWishlist deeper branches ---

func TestRegisterWishlistInvalidNameCharacters(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists", `{"name":"Bad <name> list"}`, authHeader(t, owner.ID, false), nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
	if owned, _ := database.GetOwnedWishlists(owner.ID); len(owned) != 0 {
		t.Errorf("owned wishlists = %v, want none", owned)
	}
}

func TestRegisterWishlistGroupMembershipCheckDBFailure(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	failDBOperation(t, "query", "group_memberships", 0)

	body := `{"name":"Shared List","groups":["` + group.ID.String() + `"]}`
	code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists", body, authHeader(t, owner.ID, false), nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
}

func TestRegisterWishlistCreateDBFailure(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	failDBOperation(t, "create", "wishlists", 0)

	code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists", `{"name":"Birthday list"}`, authHeader(t, owner.ID, false), nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
	if owned, _ := database.GetOwnedWishlists(owner.ID); len(owned) != 0 {
		t.Errorf("owned wishlists = %v, want none after a failed create", owned)
	}
}

func TestRegisterWishlistMembershipCreateDBFailure(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	failDBOperation(t, "create", "wishlist_memberships", 0)

	body := `{"name":"Shared List","groups":["` + group.ID.String() + `"]}`
	code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists", body, authHeader(t, owner.ID, false), nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
}

func TestRegisterWishlistGroupContextListDBFailure(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	// First wishlists query is the unique-name check; the second is the
	// group-context re-list.
	failDBOperation(t, "query", "wishlists", 1)

	code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists?groupContextID="+group.ID.String(), `{"name":"Birthday list"}`, authHeader(t, owner.ID, false), nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
}

func TestRegisterWishlistListDBFailure(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	failDBOperation(t, "query", "wishlists", 1)

	code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists", `{"name":"Birthday list"}`, authHeader(t, owner.ID, false), nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
}

func TestRegisterWishlistReturnsNewestFirst(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	existing := createTestWishlist(t, owner.ID)
	time.Sleep(5 * time.Millisecond)

	code, resp := wlDo(RegisterWishlist, "POST", "/api/auth/wishlists", `{"name":"Newest list"}`, authHeader(t, owner.ID, false), nil)
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}
	list := wishlistTestList(t, resp)
	if len(list) != 2 {
		t.Fatalf("wishlists = %v, want 2", list)
	}
	if list[0].(map[string]interface{})["name"] != "Newest list" || list[1].(map[string]interface{})["id"] != existing.ID.String() {
		t.Errorf("wishlists = %v, want the new wishlist first, sorted by creation date", list)
	}
}

// --- DeleteWishlist deeper branches ---

func TestDeleteWishlistUpdateDBFailure(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	failDBOperation(t, "update", "wishlists", 0)

	code, resp := wlDo(DeleteWishlist, "DELETE", "/api/auth/wishlists/"+wishlist.ID.String(), "", authHeader(t, owner.ID, false), wishlistTestParams(wishlist.ID))
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
	if resp["error"] != "Failed to delete wishlist." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestDeleteWishlistListDBFailure(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	// First wishlists query is the ownership check, second is the re-list.
	failDBOperation(t, "query", "wishlists", 1)

	code, resp := wlDo(DeleteWishlist, "DELETE", "/api/auth/wishlists/"+wishlist.ID.String(), "", authHeader(t, owner.ID, false), wishlistTestParams(wishlist.ID))
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
}

func TestDeleteWishlistGroupQueryMembershipDBFailure(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	group := createTestGroup(t, owner.ID)
	failDBOperation(t, "query", "group_memberships", 0)

	code, resp := wlDo(DeleteWishlist, "DELETE", "/api/auth/wishlists/"+wishlist.ID.String()+"?group="+group.ID.String(), "", authHeader(t, owner.ID, false), wishlistTestParams(wishlist.ID))
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
}

func TestDeleteWishlistGroupQueryListDBFailure(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	failDBOperation(t, "query", "wishlists", 1)

	code, resp := wlDo(DeleteWishlist, "DELETE", "/api/auth/wishlists/"+wishlist.ID.String()+"?group="+group.ID.String(), "", authHeader(t, owner.ID, false), wishlistTestParams(wishlist.ID))
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
	if resp["error"] != "Failed to get wishlists for group." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestDeleteWishlistRemainingSortedNewestFirst(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	older := createTestWishlist(t, owner.ID)
	time.Sleep(5 * time.Millisecond)
	newer := createTestWishlist(t, owner.ID)
	doomed := createTestWishlist(t, owner.ID)

	code, resp := wlDo(DeleteWishlist, "DELETE", "/api/auth/wishlists/"+doomed.ID.String(), "", authHeader(t, owner.ID, false), wishlistTestParams(doomed.ID))
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	list := wishlistTestList(t, resp)
	if len(list) != 2 || list[0].(map[string]interface{})["id"] != newer.ID.String() || list[1].(map[string]interface{})["id"] != older.ID.String() {
		t.Errorf("wishlists = %v, want [newer, older]", list)
	}
}

// wishlistTestImageDir points wishImageDir at a fresh temp dir for the test.
func wishlistTestImageDir(t *testing.T) string {
	t.Helper()
	orig := wishImageDir
	t.Cleanup(func() { wishImageDir = orig })
	wishImageDir = t.TempDir()
	return wishImageDir
}

func TestDeleteWishlistImageCleanupFailuresAreNotFatal(t *testing.T) {
	t.Run("wish lookup fails", func(t *testing.T) {
		setupControllersDB(t)
		owner := createTestUser(t)
		wishlist := createTestWishlist(t, owner.ID)
		createTestWish(t, owner.ID, wishlist.ID)
		failDBOperation(t, "query", "wishes", 0)

		code, resp := wlDo(DeleteWishlist, "DELETE", "/api/auth/wishlists/"+wishlist.ID.String(), "", authHeader(t, owner.ID, false), wishlistTestParams(wishlist.ID))
		if code != 200 {
			t.Fatalf("status = %d, want 200 (cleanup failure is only logged); body=%v", code, resp)
		}
		if _, err := database.GetWishlist(wishlist.ID); err == nil {
			t.Errorf("wishlist still enabled after delete")
		}
	})

	t.Run("image removal fails", func(t *testing.T) {
		setupControllersDB(t)
		dir := wishlistTestImageDir(t)
		owner := createTestUser(t)
		wishlist := createTestWishlist(t, owner.ID)
		wish := createTestWish(t, owner.ID, wishlist.ID)
		// A non-empty directory where the image file should be makes
		// os.Remove fail with something other than "not exist".
		blocker := imageFilePath(dir, wish.ID, false)
		if err := os.MkdirAll(filepath.Join(blocker, "child"), 0755); err != nil {
			t.Fatalf("failed to create blocker dir: %v", err)
		}

		code, resp := wlDo(DeleteWishlist, "DELETE", "/api/auth/wishlists/"+wishlist.ID.String(), "", authHeader(t, owner.ID, false), wishlistTestParams(wishlist.ID))
		if code != 200 {
			t.Fatalf("status = %d, want 200 (cleanup failure is only logged); body=%v", code, resp)
		}
		if _, err := os.Stat(blocker); err != nil {
			t.Errorf("blocker unexpectedly removed: %v", err)
		}
	})
}

// --- GetWishlist / GetWishlistObject deeper branches ---

func TestGetWishlistGroupMembershipCheckDBFailure(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	failDBOperation(t, "query", "wishlist_memberships", 0)

	code, resp := wlDo(GetWishlist, "GET", "/api/auth/wishlists/"+wishlist.ID.String(), "", authHeader(t, owner.ID, false), wishlistTestParams(wishlist.ID))
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
}

func TestGetWishlistOwnerDisabled(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	header := authHeader(t, owner.ID, false)
	owner.Enabled = boolPtr(false)
	if _, err := database.UpdateUserInDB(owner); err != nil {
		t.Fatalf("failed to disable owner: %v", err)
	}

	// Ownership is still recorded, but building the object needs the (now
	// disabled) owner's profile.
	code, resp := wlDo(GetWishlist, "GET", "/api/auth/wishlists/"+wishlist.ID.String(), "", header, wishlistTestParams(wishlist.ID))
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
	if resp["error"] != "Failed to get wishlist object." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestGetWishlistObjectUnknownWishlist(t *testing.T) {
	setupControllersDB(t)
	_, err := GetWishlistObject(uuid.New(), uuid.New())
	if err != database.ErrWishlistNotFound {
		t.Errorf("err = %v, want ErrWishlistNotFound", err)
	}
}

// --- GetWishlists deeper branches ---

func TestGetWishlistsGroupFilterListDBFailure(t *testing.T) {
	setupControllersDB(t)
	member := createTestUser(t)
	group := createTestGroup(t, member.ID)
	addGroupMembership(t, group.ID, member.ID)
	failDBOperation(t, "query", "wishlists", 0)

	code, resp := wlDo(GetWishlists, "GET", "/api/auth/wishlists?group="+group.ID.String(), "", authHeader(t, member.ID, false), nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
	if resp["error"] != "Failed to get wishlists for group." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestGetWishlistsOwnedFilterIncludesOwnedAndCollaborations(t *testing.T) {
	setupControllersDB(t)
	caller := createTestUser(t)
	other := createTestUser(t)
	bystander := createTestUser(t)
	group := createTestGroup(t, other.ID)
	addGroupMembership(t, group.ID, caller.ID)

	owned := createTestWishlist(t, caller.ID)
	collab := createTestWishlist(t, other.ID)
	addWishCollaborator(t, collab.ID, caller.ID)
	// Visible via the group and has a collaborator, just not the caller.
	groupOnly := createTestWishlist(t, other.ID)
	addWishlistMembership(t, groupOnly.ID, group.ID)
	addWishCollaborator(t, groupOnly.ID, bystander.ID)

	code, resp := wlDo(GetWishlists, "GET", "/api/auth/wishlists?owned=true", "", authHeader(t, caller.ID, false), nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	ids := wishlistTestIDs(wishlistTestList(t, resp))
	if len(ids) != 2 || !ids[owned.ID.String()] || !ids[collab.ID.String()] {
		t.Errorf("wishlists = %v, want exactly the owned and collaborated wishlists", ids)
	}
}

func TestGetWishlistsNotAMemberOfGroupFilterDBFailure(t *testing.T) {
	setupControllersDB(t)
	caller := createTestUser(t)
	// GetAllWishlistObjects makes three wishlists queries (owned, collab,
	// membership) before the notAMemberOfGroupID lookup.
	failDBOperation(t, "query", "wishlists", 3)

	code, resp := wlDo(GetWishlists, "GET", "/api/auth/wishlists?notAMemberOfGroupID="+uuid.NewString(), "", authHeader(t, caller.ID, false), nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
}

func TestGetWishlistsExpiredTrueFilter(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	past := time.Now().Add(-72 * time.Hour)
	expired := wishlistTestSave(t, createTestWishlist(t, owner.ID), func(w *models.Wishlist) {
		w.Expires = boolPtr(true)
		w.Date = &past
	})
	future := time.Now().Add(72 * time.Hour)
	wishlistTestSave(t, createTestWishlist(t, owner.ID), func(w *models.Wishlist) {
		w.Expires = boolPtr(true)
		w.Date = &future
	})

	code, resp := wlDo(GetWishlists, "GET", "/api/auth/wishlists?expired=TRUE", "", authHeader(t, owner.ID, false), nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	ids := wishlistTestIDs(wishlistTestList(t, resp))
	if len(ids) != 1 || !ids[expired.ID.String()] {
		t.Errorf("wishlists = %v, want only the expired wishlist", ids)
	}
}

// --- GetWishlistObjects / GetAllWishlistObjects deeper branches ---

func TestGetWishlistObjectsCollaborationQueryDBFailure(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	failDBOperation(t, "query", "wishlists", 1)

	if _, err := GetWishlistObjects(user.ID); err == nil {
		t.Fatal("expected an error when the collaboration lookup fails")
	}
}

func TestGetAllWishlistObjectsDBFailures(t *testing.T) {
	cases := []struct {
		name string
		skip int
	}{
		{"collaborations", 1},
		{"memberships", 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			user := createTestUser(t)
			failDBOperation(t, "query", "wishlists", c.skip)

			objects, err := GetAllWishlistObjects(user.ID)
			if err == nil {
				t.Fatal("expected an error")
			}
			if len(objects) != 0 {
				t.Errorf("objects = %v, want none on error", objects)
			}
		})
	}
}

func TestGetAllWishlistObjectsDeduplicatesOwnedCollaborations(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	other := createTestUser(t)
	own := createTestWishlist(t, user.ID)
	foreign := createTestWishlist(t, other.ID)
	// A collaborator row on the user's own wishlist (only possible via the DB)
	// must not duplicate it; the foreign one must be added.
	addWishCollaborator(t, own.ID, user.ID)
	addWishCollaborator(t, foreign.ID, user.ID)

	objects, err := GetAllWishlistObjects(user.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(objects) != 2 {
		t.Errorf("objects = %d, want 2 (own deduplicated + foreign)", len(objects))
	}
}

// --- JoinWishlist deeper branches ---

func TestJoinWishlistDBFailures(t *testing.T) {
	cases := []struct {
		name  string
		op    string
		table string
		skip  int
	}{
		{"membership check", "query", "wishlist_memberships", 0},
		{"ownership check", "query", "wishlists", 0},
		{"membership create", "create", "wishlist_memberships", 0},
		{"re-list", "query", "wishlists", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			owner := createTestUser(t)
			wishlist := createTestWishlist(t, owner.ID)
			group := createTestGroup(t, owner.ID)
			failDBOperation(t, c.op, c.table, c.skip)

			body := `{"groups":["` + group.ID.String() + `"]}`
			code, resp := wlDo(JoinWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/join", body, authHeader(t, owner.ID, false), wishlistTestParams(wishlist.ID))
			if code != 500 {
				t.Fatalf("status = %d, want 500; body=%v", code, resp)
			}
		})
	}
}

func TestJoinWishlistReturnsNewestFirst(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	older := createTestWishlist(t, owner.ID)
	time.Sleep(5 * time.Millisecond)
	newer := createTestWishlist(t, owner.ID)
	group := createTestGroup(t, owner.ID)

	body := `{"groups":["` + group.ID.String() + `"]}`
	code, resp := wlDo(JoinWishlist, "POST", "/api/auth/wishlists/"+older.ID.String()+"/join", body, authHeader(t, owner.ID, false), wishlistTestParams(older.ID))
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}
	list := wishlistTestList(t, resp)
	if len(list) != 2 || list[0].(map[string]interface{})["id"] != newer.ID.String() {
		t.Errorf("wishlists = %v, want the newer wishlist first", list)
	}
	if ok, _ := database.VerifyGroupMembershipToWishlist(older.ID, group.ID); !ok {
		t.Error("wishlist membership was not created")
	}
}

// --- RemoveFromWishlist deeper branches ---

func TestRemoveFromWishlistDBFailures(t *testing.T) {
	cases := []struct {
		name     string
		op       string
		table    string
		skip     int
		query    string
		inGroup  bool
		wantCode int
	}{
		{name: "ownership check", op: "query", table: "wishlists", wantCode: 500},
		{name: "membership lookup", op: "query", table: "wishlist_memberships", skip: 1, wantCode: 500},
		{name: "membership delete", op: "update", table: "wishlist_memberships", wantCode: 500},
		{name: "group membership check", op: "query", table: "group_memberships", query: "?group=", wantCode: 500},
		{name: "group re-list", op: "query", table: "wishlists", skip: 1, query: "?group=", inGroup: true, wantCode: 500},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			owner := createTestUser(t)
			wishlist := createTestWishlist(t, owner.ID)
			group := createTestGroup(t, owner.ID)
			addWishlistMembership(t, wishlist.ID, group.ID)
			if c.inGroup {
				addGroupMembership(t, group.ID, owner.ID)
			}
			query := c.query
			if query != "" {
				query += group.ID.String()
			}
			failDBOperation(t, c.op, c.table, c.skip)

			body := `{"group_id":"` + group.ID.String() + `"}`
			code, resp := wlDo(RemoveFromWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/remove"+query, body, authHeader(t, owner.ID, false), wishlistTestParams(wishlist.ID))
			if code != c.wantCode {
				t.Fatalf("status = %d, want %d; body=%v", code, c.wantCode, resp)
			}
		})
	}
}

func TestRemoveFromWishlistGroupQueryBadUUID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	group := createTestGroup(t, owner.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)

	body := `{"group_id":"` + group.ID.String() + `"}`
	code, resp := wlDo(RemoveFromWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/remove?group=not-a-uuid", body, authHeader(t, owner.ID, false), wishlistTestParams(wishlist.ID))
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
	if resp["error"] != "Failed to parse group ID." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestRemoveFromWishlistReturnsNewestFirst(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	older := createTestWishlist(t, owner.ID)
	time.Sleep(5 * time.Millisecond)
	newer := createTestWishlist(t, owner.ID)
	group := createTestGroup(t, owner.ID)
	addWishlistMembership(t, older.ID, group.ID)

	body := `{"group_id":"` + group.ID.String() + `"}`
	code, resp := wlDo(RemoveFromWishlist, "POST", "/api/auth/wishlists/"+older.ID.String()+"/remove", body, authHeader(t, owner.ID, false), wishlistTestParams(older.ID))
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}
	list := wishlistTestList(t, resp)
	if len(list) != 2 || list[0].(map[string]interface{})["id"] != newer.ID.String() {
		t.Errorf("wishlists = %v, want the newer wishlist first", list)
	}
	if ok, _ := database.VerifyGroupMembershipToWishlist(older.ID, group.ID); ok {
		t.Error("wishlist membership still present")
	}
}

// --- APIUpdateWishlist deeper branches ---

func TestAPIUpdateWishlistDBFailures(t *testing.T) {
	cases := []struct {
		name  string
		op    string
		table string
		skip  int
		body  string
	}{
		{"ownership check", "query", "wishlists", 0, `{"name":"Renamed List"}`},
		{"original lookup", "query", "wishlists", 1, `{"name":"Renamed List"}`},
		{"unique name check", "query", "wishlists", 2, `{"name":"Renamed List"}`},
		{"save", "update", "wishlists", 0, `{"name":"Renamed List"}`},
		{"object conversion", "query", "groups", 0, `{"name":"Renamed List"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			owner := createTestUser(t)
			wishlist := createTestWishlist(t, owner.ID)
			failDBOperation(t, c.op, c.table, c.skip)

			code, resp := wlDo(APIUpdateWishlist, "PUT", "/api/auth/wishlists/"+wishlist.ID.String(), c.body, authHeader(t, owner.ID, false), wishlistTestParams(wishlist.ID))
			if code != 500 {
				t.Fatalf("status = %d, want 500; body=%v", code, resp)
			}
		})
	}
}

func TestAPIUpdateWishlistInvalidNameCharacters(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	code, resp := wlDo(APIUpdateWishlist, "PUT", "/api/auth/wishlists/"+wishlist.ID.String(), `{"name":"Bad <name>"}`, authHeader(t, owner.ID, false), wishlistTestParams(wishlist.ID))
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
	if got, _ := database.GetWishlist(wishlist.ID); got.Name != wishlist.Name {
		t.Errorf("name = %q, want it unchanged", got.Name)
	}
}

func TestAPIUpdateWishlistDuplicateName(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	other := createTestWishlist(t, owner.ID)

	body := `{"name":"` + other.Name + `"}`
	code, resp := wlDo(APIUpdateWishlist, "PUT", "/api/auth/wishlists/"+wishlist.ID.String(), body, authHeader(t, owner.ID, false), wishlistTestParams(wishlist.ID))
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
	if resp["error"] != "There is already a wishlist with that name on your profile." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestAPIUpdateWishlistSetsDate(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	date := wlFutureDate()

	body := `{"name":"` + wishlist.Name + `","expires":true,"date":"` + date + `"}`
	code, resp := wlDo(APIUpdateWishlist, "PUT", "/api/auth/wishlists/"+wishlist.ID.String(), body, authHeader(t, owner.ID, false), wishlistTestParams(wishlist.ID))
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}
	got, err := database.GetWishlist(wishlist.ID)
	if err != nil {
		t.Fatalf("failed to reload wishlist: %v", err)
	}
	want, _ := time.Parse("2006-01-02T15:04:05.000Z", date)
	if got.Date == nil || !got.Date.Equal(want) || got.Expires == nil || !*got.Expires {
		t.Errorf("date = %v, expires = %v; want %v, true", got.Date, got.Expires, want)
	}
}

// --- ConvertWishlist*/collaborator conversion branches ---

func TestConvertWishlistCollaboratorsSkipsUnknownUsers(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	wishlistID := uuid.New()
	collabs := []models.WishlistCollaborator{
		{UserID: uuid.New(), WishlistID: wishlistID},
		{UserID: user.ID, WishlistID: wishlistID},
	}

	objects, err := ConvertWishlistCollaboratorsToWishlistCollaboratorsObjects(collabs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(objects) != 1 || objects[0].User.ID != user.ID {
		t.Errorf("objects = %v, want only the existing user", objects)
	}
}

func TestConvertWishlistToWishlistObjectDBFailures(t *testing.T) {
	cases := []struct {
		name  string
		table string
	}{
		{"groups", "groups"},
		{"collaborators", "wishlist_collaborators"},
		{"owner and collaborator users", "users"},
		{"wishes", "wishes"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			owner := createTestUser(t)
			collaborator := createTestUser(t)
			wishlist := createTestWishlist(t, owner.ID)
			addWishCollaborator(t, wishlist.ID, collaborator.ID)
			failDBOperation(t, "query", c.table, 0)

			if _, err := ConvertWishlistToWishlistObject(wishlist, &owner.ID); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestConvertWishlistToWishlistObjectWishUpdatedAt(t *testing.T) {
	t.Run("latest wish is newer than wishlist", func(t *testing.T) {
		setupControllersDB(t)
		owner := createTestUser(t)
		wishlist := createTestWishlist(t, owner.ID)
		time.Sleep(5 * time.Millisecond)
		createTestWish(t, owner.ID, wishlist.ID)
		time.Sleep(5 * time.Millisecond)
		latest := createTestWish(t, owner.ID, wishlist.ID)

		object, err := ConvertWishlistToWishlistObject(wishlist, &owner.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(object.Wishes) != 2 || object.Wishes[0].ID != latest.ID {
			t.Fatalf("wishes = %v, want 2 with the latest first", object.Wishes)
		}
		if !object.WishUpdatedAt.Equal(object.Wishes[0].UpdatedAt) {
			t.Errorf("WishUpdatedAt = %v, want the latest wish's %v", object.WishUpdatedAt, object.Wishes[0].UpdatedAt)
		}
	})

	t.Run("wishlist is newer than its wishes", func(t *testing.T) {
		setupControllersDB(t)
		owner := createTestUser(t)
		wishlist := createTestWishlist(t, owner.ID)
		createTestWish(t, owner.ID, wishlist.ID)
		time.Sleep(5 * time.Millisecond)
		wishlist = wishlistTestSave(t, wishlist, func(w *models.Wishlist) { w.Description = "touched" })

		object, err := ConvertWishlistToWishlistObject(wishlist, &owner.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(object.Wishes) != 1 {
			t.Fatalf("wishes = %v, want 1", object.Wishes)
		}
		if !object.WishUpdatedAt.Equal(wishlist.UpdatedAt) {
			t.Errorf("WishUpdatedAt = %v, want the wishlist's %v", object.WishUpdatedAt, wishlist.UpdatedAt)
		}
	})
}

func TestConvertWishlistsToWishlistObjectsSkipsFailures(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	good := createTestWishlist(t, owner.ID)
	orphan := models.Wishlist{OwnerID: uuid.New()}
	orphan.ID = uuid.New()

	objects, err := ConvertWishlistsToWishlistObjects([]models.Wishlist{orphan, good}, &owner.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(objects) != 1 || objects[0].ID != good.ID {
		t.Errorf("objects = %v, want only the wishlist whose owner exists", objects)
	}
}

// --- APICollaborateWishlist / APIUnCollaborateWishlist deeper branches ---

func TestAPICollaborateWishlistDBFailures(t *testing.T) {
	cases := []struct {
		name  string
		op    string
		table string
		skip  int
	}{
		{"collaboration check", "query", "wishlist_collaborators", 0},
		{"owner lookup", "query", "wishlists", 0},
		{"collaborator create", "create", "wishlist_collaborators", 0},
		{"re-list", "query", "wishlists", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			owner := createTestUser(t)
			collaborator := createTestUser(t)
			wishlist := createTestWishlist(t, owner.ID)
			failDBOperation(t, c.op, c.table, c.skip)

			body := `{"users":["` + collaborator.ID.String() + `"]}`
			code, resp := wlDo(APICollaborateWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/collaborate", body, authHeader(t, owner.ID, false), wishlistTestParams(wishlist.ID))
			if code != 500 {
				t.Fatalf("status = %d, want 500; body=%v", code, resp)
			}
		})
	}
}

func TestAPICollaborateWishlistReturnsNewestFirst(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	collaborator := createTestUser(t)
	older := createTestWishlist(t, owner.ID)
	time.Sleep(5 * time.Millisecond)
	newer := createTestWishlist(t, owner.ID)

	body := `{"users":["` + collaborator.ID.String() + `"]}`
	code, resp := wlDo(APICollaborateWishlist, "POST", "/api/auth/wishlists/"+older.ID.String()+"/collaborate", body, authHeader(t, owner.ID, false), wishlistTestParams(older.ID))
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}
	list := wishlistTestList(t, resp)
	if len(list) != 2 || list[0].(map[string]interface{})["id"] != newer.ID.String() {
		t.Errorf("wishlists = %v, want the newer wishlist first", list)
	}
}

func TestAPIUnCollaborateWishlistDBFailures(t *testing.T) {
	cases := []struct {
		name  string
		op    string
		table string
		skip  int
	}{
		{"owner lookup", "query", "wishlists", 0},
		{"collaboration lookup", "query", "wishlist_collaborators", 1},
		{"collaboration delete", "update", "wishlist_collaborators", 0},
		{"re-list", "query", "wishlists", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			owner := createTestUser(t)
			collaborator := createTestUser(t)
			wishlist := createTestWishlist(t, owner.ID)
			addWishCollaborator(t, wishlist.ID, collaborator.ID)
			failDBOperation(t, c.op, c.table, c.skip)

			body := `{"user_id":"` + collaborator.ID.String() + `"}`
			code, resp := wlDo(APIUnCollaborateWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/un-collaborate", body, authHeader(t, owner.ID, false), wishlistTestParams(wishlist.ID))
			if code != 500 {
				t.Fatalf("status = %d, want 500; body=%v", code, resp)
			}
		})
	}
}

func TestAPIUnCollaborateWishlistReturnsNewestFirst(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	collaborator := createTestUser(t)
	older := createTestWishlist(t, owner.ID)
	time.Sleep(5 * time.Millisecond)
	newer := createTestWishlist(t, owner.ID)
	addWishCollaborator(t, older.ID, collaborator.ID)

	body := `{"user_id":"` + collaborator.ID.String() + `"}`
	code, resp := wlDo(APIUnCollaborateWishlist, "POST", "/api/auth/wishlists/"+older.ID.String()+"/un-collaborate", body, authHeader(t, owner.ID, false), wishlistTestParams(older.ID))
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}
	list := wishlistTestList(t, resp)
	if len(list) != 2 || list[0].(map[string]interface{})["id"] != newer.ID.String() {
		t.Errorf("wishlists = %v, want the newer wishlist first", list)
	}
	if ok, _ := database.VerifyWishlistCollaboratorToWishlist(older.ID, collaborator.ID); ok {
		t.Error("collaboration still present")
	}
}

// --- GetPublicWishlist deeper branches ---

func wishlistTestPublic(t *testing.T, ownerID uuid.UUID) models.Wishlist {
	t.Helper()
	return wishlistTestSave(t, createTestWishlist(t, ownerID), func(w *models.Wishlist) {
		w.Public = boolPtr(true)
		w.Claimable = boolPtr(false)
		w.PublicHash = uuid.New()
	})
}

func TestGetPublicWishlistConversionDBFailure(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := wishlistTestPublic(t, owner.ID)
	failDBOperation(t, "query", "groups", 0)

	code, resp := wlDo(GetPublicWishlist, "GET", "/api/open/wishlists/public/"+wishlist.PublicHash.String(), "", "",
		gin.Params{{Key: "wishlist_hash", Value: wishlist.PublicHash.String()}})
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
}

func TestGetPublicWishlistWishesNewestFirst(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := wishlistTestPublic(t, owner.ID)
	createTestWish(t, owner.ID, wishlist.ID)
	time.Sleep(5 * time.Millisecond)
	latest := createTestWish(t, owner.ID, wishlist.ID)

	code, resp := wlDo(GetPublicWishlist, "GET", "/api/open/wishlists/public/"+wishlist.PublicHash.String(), "", "",
		gin.Params{{Key: "wishlist_hash", Value: wishlist.PublicHash.String()}})
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	obj, _ := resp["wishlist"].(map[string]interface{})
	wishes, _ := obj["wishes"].([]interface{})
	if len(wishes) != 2 || wishes[0].(map[string]interface{})["id"] != latest.ID.String() {
		t.Errorf("wishes = %v, want 2 with the latest first", wishes)
	}
}

// A real database error while building the wishlist object is a 500; only a
// missing wishlist or owner (TestGetWishlistOwnerDisabled) is a 400.
func TestGetWishlistObjectDBFailure(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	header := authHeader(t, owner.ID, false)
	failDBOperation(t, "query", "users", 0)

	code, resp := wlDo(GetWishlist, "GET", "/api/auth/wishlists/"+wishlist.ID.String(), "", header, wishlistTestParams(wishlist.ID))
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
	if resp["error"] != "Failed to get wishlist object." {
		t.Errorf("error = %v", resp["error"])
	}
}

// The membership is verified, then looked up again to delete it; if it's
// removed in between, the caller gets a 400 rather than a 500.
func TestRemoveFromWishlistMembershipVanishes(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	group := createTestGroup(t, owner.ID)
	membership := addWishlistMembership(t, wishlist.ID, group.ID)
	onDBOperation(t, "query", "wishlist_memberships", 1, func(tx *gorm.DB) {
		if err := tx.Exec("UPDATE wishlist_memberships SET enabled = ? WHERE id = ?", false, membership.ID).Error; err != nil {
			t.Errorf("failed to disable membership: %v", err)
		}
	})

	body := `{"group_id":"` + group.ID.String() + `"}`
	code, resp := wlDo(RemoveFromWishlist, "POST", "/api/auth/wishlists/"+wishlist.ID.String()+"/remove", body, authHeader(t, owner.ID, false), wishlistTestParams(wishlist.ID))
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
	if resp["error"] != "Failed to find group membership ID." {
		t.Errorf("error = %v", resp["error"])
	}
}
