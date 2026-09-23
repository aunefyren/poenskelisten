package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestRegisterGroupNameInvalidCharacters(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	ctx, w := groupTestContext("POST", "/api/groups", `{"name":"<script>","description":"A group"}`)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a name with disallowed characters", w.Code)
	}
}

func TestRegisterGroupDescriptionInvalidCharacters(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	ctx, w := groupTestContext("POST", "/api/groups", `{"name":"Valid Name","description":"<script>bad</script>"}`)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a description with disallowed characters", w.Code)
	}
}

func TestRegisterGroupWishlistNotFound(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	reqBody := `{"name":"Team Group","description":"desc","wishlists":["` + uuid.NewString() + `"]}`
	ctx, w := groupTestContext("POST", "/api/groups", reqBody)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a nonexistent wishlist ID", w.Code)
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

func TestJoinGroupInvalidBody(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/join", `not-json`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	JoinGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for invalid JSON", w.Code)
	}
}

func TestJoinGroupMissingAuth(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	member := createTestUser(t)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/join", `{"members":["`+member.ID.String()+`"]}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	JoinGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 without auth", w.Code)
	}
}

func TestJoinGroupCallerNotOwner(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)
	nonOwnerMember := createTestUser(t)
	addGroupMembership(t, group.ID, nonOwnerMember.ID)
	newUser := createTestUser(t)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/join", `{"members":["`+newUser.ID.String()+`"]}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, nonOwnerMember.ID, false))
	JoinGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 when the caller is a member but not the owner", w.Code)
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

func TestRemoveFromGroupInvalidBody(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/remove", `not-json`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RemoveFromGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for invalid JSON", w.Code)
	}
}

func TestRemoveFromGroupInvalidGroupID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	member := createTestUser(t)

	ctx, w := groupTestContext("POST", "/api/groups/not-a-uuid/remove", `{"member_id":"`+member.ID.String()+`"}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: "not-a-uuid"}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RemoveFromGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed group ID", w.Code)
	}
}

func TestRemoveFromGroupMissingAuth(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	member := createTestUser(t)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/remove", `{"member_id":"`+member.ID.String()+`"}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	RemoveFromGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 without auth", w.Code)
	}
}

// --- RemoveSelfFromGroup ---

// RemoveSelfFromGroup's ownership check must be evaluated against the
// authenticated caller's own ID, not an unbound, zero-value
// models.GroupMembership{}'s MemberID (uuid.Nil) - the latter let GORM's
// struct Where clause silently drop the OwnerID filter, degrading the check to
// "does an enabled group with this ID exist" (true for any enabled group) and
// blocking every caller, not just owners.
func TestRemoveSelfFromGroupMemberSuccess(t *testing.T) {
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

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201 for a plain member leaving; body=%s", w.Code, w.Body.String())
	}
}

func TestRemoveSelfFromGroupOwnerBlocked(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/leave", "")
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RemoveSelfFromGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 (owners cannot remove themselves); body=%s", w.Code, w.Body.String())
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

func TestDeleteGroupMissingAuth(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)

	ctx, w := groupTestContext("DELETE", "/api/groups/"+group.ID.String(), "")
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	DeleteGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 without auth", w.Code)
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

func TestGetGroupMissingAuth(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)

	ctx, w := groupTestContext("GET", "/api/groups/"+group.ID.String(), "")
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	GetGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 without auth", w.Code)
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

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a non-member", w.Code)
	}
}

func TestGetGroupMembersInvalidGroupID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	ctx, w := groupTestContext("GET", "/api/groups/not-a-uuid/members", "")
	ctx.Params = gin.Params{{Key: "group_id", Value: "not-a-uuid"}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	GetGroupMembers(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed group ID", w.Code)
	}
}

func TestGetGroupMembersMissingAuth(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)

	ctx, w := groupTestContext("GET", "/api/groups/"+group.ID.String()+"/members", "")
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	GetGroupMembers(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 without auth", w.Code)
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

func TestAPIUpdateGroupInvalidGroupID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	ctx, w := groupTestContext("PUT", "/api/groups/not-a-uuid", `{"name":"Renamed Group","description":"New description"}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: "not-a-uuid"}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed group ID", w.Code)
	}
}

func TestAPIUpdateGroupMissingAuth(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)

	ctx, w := groupTestContext("PUT", "/api/groups/"+group.ID.String(), `{"name":"Renamed Group","description":"New description"}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	APIUpdateGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 without auth", w.Code)
	}
}

func TestAPIUpdateGroupNameInvalidCharacters(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)

	ctx, w := groupTestContext("PUT", "/api/groups/"+group.ID.String(), `{"name":"<script>bad</script>","description":"New description"}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a name with disallowed characters", w.Code)
	}
}

func TestAPIUpdateGroupDescriptionInvalidCharacters(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)

	ctx, w := groupTestContext("PUT", "/api/groups/"+group.ID.String(), `{"name":"`+group.Name+`","description":"<script>bad</script>"}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a description with disallowed characters", w.Code)
	}
}

