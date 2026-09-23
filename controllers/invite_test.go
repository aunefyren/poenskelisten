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

func TestRegisterInvite(t *testing.T) {
	setupControllersDB(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/api/admin/invites", nil)
	RegisterInvite(ctx)

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}

	var body struct {
		Invitation string                   `json:"invitation"`
		Invites    []map[string]interface{} `json:"invites"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body.Invitation == "" {
		t.Error("expected a non-empty invitation code")
	}
	if len(body.Invites) != 1 {
		t.Errorf("invites = %v, want exactly the one just created", body.Invites)
	}
}

func TestRegisterInviteDatabaseFailure(t *testing.T) {
	// Migrate without the Invite table so GenerateRandomInvite's insert fails.
	setupControllersDB(t, &models.User{})

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/api/admin/invites", nil)
	RegisterInvite(ctx)

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the invite table is unavailable; body=%s", w.Code, w.Body.String())
	}
}

func TestGetAllInvitesDatabaseFailure(t *testing.T) {
	// Migrate without the Invite table so GetAllEnabledInvites fails.
	setupControllersDB(t, &models.User{})

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/admin/invites", nil)
	APIGetAllInvites(ctx)

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the invite table is unavailable; body=%s", w.Code, w.Body.String())
	}
}

func TestGetAllInvites(t *testing.T) {
	setupControllersDB(t)
	createTestInvite(t)
	createTestInvite(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/admin/invites", nil)
	APIGetAllInvites(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	var body struct {
		Invites []map[string]interface{} `json:"invites"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(body.Invites) != 2 {
		t.Errorf("invites = %v, want 2", body.Invites)
	}
}

func TestDeleteInviteBadID(t *testing.T) {
	setupControllersDB(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("DELETE", "/api/admin/invites/not-a-uuid", nil)
	ctx.Params = gin.Params{{Key: "invite_id", Value: "not-a-uuid"}}
	APIDeleteInvite(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed invite_id", w.Code)
	}
}

func TestDeleteInviteNotFound(t *testing.T) {
	setupControllersDB(t)

	unknown := uuid.New()
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("DELETE", "/api/admin/invites/"+unknown.String(), nil)
	ctx.Params = gin.Params{{Key: "invite_id", Value: unknown.String()}}
	APIDeleteInvite(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for an unknown invite_id", w.Code)
	}
}

func TestDeleteInviteUsed(t *testing.T) {
	setupControllersDB(t)
	invite := createTestInvite(t)
	user := createTestUser(t)
	invite.Used = true
	invite.RecipientID = &user.ID
	if result := database.Instance.Save(&invite); result.Error != nil {
		t.Fatalf("failed to mark invite used: %v", result.Error)
	}

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("DELETE", "/api/admin/invites/"+invite.ID.String(), nil)
	ctx.Params = gin.Params{{Key: "invite_id", Value: invite.ID.String()}}
	APIDeleteInvite(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a used invite (can't be deleted); body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteInviteSuccess(t *testing.T) {
	setupControllersDB(t)
	invite := createTestInvite(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("DELETE", "/api/admin/invites/"+invite.ID.String(), nil)
	ctx.Params = gin.Params{{Key: "invite_id", Value: invite.ID.String()}}
	APIDeleteInvite(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	remaining, err := database.GetAllEnabledInvites()
	if err != nil {
		t.Fatalf("failed to list invites: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("expected the invite to be gone, got %v", remaining)
	}
}

func TestConvertInviteToInviteObjectNoRecipient(t *testing.T) {
	setupControllersDB(t)
	invite := createTestInvite(t)

	obj, err := ConvertInviteToInviteObject(invite)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obj.InviteCode != invite.Code {
		t.Errorf("InviteCode = %v, want %v", obj.InviteCode, invite.Code)
	}
}

func TestConvertInviteToInviteObjectWithRecipient(t *testing.T) {
	setupControllersDB(t)
	invite := createTestInvite(t)
	user := createTestUser(t)
	invite.RecipientID = &user.ID
	if result := database.Instance.Save(&invite); result.Error != nil {
		t.Fatalf("failed to set recipient: %v", result.Error)
	}

	obj, err := ConvertInviteToInviteObject(invite)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obj.User.ID != user.ID {
		t.Errorf("User.ID = %v, want %v", obj.User.ID, user.ID)
	}
}

func TestConvertInvitesToInviteObjectsSkipsBroken(t *testing.T) {
	setupControllersDB(t)
	good := createTestInvite(t)
	broken := createTestInvite(t)
	missing := uuid.New()
	broken.RecipientID = &missing
	if result := database.Instance.Save(&broken); result.Error != nil {
		t.Fatalf("failed to set a dangling recipient: %v", result.Error)
	}

	objects := ConvertInvitesToInviteObjects([]models.Invite{good, broken})
	if len(objects) != 1 {
		t.Fatalf("objects = %v, want exactly the one good invite (the broken one should be skipped)", objects)
	}
	if objects[0].InviteCode != good.Code {
		t.Errorf("InviteCode = %v, want %v", objects[0].InviteCode, good.Code)
	}
}

func TestInviteHandlersDatabaseErrors(t *testing.T) {
	runDatabaseErrorCases(t, []dbErrorCase{
		{name: "RegisterInvite", handler: RegisterInvite, method: "POST", path: "/api/admin/invites", admin: true},
		{name: "APIDeleteInvite", handler: APIDeleteInvite, method: "DELETE", path: "/api/admin/invites/00000000-0000-0000-0000-00000000000a", params: gin.Params{{Key: "invite_id", Value: "00000000-0000-0000-0000-00000000000a"}}, admin: true},
		{name: "APIGetAllInvites", handler: APIGetAllInvites, method: "GET", path: "/api/admin/invites", admin: true},
	})
}

func TestRegisterInviteSortsNewestFirst(t *testing.T) {
	setupControllersDB(t)
	createTestInvite(t)
	createTestInvite(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/api/admin/invites", nil)
	RegisterInvite(ctx)

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Invitation string                `json:"invitation"`
		Invites    []models.InviteObject `json:"invites"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(body.Invites) != 3 {
		t.Fatalf("len(invites) = %d, want 3", len(body.Invites))
	}
	for i := 1; i < len(body.Invites); i++ {
		if body.Invites[i].CreatedAt.After(body.Invites[i-1].CreatedAt) {
			t.Errorf("invites not sorted newest first: %v after %v", body.Invites[i].CreatedAt, body.Invites[i-1].CreatedAt)
		}
	}
}

func TestRegisterInviteListFailure(t *testing.T) {
	setupControllersDB(t)
	// The insert isn't a query, so this only fails the listing after it.
	failDBOperation(t, "query", "invites", 0)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/api/admin/invites", nil)
	RegisterInvite(ctx)

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Failed to get invites from database.") {
		t.Errorf("body = %s", w.Body.String())
	}
}

func TestDeleteInviteDatabaseFailures(t *testing.T) {
	cases := []struct {
		name      string
		op        string
		skip      int
		wantError string
	}{
		{"disable invite", "update", 0, "Failed to delete invite."},
		// The first invites query is the GetInviteByID lookup.
		{"list remaining", "query", 1, "Failed to get invites from database."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			invite := createTestInvite(t)
			failDBOperation(t, c.op, "invites", c.skip)

			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Request = httptest.NewRequest("DELETE", "/api/admin/invites/"+invite.ID.String(), nil)
			ctx.Params = gin.Params{{Key: "invite_id", Value: invite.ID.String()}}
			APIDeleteInvite(ctx)

			if w.Code != 500 {
				t.Fatalf("status = %d, want 500; body=%s", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), c.wantError) {
				t.Errorf("body = %s, want error %q", w.Body.String(), c.wantError)
			}
		})
	}
}

func TestConvertInviteToInviteObjectDisabledRecipient(t *testing.T) {
	setupControllersDB(t)
	invite := createTestInvite(t)
	user := createTestUser(t)
	user.Enabled = boolPtr(false)
	if _, err := database.UpdateUserInDB(user); err != nil {
		t.Fatalf("failed to disable user: %v", err)
	}
	invite.RecipientID = &user.ID

	if _, err := ConvertInviteToInviteObject(invite); err == nil {
		t.Error("expected an error for an invite whose recipient is deleted")
	}
}
