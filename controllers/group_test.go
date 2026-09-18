package controllers

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// groupTestContext builds a gin test context for a request with an optional
// JSON body. Path params must be set on the returned context by the caller.
func groupTestContext(method, path, body string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		ctx.Request.Header.Set("Content-Type", "application/json")
	}
	return ctx, w
}

func groupJSONBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse JSON response: %v; body=%s", err, w.Body.String())
	}
	return body
}

// --- RegisterGroup ---

func TestRegisterGroupSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	ctx, w := groupTestContext("POST", "/api/groups", `{"name":"My Group","description":"A group"}`)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterGroup(ctx)

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	body := groupJSONBody(t, w)
	if body["message"] != "Group created." {
		t.Errorf("message = %v", body["message"])
	}
	groups, ok := body["groups"].([]interface{})
	if !ok || len(groups) != 1 {
		t.Fatalf("groups = %v, want 1 group", body["groups"])
	}
}

func TestRegisterGroupMissingAuth(t *testing.T) {
	setupControllersDB(t)

	ctx, w := groupTestContext("POST", "/api/groups", `{"name":"My Group","description":"A group"}`)
	RegisterGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 without auth", w.Code)
	}
}

func TestRegisterGroupInvalidBody(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	ctx, w := groupTestContext("POST", "/api/groups", `not-json`)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for invalid JSON", w.Code)
	}
}

func TestRegisterGroupNameTooShort(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	ctx, w := groupTestContext("POST", "/api/groups", `{"name":"Abc","description":"A group"}`)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a too-short name", w.Code)
	}
}

func TestRegisterGroupDuplicateName(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	ctx1, w1 := groupTestContext("POST", "/api/groups", `{"name":"Duplicate Name","description":"A group"}`)
	ctx1.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterGroup(ctx1)
	if w1.Code != 201 {
		t.Fatalf("first create status = %d, want 201; body=%s", w1.Code, w1.Body.String())
	}

	ctx2, w2 := groupTestContext("POST", "/api/groups", `{"name":"Duplicate Name","description":"A group"}`)
	ctx2.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterGroup(ctx2)
	if w2.Code != 400 {
		t.Fatalf("second create status = %d, want 400 for a duplicate name", w2.Code)
	}
}

func TestRegisterGroupWithMembersAndWishlists(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	member := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	reqBody := `{"name":"Team Group","description":"desc","members":["` + member.ID.String() + `"],"wishlists":["` + wishlist.ID.String() + `"]}`
	ctx, w := groupTestContext("POST", "/api/groups", reqBody)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterGroup(ctx)

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	body := groupJSONBody(t, w)
	groups, ok := body["groups"].([]interface{})
	if !ok || len(groups) != 1 {
		t.Fatalf("groups = %v, want 1 group", body["groups"])
	}
	group := groups[0].(map[string]interface{})
	members, ok := group["members"].([]interface{})
	if !ok || len(members) != 2 { // owner + invited member
		t.Errorf("members = %v, want 2 members", group["members"])
	}
}

func TestRegisterGroupUnknownMember(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	reqBody := `{"name":"Team Group","description":"desc","members":["` + uuid.NewString() + `"]}`
	ctx, w := groupTestContext("POST", "/api/groups", reqBody)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for an unknown member", w.Code)
	}
}

func TestRegisterGroupWishlistNotOwned(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	otherOwner := createTestUser(t)
	wishlist := createTestWishlist(t, otherOwner.ID)

	reqBody := `{"name":"Team Group","description":"desc","wishlists":["` + wishlist.ID.String() + `"]}`
	ctx, w := groupTestContext("POST", "/api/groups", reqBody)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterGroup(ctx)

	if w.Code != 401 {
		t.Fatalf("status = %d, want 401 when the wishlist isn't owned by the caller", w.Code)
	}
}

// --- JoinGroup ---

func TestJoinGroupSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	newMember := createTestUser(t)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/join", `{"members":["`+newMember.ID.String()+`"]}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	JoinGroup(ctx)

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
}

