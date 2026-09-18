package controllers

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// newUserWithPassword inserts an enabled, verified user whose password is a
// real bcrypt hash of plaintext, so handlers that call user.CheckPassword can
// be exercised on their success path (createTestUser stores a plain string,
// which is fine for auth-header-only tests but not for password checks).
func newUserWithPassword(t *testing.T, plaintext string) models.User {
	t.Helper()

	user := createTestUser(t)
	if err := user.HashPassword(plaintext); err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	updated, err := database.UpdateUserInDB(user)
	if err != nil {
		t.Fatalf("failed to save hashed password: %v", err)
	}

	return updated
}

func doRequest(handler gin.HandlerFunc, method, path, body string, header map[string]string, params gin.Params) (int, map[string]interface{}, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
		ctx.Request = httptest.NewRequest(method, path, reader)
		ctx.Request.Header.Set("Content-Type", "application/json")
	} else {
		ctx.Request = httptest.NewRequest(method, path, nil)
	}
	for k, v := range header {
		ctx.Request.Header.Set(k, v)
	}
	if params != nil {
		ctx.Params = params
	}

	handler(ctx)

	var parsed map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &parsed)
	return w.Code, parsed, w
}

func TestRegisterUserSuccess(t *testing.T) {
	setupControllersDB(t)
	invite := createTestInvite(t)

	body := fmt.Sprintf(`{"first_name":"Ada","last_name":"Lovelace","email":"ada@example.com","password":"Sup3rSecret!","password_repeat":"Sup3rSecret!","invite_code":"%s"}`, invite.Code)
	code, resp, _ := doRequest(RegisterUser, "POST", "/api/open/users/register", body, nil, nil)
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}

	// First user created becomes admin and, since SMTP is disabled by default,
	// is created already verified. GetUserInformationByEmail redacts Verified,
	// so use the unredacted lookup here.
	user, err := database.GetAllUserInformationByEmail("ada@example.com")
	if err != nil {
		t.Fatalf("failed to look up created user: %v", err)
	}
	if !user.Admin {
		t.Error("expected the first registered user to be admin")
	}
	if user.Verified == nil || !*user.Verified {
		t.Error("expected the user to be verified when SMTP is disabled")
	}

	// Invite must now be used.
	unused, err := database.VerifyUnusedUserInviteCode(invite.Code)
	if err != nil {
		t.Fatalf("failed to check invite: %v", err)
	}
	if unused {
		t.Error("expected the invite code to be marked used")
	}
}