func TestAPIUpdateGroupSameNameSkipsNameValidation(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)

	// Same name as before (even though it's short), only the description
	// changes - the name-changed branch (and its length/charset checks)
	// should be skipped entirely.
	ctx, w := groupTestContext("PUT", "/api/groups/"+group.ID.String(), `{"name":"`+group.Name+`","description":"A brand new description"}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIUpdateGroup(ctx)

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
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

func TestAPIAddWishlistsToGroupInvalidBody(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/wishlists", `not-json`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIAddWishlistsToGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for invalid JSON", w.Code)
	}
}

func TestAPIAddWishlistsToGroupMissingAuth(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	wishlist := createTestWishlist(t, owner.ID)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/wishlists", `{"wishlists":["`+wishlist.ID.String()+`"]}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	APIAddWishlistsToGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 without auth", w.Code)
	}
}

func TestAPIAddWishlistsToGroupInvalidGroupID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	ctx, w := groupTestContext("POST", "/api/groups/not-a-uuid/wishlists", `{"wishlists":["`+wishlist.ID.String()+`"]}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: "not-a-uuid"}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIAddWishlistsToGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed group ID", w.Code)
	}
}

func TestAPIAddWishlistsToGroupWishlistNotFound(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)

	ctx, w := groupTestContext("POST", "/api/groups/"+group.ID.String()+"/wishlists", `{"wishlists":["`+uuid.NewString()+`"]}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: group.ID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	APIAddWishlistsToGroup(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a nonexistent wishlist", w.Code)
	}
}

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

func TestGetGroupObjectsDatabaseFailure(t *testing.T) {
	// Migrate without GroupMembership so the join in GetGroupsAUserIsAMemberOf fails.
	setupControllersDB(t, &models.User{}, &models.Group{})
	user := createTestUser(t)

	if _, err := GetGroupObjects(user.ID); err == nil {
		t.Error("expected an error when the group_memberships table is unavailable")
	}
}

func TestGetGroupObjectDatabaseFailure(t *testing.T) {
	setupControllersDB(t, &models.User{}, &models.Group{})
	user := createTestUser(t)

	if _, err := GetGroupObject(user.ID, uuid.New()); err == nil {
		t.Error("expected an error when the group_memberships table is unavailable")
	}
}

func TestGetGroupObjectConvertFailure(t *testing.T) {
	// Group + GroupMembership present so the lookup succeeds, but the User
	// table is gone, so ConvertGroupToGroupObject's owner lookup fails.
	setupControllersDB(t, &models.User{}, &models.Group{}, &models.GroupMembership{})
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMembership(t, group.ID, owner.ID)

	if result := database.Instance.Migrator().DropTable(&models.User{}); result != nil {
		t.Fatalf("failed to drop users table: %v", result)
	}

	if _, err := GetGroupObject(owner.ID, group.ID); err == nil {
		t.Error("expected an error when the owner's user record can't be loaded")
	}
}

func TestRegisterGroupOwnerMembershipCreateFailure(t *testing.T) {
	// Group table present so the duplicate-name check and CreateGroupInDB
	// succeed, but GroupMembership is missing so the owner-membership insert
	// fails.
	setupControllersDB(t, &models.User{}, &models.Group{})
	owner := createTestUser(t)

	ctx, w := groupTestContext("POST", "/api/groups", `{"name":"My Group","description":"A group"}`)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterGroup(ctx)

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the group_memberships table is unavailable; body=%s", w.Code, w.Body.String())
	}
}

func TestRegisterGroupWishlistMembershipCreateFailure(t *testing.T) {
	// Everything needed for the group + owner membership + wishlist ownership
	// check to succeed is present, but WishlistMembership is missing so the
	// final wishlist-linking insert fails.
	setupControllersDB(t, &models.User{}, &models.Group{}, &models.GroupMembership{}, &models.Wishlist{})
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	reqBody := `{"name":"Team Group","description":"desc","wishlists":["` + wishlist.ID.String() + `"]}`
	ctx, w := groupTestContext("POST", "/api/groups", reqBody)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RegisterGroup(ctx)

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the wishlist_memberships table is unavailable; body=%s", w.Code, w.Body.String())
	}
}

func TestJoinGroupMembershipCheckFailure(t *testing.T) {
	// GroupMembership table missing so VerifyUserMembershipToGroup fails.
	setupControllersDB(t, &models.User{}, &models.Group{})
	owner := createTestUser(t)
	newMember := createTestUser(t)

	ctx, w := groupTestContext("POST", "/api/groups/x/join", `{"members":["`+newMember.ID.String()+`"]}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: uuid.NewString()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	JoinGroup(ctx)

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the group_memberships table is unavailable; body=%s", w.Code, w.Body.String())
	}
}

func TestJoinGroupOwnershipCheckFailure(t *testing.T) {
	// GroupMembership present (so the not-already-a-member check succeeds),
	// Group missing so the ownership check fails.
	setupControllersDB(t, &models.User{}, &models.GroupMembership{})
	owner := createTestUser(t)
	newMember := createTestUser(t)

	ctx, w := groupTestContext("POST", "/api/groups/x/join", `{"members":["`+newMember.ID.String()+`"]}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: uuid.NewString()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	JoinGroup(ctx)

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the groups table is unavailable; body=%s", w.Code, w.Body.String())
	}
}