func TestJoinGroupNoMembers(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/join", `{"members":[]}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	JoinGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for an empty members list", w.Code)
	}
}

func TestJoinGroupAlreadyMember(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	member := createTestUser(t)
	addGroupMembership(t, group.ID, member.ID)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/join", `{"members":["`+member.ID.String()+`"]}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	JoinGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 when already a member", w.Code)
	}
}

func TestJoinGroupUnknownUser(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/join", `{"members":["`+uuid.NewString()+`"]}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	JoinGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for an unknown user", w.Code)
	}
}

func TestJoinGroupInvalidGroupID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	member := createTestUser(t)

	ctx, w := groupTestContext("POST", "/api/groups/not-a-uuid/join", `{"members":["`+member.ID.String()+`"]}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: "not-a-uuid"}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	JoinGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed group ID", w.Code)
	}
}

// --- RemoveFromGroup ---

func TestRemoveFromGroupSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	member := createTestUser(t)
	addGroupMembership(t, group.ID, member.ID)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/remove", `{"member_id":"`+member.ID.String()+`"}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RemoveFromGroup(ctx)

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
}

func TestRemoveFromGroupNotOwner(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	member := createTestUser(t)
	addGroupMembership(t, group.ID, member.ID)
	stranger := createTestUser(t)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/remove", `{"member_id":"`+member.ID.String()+`"}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, stranger.ID, false))
	RemoveFromGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 when the caller doesn't own the group", w.Code)
	}
}

func TestRemoveFromGroupCannotRemoveOwner(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/remove", `{"member_id":"`+owner.ID.String()+`"}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RemoveFromGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 when trying to remove the owner", w.Code)
	}
}

func TestRemoveFromGroupMembershipMissing(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	notAMember := createTestUser(t)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/remove", `{"member_id":"`+notAMember.ID.String()+`"}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RemoveFromGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 when the target isn't a member", w.Code)
	}
}

// --- RemoveSelfFromGroup ---

// RemoveSelfFromGroup is currently broken for every caller, not just owners.
// It builds an unbound, zero-value models.GroupMembership{} and passes its
// MemberID (uuid.Nil) into database.VerifyUserOwnershipToGroup instead of the
// authenticated caller's ID. That function filters with a GORM struct Where
// clause (Instance.Where(&models.Group{OwnerID: UserID})), and GORM silently
// drops zero-value fields from a struct condition - so the OwnerID filter
// vanishes and the query degrades to "does an enabled group with this ID
// exist", which is true for any enabled group. The handler then always hits
// its "Owners cannot remove themselves as members." branch, so nobody can
// ever leave a group through this endpoint. This is a real bug (both tests
// below document current, broken behavior; not fixed here per instructions).
func TestRemoveSelfFromGroupAlwaysBlockedByBrokenOwnershipCheck(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	member := createTestUser(t)
	addGroupMembership(t, group.ID, member.ID)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/leave", "")
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, member.ID, false))
	RemoveSelfFromGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 (current, buggy behavior: a plain member can't leave either); body=%s", w.Code, w.Body.String())
	}
}

func TestRemoveSelfFromGroupNotAMember(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	stranger := createTestUser(t)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/leave", "")
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, stranger.ID, false))
	RemoveSelfFromGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 when the caller isn't a member", w.Code)
	}
}

// --- DeleteGroup ---

func TestDeleteGroupSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)

	ctx, w := groupTestContext("DELETE", "/api/groups/"+group.ID.String(), "")
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	DeleteGroup(ctx)

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteGroupNotOwner(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	stranger := createTestUser(t)

	ctx, w := groupTestContext("DELETE", "/api/groups/"+group.ID.String(), "")
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, stranger.ID, false))
	DeleteGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 when the caller doesn't own the group", w.Code)
	}
}

func TestDeleteGroupInvalidID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	ctx, w := groupTestContext("DELETE", "/api/groups/not-a-uuid", "")
	ctx.Params = gin.Params{{Key: "group_id", Value: "not-a-uuid"}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	DeleteGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed group ID", w.Code)
	}
}