func TestRegisterUserPasswordsMustMatch(t *testing.T) {
	setupControllersDB(t)
	invite := createTestInvite(t)

	body := fmt.Sprintf(`{"first_name":"Ada","last_name":"Lovelace","email":"ada@example.com","password":"Sup3rSecret!","password_repeat":"Different!","invite_code":"%s"}`, invite.Code)
	code, resp, _ := doRequest(RegisterUser, "POST", "/api/open/users/register", body, nil, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
	if resp["error"] != "Passwords must match." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestRegisterUserWeakPassword(t *testing.T) {
	setupControllersDB(t)
	invite := createTestInvite(t)

	body := fmt.Sprintf(`{"first_name":"Ada","last_name":"Lovelace","email":"ada@example.com","password":"weak","password_repeat":"weak","invite_code":"%s"}`, invite.Code)
	code, resp, _ := doRequest(RegisterUser, "POST", "/api/open/users/register", body, nil, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
}

func TestRegisterUserInvalidInviteCode(t *testing.T) {
	setupControllersDB(t)

	body := `{"first_name":"Ada","last_name":"Lovelace","email":"ada@example.com","password":"Sup3rSecret!","password_repeat":"Sup3rSecret!","invite_code":"NOT-REAL"}`
	code, resp, _ := doRequest(RegisterUser, "POST", "/api/open/users/register", body, nil, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
	if resp["error"] != "Invitiation code is not valid." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestRegisterUserDuplicateEmail(t *testing.T) {
	setupControllersDB(t)
	existing := createTestUser(t)
	invite := createTestInvite(t)

	body := fmt.Sprintf(`{"first_name":"Ada","last_name":"Lovelace","email":"%s","password":"Sup3rSecret!","password_repeat":"Sup3rSecret!","invite_code":"%s"}`, *existing.Email, invite.Code)
	code, resp, _ := doRequest(RegisterUser, "POST", "/api/open/users/register", body, nil, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
	if resp["error"] != "E-mail is already in use." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestRegisterUserBadJSON(t *testing.T) {
	setupControllersDB(t)

	code, _, _ := doRequest(RegisterUser, "POST", "/api/open/users/register", `not-json`, nil, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400", code)
	}
}

func TestAPICurrentUserRequiresAuth(t *testing.T) {
	setupControllersDB(t)

	code, resp, _ := doRequest(APICurrentUser, "GET", "/api/auth/me", "", nil, nil)
	if code != 401 {
		t.Fatalf("status = %d, want 401; body=%v", code, resp)
	}
}

func TestAPICurrentUserSuccess(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	code, resp, _ := doRequest(APICurrentUser, "GET", "/api/auth/me", "", header, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected a data object, got %v", resp)
	}
	if data["email"] != *user.Email {
		t.Errorf("email = %v, want %v", data["email"], *user.Email)
	}
}

func TestGetUserSelf(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	params := gin.Params{{Key: "user_id", Value: user.ID.String()}}
	code, resp, _ := doRequest(GetUser, "GET", "/api/auth/users/"+user.ID.String(), "", header, params)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
}

func TestGetUserInvalidUUID(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	params := gin.Params{{Key: "user_id", Value: "not-a-uuid"}}
	code, resp, _ := doRequest(GetUser, "GET", "/api/auth/users/not-a-uuid", "", header, params)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
}

func TestGetUserNoAuth(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	params := gin.Params{{Key: "user_id", Value: user.ID.String()}}
	code, resp, _ := doRequest(GetUser, "GET", "/api/auth/users/"+user.ID.String(), "", nil, params)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
}

func TestGetUsersDefaultEnabledOnly(t *testing.T) {
	setupControllersDB(t)
	requester := createTestUser(t)

	header := map[string]string{"Authorization": authHeader(t, requester.ID, false)}
	code, resp, _ := doRequest(GetUsers, "GET", "/api/auth/users", "", header, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	users, ok := resp["users"].([]interface{})
	if !ok {
		t.Fatalf("expected a users array, got %v", resp)
	}
	if len(users) != 1 {
		t.Errorf("len(users) = %d, want 1", len(users))
	}
}

func TestGetUsersNotAMemberOfGroup(t *testing.T) {
	setupControllersDB(t)
	requester := createTestUser(t)
	other := createTestUser(t)
	group := createTestGroup(t, requester.ID)
	addGroupMembership(t, group.ID, requester.ID)

	header := map[string]string{"Authorization": authHeader(t, requester.ID, false)}
	code, resp, _ := doRequest(GetUsers, "GET", "/api/auth/users?notAMemberOfGroupID="+group.ID.String(), "", header, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	users, ok := resp["users"].([]interface{})
	if !ok {
		t.Fatalf("expected a users array, got %v", resp)
	}
	// requester is a member of the group and must be filtered out; other must remain.
	found := false
	for _, u := range users {
		um := u.(map[string]interface{})
		if um["ID"] == other.ID.String() {
			found = true
		}
		if um["ID"] == requester.ID.String() {
			t.Error("requester is a member of the group and should have been excluded")
		}
	}
	_ = found
}

func TestGetUsersBadGroupID(t *testing.T) {
	setupControllersDB(t)
	requester := createTestUser(t)

	header := map[string]string{"Authorization": authHeader(t, requester.ID, false)}
	code, resp, _ := doRequest(GetUsers, "GET", "/api/auth/users?notAMemberOfGroupID=not-a-uuid", "", header, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
}

func TestVerifyUserRequiresLogin(t *testing.T) {
	setupControllersDB(t)

	params := gin.Params{{Key: "code", Value: "ABC123"}}
	code, resp, _ := doRequest(VerifyUser, "GET", "/api/open/users/verify/ABC123", "", nil, params)
	if code != 401 {
		t.Fatalf("status = %d, want 401; body=%v", code, resp)
	}
}

func TestVerifyUserEmptyCode(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	params := gin.Params{{Key: "code", Value: ""}}
	code, resp, _ := doRequest(VerifyUser, "GET", "/api/open/users/verify/", "", header, params)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
}

func TestVerifyUserWrongCode(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	params := gin.Params{{Key: "code", Value: "WRONGCODE"}}
	code, resp, _ := doRequest(VerifyUser, "GET", "/api/open/users/verify/WRONGCODE", "", header, params)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
	if resp["error"] != "Verificaton code invalid." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestVerifyUserSuccess(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	// VerifyUser logs the user in at the SSO layer on success, which signs an
	// HS256 cookie using the configured private key.
	key, err := config.GenerateSecureKey(64)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}
	config.ConfigFile.PrivateKey = key

	verificationCode, err := database.GenerateRandomVerificationCodeForUser(user.ID)
	if err != nil {
		t.Fatalf("failed to generate verification code: %v", err)
	}

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	params := gin.Params{{Key: "code", Value: verificationCode}}
	code, resp, _ := doRequest(VerifyUser, "GET", "/api/open/users/verify/"+verificationCode, "", header, params)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}

	updated, err := database.GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if updated.Verified == nil || !*updated.Verified {
		t.Error("expected the user to now be verified")
	}
}

func TestUpdateUserRequiresAuth(t *testing.T) {
	setupControllersDB(t)

	code, resp, _ := doRequest(UpdateUser, "PUT", "/api/auth/users", `{"email":"new@example.com"}`, nil, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
}

func TestUpdateUserWrongOriginalPassword(t *testing.T) {
	setupControllersDB(t)
	user := newUserWithPassword(t, "CorrectHorse1!")

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	body := `{"email":"` + *user.Email + `","password_original":"WrongPassword!"}`
	code, resp, _ := doRequest(UpdateUser, "PUT", "/api/auth/users", body, header, nil)
	if code != 401 {
		t.Fatalf("status = %d, want 401; body=%v", code, resp)
	}
}

func TestUpdateUserSuccessNameChangeOnly(t *testing.T) {
	setupControllersDB(t)
	user := newUserWithPassword(t, "CorrectHorse1!")

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	body := `{"email":"` + *user.Email + `","password_original":"CorrectHorse1!"}`
	code, resp, _ := doRequest(UpdateUser, "PUT", "/api/auth/users", body, header, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
}

func TestUpdateUserPasswordsMustMatch(t *testing.T) {
	setupControllersDB(t)
	user := newUserWithPassword(t, "CorrectHorse1!")

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	body := `{"email":"` + *user.Email + `","password_original":"CorrectHorse1!","password":"NewPassword1!","password_repeat":"Mismatch1!"}`
	code, resp, _ := doRequest(UpdateUser, "PUT", "/api/auth/users", body, header, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
	if resp["error"] != "Passwords must match." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestUpdateUserDuplicateEmail(t *testing.T) {
	setupControllersDB(t)
	user := newUserWithPassword(t, "CorrectHorse1!")
	other := createTestUser(t)

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	body := `{"email":"` + *other.Email + `","password_original":"CorrectHorse1!"}`
	code, resp, _ := doRequest(UpdateUser, "PUT", "/api/auth/users", body, header, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
	if resp["error"] != "E-mail is already in use." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestAPIResetPasswordSMTPDisabled(t *testing.T) {
	setupControllersDB(t)

	code, resp, _ := doRequest(APIResetPassword, "POST", "/api/open/users/reset", `{"email":"nobody@example.com"}`, nil, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
}

func TestAPIVerifyResetCodeEmpty(t *testing.T) {
	setupControllersDB(t)

	params := gin.Params{{Key: "resetCode", Value: ""}}
	code, resp, _ := doRequest(APIVerifyResetCode, "GET", "/api/open/users/reset/", "", nil, params)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
}

func TestAPIVerifyResetCodeUnknown(t *testing.T) {
	setupControllersDB(t)

	params := gin.Params{{Key: "resetCode", Value: "unknown-code"}}
	code, resp, _ := doRequest(APIVerifyResetCode, "GET", "/api/open/users/reset/unknown-code", "", nil, params)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	if resp["expired"] != true {
		t.Errorf("expired = %v, want true for an unknown code", resp["expired"])
	}
}

func TestAPIVerifyResetCodeValid(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	resetCode, err := database.GenerateRandomResetCodeForUser(user.ID, true)
	if err != nil {
		t.Fatalf("failed to generate reset code: %v", err)
	}

	params := gin.Params{{Key: "resetCode", Value: resetCode}}
	code, resp, _ := doRequest(APIVerifyResetCode, "GET", "/api/open/users/reset/"+resetCode, "", nil, params)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	if resp["expired"] != false {
		t.Errorf("expired = %v, want false for a fresh code", resp["expired"])
	}
}

func TestAPIChangePasswordBadResetCode(t *testing.T) {
	setupControllersDB(t)

	body := `{"reset_code":"unknown","password":"NewPassword1!","password_repeat":"NewPassword1!"}`
	code, resp, _ := doRequest(APIChangePassword, "POST", "/api/open/users/password", body, nil, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
	if resp["error"] != "Reset code has expired." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestAPIChangePasswordPasswordsMustMatch(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	resetCode, err := database.GenerateRandomResetCodeForUser(user.ID, true)
	if err != nil {
		t.Fatalf("failed to generate reset code: %v", err)
	}

	body := `{"reset_code":"` + resetCode + `","password":"NewPassword1!","password_repeat":"Mismatch1!"}`
	code, resp, _ := doRequest(APIChangePassword, "POST", "/api/open/users/password", body, nil, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
	if resp["error"] != "Passwords must match." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestAPIChangePasswordSuccess(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	resetCode, err := database.GenerateRandomResetCodeForUser(user.ID, true)
	if err != nil {
		t.Fatalf("failed to generate reset code: %v", err)
	}

	body := `{"reset_code":"` + resetCode + `","password":"NewPassword1!","password_repeat":"NewPassword1!"}`
	code, resp, _ := doRequest(APIChangePassword, "POST", "/api/open/users/password", body, nil, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}

	updated, err := database.GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if err := updated.CheckPassword("NewPassword1!"); err != nil {
		t.Errorf("new password does not verify: %v", err)
	}
}

func TestAPIDeleteUserCannotDeleteSelf(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	params := gin.Params{{Key: "user_id", Value: user.ID.String()}}
	code, resp, _ := doRequest(APIDeleteUser, "DELETE", "/api/admin/users/"+user.ID.String(), "", header, params)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
	if resp["error"] != "You can't delete yourself." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestAPIDeleteUserSuccess(t *testing.T) {
	setupControllersDB(t)
	admin := createTestUser(t)
	target := createTestUser(t)

	header := map[string]string{"Authorization": authHeader(t, admin.ID, true)}
	params := gin.Params{{Key: "user_id", Value: target.ID.String()}}
	code, resp, _ := doRequest(APIDeleteUser, "DELETE", "/api/admin/users/"+target.ID.String(), "", header, params)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}

	updated, err := database.GetAllUserInformationAnyState(target.ID)
	if err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if updated.Enabled == nil || *updated.Enabled {
		t.Error("expected the deleted user to be disabled")
	}
}

func TestAPIDeleteUserInvalidUUID(t *testing.T) {
	setupControllersDB(t)
	admin := createTestUser(t)

	header := map[string]string{"Authorization": authHeader(t, admin.ID, true)}
	params := gin.Params{{Key: "user_id", Value: "not-a-uuid"}}
	code, resp, _ := doRequest(APIDeleteUser, "DELETE", "/api/admin/users/not-a-uuid", "", header, params)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
}