func TestRemoveFromGroupMembershipCheckFailure(t *testing.T) {
	setupControllersDB(t, &models.User{}, &models.Group{})
	owner := createTestUser(t)
	member := createTestUser(t)

	ctx, w := groupTestContext("POST", "/api/groups/x/remove", `{"member_id":"`+member.ID.String()+`"}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: uuid.NewString()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RemoveFromGroup(ctx)

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the group_memberships table is unavailable; body=%s", w.Code, w.Body.String())
	}
}

func TestRemoveFromGroupInformationCheckFailure(t *testing.T) {
	// GroupMembership present so the membership check succeeds; Group missing
	// so GetGroupInformation fails next.
	setupControllersDB(t, &models.User{}, &models.GroupMembership{})
	owner := createTestUser(t)
	member := createTestUser(t)
	groupID := uuid.New()
	addGroupMembership(t, groupID, member.ID)

	ctx, w := groupTestContext("POST", "/api/groups/x/remove", `{"member_id":"`+member.ID.String()+`"}`)
	ctx.Params = gin.Params{{Key: "group_id", Value: groupID.String()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	RemoveFromGroup(ctx)

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the groups table is unavailable; body=%s", w.Code, w.Body.String())
	}
}

func TestGetGroupsMemberOfWishlistBadUUID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	ctx, w := groupTestContext("GET", "/api/groups?memberOfWishlistID=not-a-uuid", "")
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	GetGroups(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed memberOfWishlistID", w.Code)
	}
}

func TestGetGroupsNotAMemberOfWishlistBadUUID(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)

	ctx, w := groupTestContext("GET", "/api/groups?notAMemberOfWishlistID=not-a-uuid", "")
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	GetGroups(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed notAMemberOfWishlistID", w.Code)
	}
}

func TestDeleteGroupOwnershipCheckFailure(t *testing.T) {
	setupControllersDB(t, &models.User{})
	owner := createTestUser(t)

	ctx, w := groupTestContext("DELETE", "/api/groups/x", "")
	ctx.Params = gin.Params{{Key: "group_id", Value: uuid.NewString()}}
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	DeleteGroup(ctx)

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the groups table is unavailable; body=%s", w.Code, w.Body.String())
	}
}

func TestGroupHandlersDatabaseErrors(t *testing.T) {
	groupParam := gin.Params{{Key: "group_id", Value: "00000000-0000-0000-0000-00000000000a"}}
	runDatabaseErrorCases(t, []dbErrorCase{
		{name: "RegisterGroup", handler: RegisterGroup, method: "POST", path: "/api/auth/groups", body: `{"name":"Group","description":"Desc"}`},
		{name: "JoinGroup", handler: JoinGroup, method: "POST", path: "/api/auth/groups/00000000-0000-0000-0000-00000000000a/join", body: `{"members":["00000000-0000-0000-0000-00000000000a"]}`, params: groupParam},
		{name: "RemoveFromGroup", handler: RemoveFromGroup, method: "POST", path: "/api/auth/groups/00000000-0000-0000-0000-00000000000a/remove", body: `{"member_id":"00000000-0000-0000-0000-00000000000a"}`, params: groupParam},
		{name: "APIAddWishlistsToGroup", handler: APIAddWishlistsToGroup, method: "POST", path: "/api/auth/groups/00000000-0000-0000-0000-00000000000a/add", body: `{"wishlists":["00000000-0000-0000-0000-00000000000a"]}`, params: groupParam},
		{name: "RemoveSelfFromGroup", handler: RemoveSelfFromGroup, method: "POST", path: "/api/auth/groups/00000000-0000-0000-0000-00000000000a/leave", params: groupParam},
		{name: "DeleteGroup", handler: DeleteGroup, method: "DELETE", path: "/api/auth/groups/00000000-0000-0000-0000-00000000000a", params: groupParam},
		{name: "GetGroups", handler: GetGroups, method: "GET", path: "/api/auth/groups"},
		{name: "GetGroups/owned", handler: GetGroups, method: "GET", path: "/api/auth/groups?owned=true"},
		{name: "GetGroups/memberOfWishlistID", handler: GetGroups, method: "GET", path: "/api/auth/groups?memberOfWishlistID=00000000-0000-0000-0000-00000000000a"},
		{name: "GetGroups/notAMemberOfWishlistID", handler: GetGroups, method: "GET", path: "/api/auth/groups?notAMemberOfWishlistID=00000000-0000-0000-0000-00000000000a"},
		{name: "GetGroup", handler: GetGroup, method: "GET", path: "/api/auth/groups/00000000-0000-0000-0000-00000000000a", params: groupParam},
		{name: "GetGroupMembers", handler: GetGroupMembers, method: "GET", path: "/api/auth/groups/00000000-0000-0000-0000-00000000000a/members", params: groupParam},
		{name: "APIUpdateGroup", handler: APIUpdateGroup, method: "POST", path: "/api/auth/groups/00000000-0000-0000-0000-00000000000a", body: `{"name":"Group","description":"Desc"}`, params: groupParam},
	})
}

func TestGroupHandlersRequireAuth(t *testing.T) {
	runUnauthenticatedCases(t, []dbErrorCase{
		{name: "RemoveSelfFromGroup", handler: RemoveSelfFromGroup, method: "POST", path: "/api/auth/groups/00000000-0000-0000-0000-00000000000a/leave", params: gin.Params{{Key: "group_id", Value: "00000000-0000-0000-0000-00000000000a"}}},
	})
}

func TestRemoveSelfFromGroupMalformedID(t *testing.T) {
	setupControllersDB(t)
	header := map[string]string{"Authorization": authHeader(t, uuid.New(), false)}

	status, body, _ := doRequest(RemoveSelfFromGroup, "POST", "/api/auth/groups/not-a-uuid/leave", "", header, gin.Params{{Key: "group_id", Value: "not-a-uuid"}})
	if status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body=%v", status, body)
	}
}

// --- Deeper failure branches and multi-group ordering ---

// groupTestFixture is a group owned by owner with owner and member enrolled,
// plus an outsider (not in the group) and a wishlist the owner has not yet
// linked to the group.
type groupTestFixture struct {
	owner    models.User
	member   models.User
	outsider models.User
	group    models.Group
	wishlist models.Wishlist
}

func groupTestNewFixture(t *testing.T) groupTestFixture {
	t.Helper()
	f := groupTestFixture{
		owner:    createTestUser(t),
		member:   createTestUser(t),
		outsider: createTestUser(t),
	}
	f.group = createTestGroup(t, f.owner.ID)
	addGroupMembership(t, f.group.ID, f.owner.ID)
	addGroupMembership(t, f.group.ID, f.member.ID)
	f.wishlist = createTestWishlist(t, f.owner.ID)
	return f
}

// groupTestCreateGroupAt inserts a group owned by ownerID with an explicit
// creation time and enrolls every given member, so tests asserting the
// handlers' created-at ordering don't depend on clock resolution.
func groupTestCreateGroupAt(t *testing.T, ownerID uuid.UUID, name string, createdAt time.Time, memberIDs ...uuid.UUID) models.Group {
	t.Helper()
	group := models.Group{Name: name, Enabled: true, OwnerID: ownerID}
	group.ID = uuid.New()
	group.CreatedAt = createdAt
	created, err := database.CreateGroupInDB(group)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	for _, memberID := range memberIDs {
		addGroupMembership(t, created.ID, memberID)
	}
	return created
}

// groupTestNames returns the "name" of each entry in body["groups"], in order.
func groupTestNames(t *testing.T, body map[string]interface{}) []string {
	t.Helper()
	groups, ok := body["groups"].([]interface{})
	if !ok {
		t.Fatalf("groups = %v, want a list", body["groups"])
	}
	names := []string{}
	for _, g := range groups {
		names = append(names, g.(map[string]interface{})["name"].(string))
	}
	return names
}

func TestGroupHandlersInjectedDatabaseFailures(t *testing.T) {
	groupParam := func(f groupTestFixture) gin.Params {
		return gin.Params{{Key: "group_id", Value: f.group.ID.String()}}
	}
	cases := []struct {
		name       string
		handler    gin.HandlerFunc
		method     string
		path       string
		body       func(f groupTestFixture) string
		caller     func(f groupTestFixture) uuid.UUID
		op, table  string
		skip       int
		wantStatus int
		wantError  string
	}{
		{
			name: "RegisterGroup/create group", handler: RegisterGroup, method: "POST", path: "/api/auth/groups",
			body:   func(f groupTestFixture) string { return `{"name":"Fresh Group","description":"Desc"}` },
			caller: func(f groupTestFixture) uuid.UUID { return f.owner.ID },
			op:     "create", table: "groups", skip: 0,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to create group in database.",
		},
		{
			// A lookup that fails (rather than finding no such user) is an
			// internal error; TestRegisterGroupUnknownMember covers the 400.
			name: "RegisterGroup/member lookup", handler: RegisterGroup, method: "POST", path: "/api/auth/groups",
			body: func(f groupTestFixture) string {
				return `{"name":"Fresh Group","description":"Desc","members":["` + f.outsider.ID.String() + `"]}`
			},
			caller: func(f groupTestFixture) uuid.UUID { return f.owner.ID },
			op:     "query", table: "users", skip: 0,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to get user.",
		},
		{
			// The owner's own membership is create #1; the requested member's is #2.
			name: "RegisterGroup/create member membership", handler: RegisterGroup, method: "POST", path: "/api/auth/groups",
			body: func(f groupTestFixture) string {
				return `{"name":"Fresh Group","description":"Desc","members":["` + f.outsider.ID.String() + `"]}`
			},
			caller: func(f groupTestFixture) uuid.UUID { return f.owner.ID },
			op:     "create", table: "group_memberships", skip: 1,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to create group memberships.",
		},
		{
			// Query #1 on groups is the duplicate-name check.
			name: "RegisterGroup/list groups", handler: RegisterGroup, method: "POST", path: "/api/auth/groups",
			body:   func(f groupTestFixture) string { return `{"name":"Fresh Group","description":"Desc"}` },
			caller: func(f groupTestFixture) uuid.UUID { return f.owner.ID },
			op:     "query", table: "groups", skip: 1,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to get group objects.",
		},
		{
			name: "JoinGroup/create membership", handler: JoinGroup, method: "POST", path: "/api/auth/groups/x/join",
			body:   func(f groupTestFixture) string { return `{"members":["` + f.outsider.ID.String() + `"]}` },
			caller: func(f groupTestFixture) uuid.UUID { return f.owner.ID },
			op:     "create", table: "group_memberships", skip: 0,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to create group membership in database.",
		},
		{
			// Query #1 on groups is the ownership check.
			name: "JoinGroup/list groups", handler: JoinGroup, method: "POST", path: "/api/auth/groups/x/join",
			body:   func(f groupTestFixture) string { return `{"members":["` + f.outsider.ID.String() + `"]}` },
			caller: func(f groupTestFixture) uuid.UUID { return f.owner.ID },
			op:     "query", table: "groups", skip: 1,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to get groups for user.",
		},
		{
			// Query #1 on group_memberships is the membership check.
			name: "RemoveFromGroup/membership lookup", handler: RemoveFromGroup, method: "POST", path: "/api/auth/groups/x/remove",
			body:   func(f groupTestFixture) string { return `{"member_id":"` + f.member.ID.String() + `"}` },
			caller: func(f groupTestFixture) uuid.UUID { return f.owner.ID },
			op:     "query", table: "group_memberships", skip: 1,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to verify membership.",
		},
		{
			name: "RemoveFromGroup/delete membership", handler: RemoveFromGroup, method: "POST", path: "/api/auth/groups/x/remove",
			body:   func(f groupTestFixture) string { return `{"member_id":"` + f.member.ID.String() + `"}` },
			caller: func(f groupTestFixture) uuid.UUID { return f.owner.ID },
			op:     "update", table: "group_memberships", skip: 0,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to delete group membership.",
		},
		{
			// Query #1 on groups is GetGroupInformation.
			name: "RemoveFromGroup/list groups", handler: RemoveFromGroup, method: "POST", path: "/api/auth/groups/x/remove",
			body:   func(f groupTestFixture) string { return `{"member_id":"` + f.member.ID.String() + `"}` },
			caller: func(f groupTestFixture) uuid.UUID { return f.owner.ID },
			op:     "query", table: "groups", skip: 1,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to get group objects.",
		},
		{
			name: "RemoveSelfFromGroup/ownership check", handler: RemoveSelfFromGroup, method: "POST", path: "/api/auth/groups/x/leave",
			caller: func(f groupTestFixture) uuid.UUID { return f.member.ID },
			op:     "query", table: "groups", skip: 0,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to verify ownership of group.",
		},
		{
			name: "RemoveSelfFromGroup/membership lookup", handler: RemoveSelfFromGroup, method: "POST", path: "/api/auth/groups/x/leave",
			caller: func(f groupTestFixture) uuid.UUID { return f.member.ID },
			op:     "query", table: "group_memberships", skip: 1,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to verify membership to group.",
		},
		{
			name: "RemoveSelfFromGroup/delete membership", handler: RemoveSelfFromGroup, method: "POST", path: "/api/auth/groups/x/leave",
			caller: func(f groupTestFixture) uuid.UUID { return f.member.ID },
			op:     "update", table: "group_memberships", skip: 0,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to delete group membership.",
		},
		{
			name: "RemoveSelfFromGroup/list groups", handler: RemoveSelfFromGroup, method: "POST", path: "/api/auth/groups/x/leave",
			caller: func(f groupTestFixture) uuid.UUID { return f.member.ID },
			op:     "query", table: "groups", skip: 1,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to get group objects.",
		},
		{
			name: "DeleteGroup/disable group", handler: DeleteGroup, method: "DELETE", path: "/api/auth/groups/x",
			caller: func(f groupTestFixture) uuid.UUID { return f.owner.ID },
			op:     "update", table: "groups", skip: 0,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to delete the group.",
		},
		{
			name: "DeleteGroup/list groups", handler: DeleteGroup, method: "DELETE", path: "/api/auth/groups/x",
			caller: func(f groupTestFixture) uuid.UUID { return f.owner.ID },
			op:     "query", table: "groups", skip: 1,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to get group objects.",
		},
		{
			name: "GetGroup/load group object", handler: GetGroup, method: "GET", path: "/api/auth/groups/x",
			caller: func(f groupTestFixture) uuid.UUID { return f.member.ID },
			op:     "query", table: "groups", skip: 0,
			wantStatus: http.StatusInternalServerError, wantError: "Failed process group object.",
		},
		{
			name: "GetGroupMembers/list memberships", handler: GetGroupMembers, method: "GET", path: "/api/auth/groups/x/members",
			caller: func(f groupTestFixture) uuid.UUID { return f.member.ID },
			op:     "query", table: "group_memberships", skip: 1,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to get group memberships for group.",
		},
		{
			name: "GetGroupMembers/load member", handler: GetGroupMembers, method: "GET", path: "/api/auth/groups/x/members",
			caller: func(f groupTestFixture) uuid.UUID { return f.member.ID },
			op:     "query", table: "users", skip: 0,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to get user object for group member.",
		},
		{
			// Query #1 on groups is the ownership check.
			name: "APIUpdateGroup/load original", handler: APIUpdateGroup, method: "POST", path: "/api/auth/groups/x",
			body:   func(f groupTestFixture) string { return `{"name":"Renamed Group","description":"New description"}` },
			caller: func(f groupTestFixture) uuid.UUID { return f.owner.ID },
			op:     "query", table: "groups", skip: 1,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to find group.",
		},
		{
			name: "APIUpdateGroup/duplicate-name check", handler: APIUpdateGroup, method: "POST", path: "/api/auth/groups/x",
			body:   func(f groupTestFixture) string { return `{"name":"Renamed Group","description":"New description"}` },
			caller: func(f groupTestFixture) uuid.UUID { return f.owner.ID },
			op:     "query", table: "groups", skip: 2,
			wantStatus: http.StatusInternalServerError, wantError: "Failed verify group name.",
		},
		{
			name: "APIUpdateGroup/save", handler: APIUpdateGroup, method: "POST", path: "/api/auth/groups/x",
			body:   func(f groupTestFixture) string { return `{"name":"Renamed Group","description":"New description"}` },
			caller: func(f groupTestFixture) uuid.UUID { return f.owner.ID },
			op:     "update", table: "groups", skip: 0,
			wantStatus: http.StatusInternalServerError, wantError: "Failed update group.",
		},
		{
			// Unchanged name skips the duplicate-name query, so query #3 is
			// the reload after saving.
			name: "APIUpdateGroup/reload", handler: APIUpdateGroup, method: "POST", path: "/api/auth/groups/x",
			body: func(f groupTestFixture) string {
				return `{"name":"` + f.group.Name + `","description":"New description"}`
			},
			caller: func(f groupTestFixture) uuid.UUID { return f.owner.ID },
			op:     "query", table: "groups", skip: 2,
			wantStatus: http.StatusInternalServerError, wantError: "Failed convert group to group object.",
		},
		{
			name: "APIAddWishlistsToGroup/wishlist link check", handler: APIAddWishlistsToGroup, method: "POST", path: "/api/auth/groups/x/add",
			body:   func(f groupTestFixture) string { return `{"wishlists":["` + f.wishlist.ID.String() + `"]}` },
			caller: func(f groupTestFixture) uuid.UUID { return f.owner.ID },
			op:     "query", table: "wishlist_memberships", skip: 0,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to verify membership to group.",
		},
		{
			name: "APIAddWishlistsToGroup/group membership check", handler: APIAddWishlistsToGroup, method: "POST", path: "/api/auth/groups/x/add",
			body:   func(f groupTestFixture) string { return `{"wishlists":["` + f.wishlist.ID.String() + `"]}` },
			caller: func(f groupTestFixture) uuid.UUID { return f.owner.ID },
			op:     "query", table: "group_memberships", skip: 0,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to verify membership to group.",
		},
		{
			name: "APIAddWishlistsToGroup/create link", handler: APIAddWishlistsToGroup, method: "POST", path: "/api/auth/groups/x/add",
			body:   func(f groupTestFixture) string { return `{"wishlists":["` + f.wishlist.ID.String() + `"]}` },
			caller: func(f groupTestFixture) uuid.UUID { return f.owner.ID },
			op:     "create", table: "wishlist_memberships", skip: 0,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to create group membership for wishlist in database.",
		},
		{
			name: "APIAddWishlistsToGroup/list groups", handler: APIAddWishlistsToGroup, method: "POST", path: "/api/auth/groups/x/add",
			body:   func(f groupTestFixture) string { return `{"wishlists":["` + f.wishlist.ID.String() + `"]}` },
			caller: func(f groupTestFixture) uuid.UUID { return f.owner.ID },
			op:     "query", table: "groups", skip: 0,
			wantStatus: http.StatusInternalServerError, wantError: "Failed to get groups for user.",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			f := groupTestNewFixture(t)
			body := ""
			if c.body != nil {
				body = c.body(f)
			}
			header := map[string]string{"Authorization": authHeader(t, c.caller(f), false)}
			failDBOperation(t, c.op, c.table, c.skip)

			status, resp, _ := doRequest(c.handler, c.method, c.path, body, header, groupParam(f))
			if status != c.wantStatus {
				t.Fatalf("status = %d, want %d; body=%v", status, c.wantStatus, resp)
			}
			if resp["error"] != c.wantError {
				t.Errorf("error = %v, want %q", resp["error"], c.wantError)
			}
		})
	}
}

func TestGetGroupsWishlistFilterDatabaseFailure(t *testing.T) {
	for _, query := range []string{"memberOfWishlistID", "notAMemberOfWishlistID"} {
		t.Run(query, func(t *testing.T) {
			setupControllersDB(t)
			f := groupTestNewFixture(t)
			header := map[string]string{"Authorization": authHeader(t, f.owner.ID, false)}
			failDBOperation(t, "query", "wishlist_memberships", 0)

			status, resp, _ := doRequest(GetGroups, "GET", "/api/auth/groups?"+query+"="+f.wishlist.ID.String(), "", header, nil)
			if status != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500; body=%v", status, resp)
			}
			if resp["error"] != "Failed to validate group membership." {
				t.Errorf("error = %v", resp["error"])
			}
		})
	}
}

// The mutating handlers answer with the caller's groups newest first.
func TestGroupHandlersSortGroupsNewestFirst(t *testing.T) {
	older := time.Now().Add(-2 * time.Hour)
	newer := time.Now().Add(-1 * time.Hour)

	t.Run("RegisterGroup", func(t *testing.T) {
		setupControllersDB(t)
		owner := createTestUser(t)
		groupTestCreateGroupAt(t, owner.ID, "Older Group", older, owner.ID)
		header := map[string]string{"Authorization": authHeader(t, owner.ID, false)}

		status, resp, _ := doRequest(RegisterGroup, "POST", "/api/auth/groups", `{"name":"Newest Group","description":"Desc"}`, header, nil)
		if status != http.StatusCreated {
			t.Fatalf("status = %d, want 201; body=%v", status, resp)
		}
		if got := groupTestNames(t, resp); len(got) != 2 || got[0] != "Newest Group" || got[1] != "Older Group" {
			t.Errorf("groups = %v, want [Newest Group Older Group]", got)
		}
	})

	t.Run("JoinGroup", func(t *testing.T) {
		setupControllersDB(t)
		owner := createTestUser(t)
		joiner := createTestUser(t)
		target := groupTestCreateGroupAt(t, owner.ID, "Older Group", older, owner.ID)
		groupTestCreateGroupAt(t, owner.ID, "Newer Group", newer, owner.ID)
		header := map[string]string{"Authorization": authHeader(t, owner.ID, false)}

		status, resp, _ := doRequest(JoinGroup, "POST", "/api/auth/groups/x/join", `{"members":["`+joiner.ID.String()+`"]}`, header, gin.Params{{Key: "group_id", Value: target.ID.String()}})
		if status != http.StatusCreated {
			t.Fatalf("status = %d, want 201; body=%v", status, resp)
		}
		if got := groupTestNames(t, resp); len(got) != 2 || got[0] != "Newer Group" || got[1] != "Older Group" {
			t.Errorf("groups = %v, want [Newer Group Older Group]", got)
		}
	})

	t.Run("RemoveFromGroup", func(t *testing.T) {
		setupControllersDB(t)
		owner := createTestUser(t)
		member := createTestUser(t)
		target := groupTestCreateGroupAt(t, owner.ID, "Older Group", older, owner.ID, member.ID)
		groupTestCreateGroupAt(t, owner.ID, "Newer Group", newer, owner.ID)
		header := map[string]string{"Authorization": authHeader(t, owner.ID, false)}

		status, resp, _ := doRequest(RemoveFromGroup, "POST", "/api/auth/groups/x/remove", `{"member_id":"`+member.ID.String()+`"}`, header, gin.Params{{Key: "group_id", Value: target.ID.String()}})
		if status != http.StatusCreated {
			t.Fatalf("status = %d, want 201; body=%v", status, resp)
		}
		if got := groupTestNames(t, resp); len(got) != 2 || got[0] != "Newer Group" || got[1] != "Older Group" {
			t.Errorf("groups = %v, want [Newer Group Older Group]", got)
		}
		if isMember, err := database.VerifyUserMembershipToGroup(member.ID, target.ID); err != nil || isMember {
			t.Errorf("member still in group after removal: isMember=%v err=%v", isMember, err)
		}
	})

	t.Run("RemoveSelfFromGroup", func(t *testing.T) {
		setupControllersDB(t)
		owner := createTestUser(t)
		member := createTestUser(t)
		groupTestCreateGroupAt(t, owner.ID, "Older Group", older, owner.ID, member.ID)
		groupTestCreateGroupAt(t, owner.ID, "Newer Group", newer, owner.ID, member.ID)
		leaving := groupTestCreateGroupAt(t, owner.ID, "Left Group", time.Now(), owner.ID, member.ID)
		header := map[string]string{"Authorization": authHeader(t, member.ID, false)}

		status, resp, _ := doRequest(RemoveSelfFromGroup, "POST", "/api/auth/groups/x/leave", "", header, gin.Params{{Key: "group_id", Value: leaving.ID.String()}})
		if status != http.StatusCreated {
			t.Fatalf("status = %d, want 201; body=%v", status, resp)
		}
		if got := groupTestNames(t, resp); len(got) != 2 || got[0] != "Newer Group" || got[1] != "Older Group" {
			t.Errorf("groups = %v, want [Newer Group Older Group]", got)
		}
	})

	t.Run("DeleteGroup", func(t *testing.T) {
		setupControllersDB(t)
		owner := createTestUser(t)
		groupTestCreateGroupAt(t, owner.ID, "Older Group", older, owner.ID)
		groupTestCreateGroupAt(t, owner.ID, "Newer Group", newer, owner.ID)
		doomed := groupTestCreateGroupAt(t, owner.ID, "Doomed Group", time.Now(), owner.ID)
		header := map[string]string{"Authorization": authHeader(t, owner.ID, false)}

		status, resp, _ := doRequest(DeleteGroup, "DELETE", "/api/auth/groups/x", "", header, gin.Params{{Key: "group_id", Value: doomed.ID.String()}})
		if status != http.StatusCreated {
			t.Fatalf("status = %d, want 201; body=%v", status, resp)
		}
		if got := groupTestNames(t, resp); len(got) != 2 || got[0] != "Newer Group" || got[1] != "Older Group" {
			t.Errorf("groups = %v, want [Newer Group Older Group]", got)
		}
	})

	t.Run("APIAddWishlistsToGroup", func(t *testing.T) {
		setupControllersDB(t)
		owner := createTestUser(t)
		wishlist := createTestWishlist(t, owner.ID)
		target := groupTestCreateGroupAt(t, owner.ID, "Older Group", older, owner.ID)
		groupTestCreateGroupAt(t, owner.ID, "Newer Group", newer, owner.ID)
		header := map[string]string{"Authorization": authHeader(t, owner.ID, false)}

		status, resp, _ := doRequest(APIAddWishlistsToGroup, "POST", "/api/auth/groups/x/add", `{"wishlists":["`+wishlist.ID.String()+`"]}`, header, gin.Params{{Key: "group_id", Value: target.ID.String()}})
		if status != http.StatusCreated {
			t.Fatalf("status = %d, want 201; body=%v", status, resp)
		}
		if got := groupTestNames(t, resp); len(got) != 2 || got[0] != "Newer Group" || got[1] != "Older Group" {
			t.Errorf("groups = %v, want [Newer Group Older Group]", got)
		}
		if linked, err := database.VerifyGroupMembershipToWishlist(wishlist.ID, target.ID); err != nil || !linked {
			t.Errorf("wishlist not linked to group: linked=%v err=%v", linked, err)
		}
	})
}

func TestConvertGroupToGroupObjectMembershipsFailure(t *testing.T) {
	setupControllersDB(t)
	f := groupTestNewFixture(t)
	failDBOperation(t, "query", "group_memberships", 0)

	groupObject, err := ConvertGroupToGroupObject(f.group)
	if err == nil {
		t.Fatal("expected an error when group memberships can't be loaded")
	}
	// The partially-built object (owner filled in, no members) is returned
	// alongside the error.
	if groupObject.ID != f.group.ID || groupObject.Owner.ID != f.owner.ID || len(groupObject.Members) != 0 {
		t.Errorf("groupObject = %+v, want the group with its owner and no members", groupObject)
	}
}

func TestConvertGroupToGroupObjectMemberLookupFailure(t *testing.T) {
	setupControllersDB(t)
	f := groupTestNewFixture(t)
	// Users query #1 is the owner lookup; #2 is the first member.
	failDBOperation(t, "query", "users", 1)

	groupObject, err := ConvertGroupToGroupObject(f.group)
	if err == nil {
		t.Fatal("expected an error when a member's user record can't be loaded")
	}
	if groupObject.ID != uuid.Nil {
		t.Errorf("groupObject = %+v, want the zero value", groupObject)
	}
}

func TestConvertGroupsToGroupObjectsSkipsBrokenGroup(t *testing.T) {
	setupControllersDB(t)
	f := groupTestNewFixture(t)
	orphan := models.Group{Name: "Orphaned Group", Enabled: true, OwnerID: uuid.New()}
	orphan.ID = uuid.New()

	groupObjects := ConvertGroupsToGroupObjects([]models.Group{orphan, f.group})
	if len(groupObjects) != 1 || groupObjects[0].ID != f.group.ID {
		t.Fatalf("groupObjects = %+v, want only the group whose owner exists", groupObjects)
	}
	if len(groupObjects[0].Members) != 2 {
		t.Errorf("members = %d, want 2", len(groupObjects[0].Members))
	}
}