// --- GetGroups ---

func TestGetGroupsSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	ownedGroup := createTestGroup(t, owner.ID)
	addGroupMembership(t, ownedGroup.ID, owner.ID)

	otherOwner := createTestUser(t)
	memberGroup := createTestGroup(t, otherOwner.ID)
	addGroupMembership(t, memberGroup.ID, otherOwner.ID)
	addGroupMembership(t, memberGroup.ID, owner.ID)

	ctx, w := groupTestContext("GET", "/api/groups", "")
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	GetGroups(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	body := groupJSONBody(t, w)
	groups, ok := body["groups"].([]interface{})
	if !ok || len(groups) != 2 {
		t.Fatalf("groups = %v, want 2 groups", body["groups"])
	}
}

func TestGetGroupsOwnedFilter(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	ownedGroup := createTestGroup(t, owner.ID)
	addGroupMembership(t, ownedGroup.ID, owner.ID)

	otherOwner := createTestUser(t)
	memberGroup := createTestGroup(t, otherOwner.ID)
	addGroupMembership(t, memberGroup.ID, otherOwner.ID)
	addGroupMembership(t, memberGroup.ID, owner.ID)

	ctx, w := groupTestContext("GET", "/api/groups?owned=true", "")
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	GetGroups(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	body := groupJSONBody(t, w)
	groups, ok := body["groups"].([]interface{})
	if !ok || len(groups) != 1 {
		t.Fatalf("groups = %v, want 1 owned group", body["groups"])
	}
}

func TestGetGroupsMemberOfWishlistFilter(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	wishlist := createTestWishlist(t, owner.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)

	otherGroup := createTestGroup(t, owner.ID)
	addGroupMembership(t, otherGroup.ID, owner.ID)

	ctx, w := groupTestContext("GET", "/api/groups?memberOfWishlistID="+wishlist.ID.String(), "")
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	GetGroups(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	body := groupJSONBody(t, w)
	groups, ok := body["groups"].([]interface{})
	if !ok || len(groups) != 1 {
		t.Fatalf("groups = %v, want 1 group linked to the wishlist", body["groups"])
	}
}

func TestGetGroupsNotAMemberOfWishlistFilter(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	wishlist := createTestWishlist(t, owner.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)

	otherGroup := createTestGroup(t, owner.ID)
	addGroupMembership(t, otherGroup.ID, owner.ID)

	ctx, w := groupTestContext("GET", "/api/groups?notAMemberOfWishlistID="+wishlist.ID.String(), "")
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	GetGroups(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	body := groupJSONBody(t, w)
	groups, ok := body["groups"].([]interface{})
	if !ok || len(groups) != 1 {
		t.Fatalf("groups = %v, want 1 group not linked to the wishlist", body["groups"])
	}
}

func TestGetGroupsMissingAuth(t *testing.T) {
	setupControllersDB(t)

	ctx, w := groupTestContext("GET", "/api/groups", "")
	GetGroups(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 without auth", w.Code)
	}
}

// --- GetGroup ---

func TestGetGroupSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)

	ctx, w := groupTestContext("GET", "/api/groups/"+group.ID.String(), "")
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	GetGroup(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestGetGroupNotMember(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	stranger := createTestUser(t)

	ctx, w := groupTestContext("GET", "/api/groups/"+group.ID.String(), "")
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, stranger.ID, false))
	GetGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a non-member", w.Code)
	}
}

func TestGetGroupInvalidID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	ctx, w := groupTestContext("GET", "/api/groups/not-a-uuid", "")
	ctx.Params = gin.Params{{Key: "group_id", Value: "not-a-uuid"}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	GetGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed group ID", w.Code)
	}
}

// --- GetGroupMembers ---

func TestGetGroupMembersSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	member := createTestUser(t)
	addGroupMembership(t, group.ID, member.ID)

	ctx, w := groupTestContext("GET", "/api/groups/"+group.ID.String()+"/members", "")
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	GetGroupMembers(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	body := groupJSONBody(t, w)
	members, ok := body["group_members"].([]interface{})
	if !ok || len(members) != 2 {
		t.Fatalf("group_members = %v, want 2 members", body["group_members"])
	}
}

