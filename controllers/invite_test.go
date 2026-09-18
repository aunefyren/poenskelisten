package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"encoding/json"
	"net/http/httptest"
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

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 for an unknown invite_id", w.Code)
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

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 for a used invite (can't be deleted); body=%s", w.Code, w.Body.String())
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

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
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

	objects, err := ConvertInvitesToInviteObjects([]models.Invite{good, broken})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(objects) != 1 {
		t.Fatalf("objects = %v, want exactly the one good invite (the broken one should be skipped)", objects)
	}
	if objects[0].InviteCode != good.Code {
		t.Errorf("InviteCode = %v, want %v", objects[0].InviteCode, good.Code)
	}
}