func TestGetGroupMembersNotMember(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	stranger := createTestUser(t)

	ctx, w := groupTestContext("GET", "/api/groups/"+group.ID.String()+"/members", "")
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, stranger.ID, false))
	GetGroupMembers(ctx)

	// The handler reports non-membership as a 500, not a 400 - documenting
	// actual behavior here rather than the arguably-more-correct 400/403.
	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 for a non-member (current handler behavior)", w.Code)
	}
}

// --- APIUpdateGroup ---

func TestAPIUpdateGroupSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)

	ctx, w := groupTestContext("PUT", "/api/groups/"+group.ID.String(), `{"name":"Renamed Group","description":"New description"}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateGroup(ctx)

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
}

func TestAPIUpdateGroupNotOwner(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	stranger := createTestUser(t)

	ctx, w := groupTestContext("PUT", "/api/groups/"+group.ID.String(), `{"name":"Renamed Group","description":"New description"}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, stranger.ID, false))
	APIUpdateGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 when the caller doesn't own the group", w.Code)
	}
}

func TestAPIUpdateGroupNameTooShort(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)

	ctx, w := groupTestContext("PUT", "/api/groups/"+group.ID.String(), `{"name":"Abc","description":"New description"}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a too-short name", w.Code)
	}
}

func TestAPIUpdateGroupDuplicateName(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	groupA := createTestGroup(t, owner.ID)
	addGroupMembership(t, groupA.ID, owner.ID)
	groupB := createTestGroup(t, owner.ID)
	addGroupMembership(t, groupB.ID, owner.ID)

	ctx, w := groupTestContext("PUT", "/api/groups/"+groupB.ID.String(), `{"name":"`+groupA.Name+`","description":"New description"}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: groupB.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a name collision with another owned group", w.Code)
	}
}

func TestAPIUpdateGroupDescriptionTooShort(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)

	ctx, w := groupTestContext("PUT", "/api/groups/"+group.ID.String(), `{"name":"`+group.Name+`","description":"hi"}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a too-short description", w.Code)
	}
}

func TestAPIUpdateGroupInvalidBody(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)

	ctx, w := groupTestContext("PUT", "/api/groups/"+group.ID.String(), `not-json`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for invalid JSON", w.Code)
	}
}

// --- APIAddWishlistsToGroup ---

func TestAPIAddWishlistsToGroupSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	wishlist := createTestWishlist(t, owner.ID)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/wishlists", `{"wishlists":["`+wishlist.ID.String()+`"]}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIAddWishlistsToGroup(ctx)

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
}

func TestAPIAddWishlistsToGroupEmptyList(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/wishlists", `{"wishlists":[]}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIAddWishlistsToGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for an empty wishlists list", w.Code)
	}
}

func TestAPIAddWishlistsToGroupNotWishlistOwner(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	otherOwner := createTestUser(t)
	wishlist := createTestWishlist(t, otherOwner.ID)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/wishlists", `{"wishlists":["`+wishlist.ID.String()+`"]}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIAddWishlistsToGroup(ctx)

	if w.Code != 401 {
		t.Fatalf("status = %d, want 401 when the wishlist isn't owned by the caller", w.Code)
	}
}

func TestAPIAddWishlistsToGroupAlreadyLinked(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	wishlist := createTestWishlist(t, owner.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/wishlists", `{"wishlists":["`+wishlist.ID.String()+`"]}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIAddWishlistsToGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 when the wishlist is already linked", w.Code)
	}
}

func TestAPIAddWishlistsToGroupNotGroupMember(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	stranger := createTestUser(t)
	wishlist := createTestWishlist(t, stranger.ID)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/wishlists", `{"wishlists":["`+wishlist.ID.String()+`"]}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, stranger.ID, false))
	APIAddWishlistsToGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 when the caller isn't a member of the group", w.Code)
	}
}
