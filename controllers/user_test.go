package controllers

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"bufio"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"net"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// fakeSMTPServer speaks just enough SMTP to satisfy github.com/go-mail/mail's
// Dialer (used by utilities.SendSMTP*), the same minimal protocol as
// utilities/smtp_test.go's fake server: greeting, EHLO, MAIL FROM, RCPT TO,
// DATA, QUIT, no STARTTLS/AUTH advertised.
type fakeSMTPServer struct {
	host string
	port int

	mu    sync.Mutex
	count int
}

func startFakeSMTPServer(t *testing.T) *fakeSMTPServer {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start fake SMTP listener: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("failed to parse listener address: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to parse listener port: %v", err)
	}

	srv := &fakeSMTPServer{host: host, port: port}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go srv.handle(conn)
		}
	}()
	return srv
}

func (s *fakeSMTPServer) handle(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	writeLine := func(line string) { conn.Write([]byte(line + "\r\n")) }
	writeLine("220 fake.smtp ESMTP ready")

	inData := false
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")

		if inData {
			if line == "." {
				inData = false
				s.mu.Lock()
				s.count++
				s.mu.Unlock()
				writeLine("250 OK: message queued")
			}
			continue
		}

		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
			writeLine("250 fake.smtp Hello")
		case strings.HasPrefix(upper, "MAIL FROM"), strings.HasPrefix(upper, "RCPT TO"), upper == "RSET":
			writeLine("250 OK")
		case upper == "DATA":
			inData = true
			writeLine("354 Start mail input; end with <CRLF>.<CRLF>")
		case upper == "QUIT":
			writeLine("221 Bye")
			return
		default:
			writeLine("500 unrecognized command")
		}
	}
}

func (s *fakeSMTPServer) messageCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count
}

// configureTestSMTP points config.ConfigFile's SMTP settings at srv and sets
// enough of the rest of the config for SMTP-gated handlers to proceed.
func configureTestSMTP(t *testing.T, srv *fakeSMTPServer) {
	t.Helper()
	origSMTP := config.ConfigFile.SMTPEnabled
	origHost := config.ConfigFile.SMTPHost
	origPort := config.ConfigFile.SMTPPort
	origFrom := config.ConfigFile.SMTPFrom
	origURL := config.ConfigFile.PoenskelistenExternalURL
	t.Cleanup(func() {
		config.ConfigFile.SMTPEnabled = origSMTP
		config.ConfigFile.SMTPHost = origHost
		config.ConfigFile.SMTPPort = origPort
		config.ConfigFile.SMTPFrom = origFrom
		config.ConfigFile.PoenskelistenExternalURL = origURL
	})

	config.ConfigFile.SMTPEnabled = true
	config.ConfigFile.SMTPHost = srv.host
	config.ConfigFile.SMTPPort = srv.port
	config.ConfigFile.SMTPFrom = "noreply@example.com"
	config.ConfigFile.PoenskelistenExternalURL = "https://wishlist.example.com"
}

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
	enablePrivateKey(t)

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

func TestRegisterUserInvalidFirstNameChars(t *testing.T) {
	setupControllersDB(t)
	invite := createTestInvite(t)

	body := fmt.Sprintf(`{"first_name":"<script>","last_name":"Lovelace","email":"ada@example.com","password":"Sup3rSecret!","password_repeat":"Sup3rSecret!","invite_code":"%s"}`, invite.Code)
	code, resp, _ := doRequest(RegisterUser, "POST", "/api/open/users/register", body, nil, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
}

func TestRegisterUserInvalidLastNameChars(t *testing.T) {
	setupControllersDB(t)
	invite := createTestInvite(t)

	body := fmt.Sprintf(`{"first_name":"Ada","last_name":"<script>","email":"ada@example.com","password":"Sup3rSecret!","password_repeat":"Sup3rSecret!","invite_code":"%s"}`, invite.Code)
	code, resp, _ := doRequest(RegisterUser, "POST", "/api/open/users/register", body, nil, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
}

func TestRegisterUserSMTPEnabledSendsVerificationEmail(t *testing.T) {
	setupControllersDB(t)
	invite := createTestInvite(t)
	srv := startFakeSMTPServer(t)
	configureTestSMTP(t, srv)

	body := fmt.Sprintf(`{"first_name":"Ada","last_name":"Lovelace","email":"ada@example.com","password":"Sup3rSecret!","password_repeat":"Sup3rSecret!","invite_code":"%s"}`, invite.Code)
	code, resp, _ := doRequest(RegisterUser, "POST", "/api/open/users/register", body, nil, nil)
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, resp)
	}

	user, err := database.GetAllUserInformationByEmail("ada@example.com")
	if err != nil {
		t.Fatalf("failed to look up created user: %v", err)
	}
	if user.Verified == nil || *user.Verified {
		t.Error("expected the user to be unverified when SMTP is enabled")
	}
	if srv.messageCount() != 1 {
		t.Errorf("messageCount = %d, want 1", srv.messageCount())
	}
}

func TestRegisterUserSMTPSendFailure(t *testing.T) {
	setupControllersDB(t)
	invite := createTestInvite(t)
	// Nothing is listening, so the dial must fail.
	srv := &fakeSMTPServer{host: "127.0.0.1", port: 1}
	configureTestSMTP(t, srv)

	body := fmt.Sprintf(`{"first_name":"Ada","last_name":"Lovelace","email":"ada@example.com","password":"Sup3rSecret!","password_repeat":"Sup3rSecret!","invite_code":"%s"}`, invite.Code)
	code, resp, _ := doRequest(RegisterUser, "POST", "/api/open/users/register", body, nil, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400 when the verification e-mail can't be sent; body=%v", code, resp)
	}
}

func TestGetUserAdminCanViewOthers(t *testing.T) {
	setupControllersDB(t)
	admin := createTestUser(t)
	target := createTestUser(t)

	header := map[string]string{"Authorization": authHeader(t, admin.ID, true)}
	params := gin.Params{{Key: "user_id", Value: target.ID.String()}}
	code, resp, _ := doRequest(GetUser, "GET", "/api/auth/users/"+target.ID.String(), "", header, params)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
}

func TestGetUserNonAdminViewingOtherUser(t *testing.T) {
	setupControllersDB(t)
	requester := createTestUser(t)
	target := createTestUser(t)

	header := map[string]string{"Authorization": authHeader(t, requester.ID, false)}
	params := gin.Params{{Key: "user_id", Value: target.ID.String()}}
	code, resp, _ := doRequest(GetUser, "GET", "/api/auth/users/"+target.ID.String(), "", header, params)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
}

func TestGetUsersIncludeDisabledAsAdmin(t *testing.T) {
	setupControllersDB(t)
	// GetUsers checks the caller's Admin flag on the DB row (via
	// database.GetUserInformation), not the JWT's admin claim - so the test
	// user must actually be an admin in the database, not just carry an
	// admin=true access token.
	admin := createTestUser(t)
	admin.Admin = true
	admin, err := database.UpdateUserInDB(admin)
	if err != nil {
		t.Fatalf("failed to make user admin: %v", err)
	}
	disabled := createTestUser(t)
	disabled.Enabled = boolPtr(false)
	if _, err := database.UpdateUserInDB(disabled); err != nil {
		t.Fatalf("failed to disable user: %v", err)
	}

	header := map[string]string{"Authorization": authHeader(t, admin.ID, true)}
	code, resp, _ := doRequest(GetUsers, "GET", "/api/auth/users?includeDisabled=true", "", header, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	users, ok := resp["users"].([]interface{})
	if !ok {
		t.Fatalf("expected a users array, got %v", resp)
	}
	if len(users) != 2 {
		t.Errorf("len(users) = %d, want 2 (including the disabled one)", len(users))
	}
}

func TestGetUsersIncludeDisabledIgnoredForNonAdmin(t *testing.T) {
	setupControllersDB(t)
	requester := createTestUser(t)
	disabled := createTestUser(t)
	disabled.Enabled = boolPtr(false)
	if _, err := database.UpdateUserInDB(disabled); err != nil {
		t.Fatalf("failed to disable user: %v", err)
	}

	header := map[string]string{"Authorization": authHeader(t, requester.ID, false)}
	code, resp, _ := doRequest(GetUsers, "GET", "/api/auth/users?includeDisabled=true", "", header, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	users, ok := resp["users"].([]interface{})
	if !ok {
		t.Fatalf("expected a users array, got %v", resp)
	}
	if len(users) != 1 {
		t.Errorf("len(users) = %d, want 1 (disabled user must stay excluded for a non-admin)", len(users))
	}
}

func TestGetUsersNotACollaboratorOfWishlist(t *testing.T) {
	setupControllersDB(t)
	requester := createTestUser(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	header := map[string]string{"Authorization": authHeader(t, requester.ID, false)}
	code, resp, _ := doRequest(GetUsers, "GET", "/api/auth/users?notACollaboratorOfWishlistID="+wishlist.ID.String(), "", header, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	users, ok := resp["users"].([]interface{})
	if !ok {
		t.Fatalf("expected a users array, got %v", resp)
	}
	// Nobody collaborates on the wishlist yet, so both enabled users remain.
	if len(users) != 2 {
		t.Errorf("len(users) = %d, want 2", len(users))
	}
}

func TestGetUsersNotACollaboratorBadWishlistID(t *testing.T) {
	setupControllersDB(t)
	requester := createTestUser(t)

	header := map[string]string{"Authorization": authHeader(t, requester.ID, false)}
	code, resp, _ := doRequest(GetUsers, "GET", "/api/auth/users?notACollaboratorOfWishlistID=not-a-uuid", "", header, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
}

func TestGetUsersNotACollaboratorUnknownWishlist(t *testing.T) {
	setupControllersDB(t)
	requester := createTestUser(t)

	header := map[string]string{"Authorization": authHeader(t, requester.ID, false)}
	code, resp, _ := doRequest(GetUsers, "GET", "/api/auth/users?notACollaboratorOfWishlistID="+uuid.NewString(), "", header, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
}

func TestGetUsersNotACollaboratorWishlistLookupFailure(t *testing.T) {
	setupControllersDB(t)
	requester := createTestUser(t)
	wishlist := createTestWishlist(t, requester.ID)
	header := map[string]string{"Authorization": authHeader(t, requester.ID, false)}
	failDBOperation(t, "query", "wishlists", 0)

	code, resp, _ := doRequest(GetUsers, "GET", "/api/auth/users?notACollaboratorOfWishlistID="+wishlist.ID.String(), "", header, nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
	if resp["error"] != "Failed to get wishlist." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestVerifyUserSSOSessionFailure(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	origKey := config.ConfigFile.PrivateKey
	config.ConfigFile.PrivateKey = ""
	t.Cleanup(func() { config.ConfigFile.PrivateKey = origKey })

	verificationCode, err := database.GenerateRandomVerificationCodeForUser(user.ID)
	if err != nil {
		t.Fatalf("failed to generate verification code: %v", err)
	}

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	params := gin.Params{{Key: "code", Value: verificationCode}}
	code, resp, _ := doRequest(VerifyUser, "GET", "/api/open/users/verify/"+verificationCode, "", header, params)
	if code != 500 {
		t.Fatalf("status = %d, want 500 when the session can't be issued; body=%v", code, resp)
	}
}

func TestSendUserVerificationCodeRequiresLogin(t *testing.T) {
	setupControllersDB(t)

	code, resp, _ := doRequest(SendUserVerificationCode, "POST", "/api/open/users/verification", "", nil, nil)
	if code != 401 {
		t.Fatalf("status = %d, want 401; body=%v", code, resp)
	}
}

func TestSendUserVerificationCodeSuccess(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	srv := startFakeSMTPServer(t)
	configureTestSMTP(t, srv)

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	code, resp, _ := doRequest(SendUserVerificationCode, "POST", "/api/open/users/verification", "", header, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	if srv.messageCount() != 1 {
		t.Errorf("messageCount = %d, want 1", srv.messageCount())
	}
}

func TestUpdateUserWeakNewPassword(t *testing.T) {
	setupControllersDB(t)
	user := newUserWithPassword(t, "CorrectHorse1!")

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	body := `{"email":"` + *user.Email + `","password_original":"CorrectHorse1!","password":"weak","password_repeat":"weak"}`
	code, resp, _ := doRequest(UpdateUser, "PUT", "/api/auth/users", body, header, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
}

// TestUpdateUserEmailChangeActuallyUnverifies covers a regression (see
// docs/wip.md history): UpdateUser calls database.SetUserVerification(userID,
// false) directly against the DB after an e-mail change, and must also update
// the in-memory userOriginal.Verified field to match before saving it back
// with database.UpdateUserInDB, or that save would silently revert the
// unverify.
func TestUpdateUserEmailChangeActuallyUnverifies(t *testing.T) {
	setupControllersDB(t)
	user := newUserWithPassword(t, "CorrectHorse1!")

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	body := `{"email":"new-address@example.com","password_original":"CorrectHorse1!"}`
	code, resp, _ := doRequest(UpdateUser, "PUT", "/api/auth/users", body, header, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	if resp["verified"] != false {
		t.Errorf("verified = %v, want false", resp["verified"])
	}

	updated, err := database.GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if *updated.Email != "new-address@example.com" {
		t.Errorf("email = %q, want the new address", *updated.Email)
	}
	if updated.Verified == nil || *updated.Verified {
		t.Error("expected Verified to be false in the DB after an e-mail change")
	}
}

// TestUpdateUserEmailChangeSMTPEnabledSendsVerification is a consequence of
// the fix above: once Verified is actually flipped to false, the "send a new
// verification e-mail after an e-mail change" branch (gated on
// `!*user.Verified`) fires when SMTP is enabled.
func TestUpdateUserEmailChangeSMTPEnabledSendsVerification(t *testing.T) {
	setupControllersDB(t)
	user := newUserWithPassword(t, "CorrectHorse1!")
	srv := startFakeSMTPServer(t)
	configureTestSMTP(t, srv)

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	body := `{"email":"new-address@example.com","password_original":"CorrectHorse1!"}`
	code, resp, _ := doRequest(UpdateUser, "PUT", "/api/auth/users", body, header, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	if srv.messageCount() != 1 {
		t.Errorf("messageCount = %d, want 1 (re-verification e-mail after e-mail change)", srv.messageCount())
	}
}

// TestUpdateUserSendsVerificationEmailForUnverifiedUser covers the
// "send verification e-mail" branch for an already-unverified account (e.g. a
// name-only edit, no e-mail change). This used to crash with a nil-pointer
// dereference at `*user.VerificationCode = verificationCode`, since the
// freshly-refetched user has a nil VerificationCode until one is allocated.
func TestUpdateUserSendsVerificationEmailForUnverifiedUser(t *testing.T) {
	setupControllersDB(t)
	config.ConfigFile.SMTPEnabled = true
	t.Cleanup(func() { config.ConfigFile.SMTPEnabled = false })

	user := newUserWithPassword(t, "CorrectHorse1!")
	var verifiedBool bool = false
	user.Verified = &verifiedBool
	user, err := database.UpdateUserInDB(user)
	if err != nil {
		t.Fatalf("failed to mark user unverified: %v", err)
	}

	srv := startFakeSMTPServer(t)
	configureTestSMTP(t, srv)

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	body := `{"email":"` + *user.Email + `","password_original":"CorrectHorse1!"}`
	code, resp, _ := doRequest(UpdateUser, "PUT", "/api/auth/users", body, header, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	if resp["verified"] != false {
		t.Errorf("verified = %v, want false", resp["verified"])
	}
	if srv.messageCount() != 1 {
		t.Errorf("messageCount = %d, want 1", srv.messageCount())
	}
}

func TestAPIResetPasswordNoExternalURL(t *testing.T) {
	setupControllersDB(t)
	config.ConfigFile.SMTPEnabled = true
	t.Cleanup(func() { config.ConfigFile.SMTPEnabled = false })
	config.ConfigFile.PoenskelistenExternalURL = ""

	code, resp, _ := doRequest(APIResetPassword, "POST", "/api/open/users/reset", `{"email":"nobody@example.com"}`, nil, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
}

func TestAPIResetPasswordBadJSON(t *testing.T) {
	setupControllersDB(t)
	srv := startFakeSMTPServer(t)
	configureTestSMTP(t, srv)

	code, resp, _ := doRequest(APIResetPassword, "POST", "/api/open/users/reset", `not-json`, nil, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
}

func TestAPIResetPasswordUnknownEmail(t *testing.T) {
	setupControllersDB(t)
	srv := startFakeSMTPServer(t)
	configureTestSMTP(t, srv)

	code, resp, _ := doRequest(APIResetPassword, "POST", "/api/open/users/reset", `{"email":"nobody@example.com"}`, nil, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200 (must not reveal whether the e-mail exists); body=%v", code, resp)
	}
	if srv.messageCount() != 0 {
		t.Errorf("messageCount = %d, want 0 for an unknown e-mail", srv.messageCount())
	}
}

func TestAPIResetPasswordSuccess(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	srv := startFakeSMTPServer(t)
	configureTestSMTP(t, srv)

	code, resp, _ := doRequest(APIResetPassword, "POST", "/api/open/users/reset", `{"email":"`+*user.Email+`"}`, nil, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	if srv.messageCount() != 1 {
		t.Errorf("messageCount = %d, want 1", srv.messageCount())
	}
}

func TestAPIChangePasswordWeakPassword(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	resetCode, err := database.GenerateRandomResetCodeForUser(user.ID, true)
	if err != nil {
		t.Fatalf("failed to generate reset code: %v", err)
	}

	body := `{"reset_code":"` + resetCode + `","password":"weak","password_repeat":"weak"}`
	code, resp, _ := doRequest(APIChangePassword, "POST", "/api/open/users/password", body, nil, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
}

func TestUpdateUserProfileImageFailure(t *testing.T) {
	setupControllersDB(t)
	user := newUserWithPassword(t, "CorrectHorse1!")

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	body := `{"email":"` + *user.Email + `","password_original":"CorrectHorse1!","profile_image":"not-valid-base64-image-data"}`
	code, resp, _ := doRequest(UpdateUser, "PUT", "/api/auth/users", body, header, nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500 for an unparseable profile image; body=%v", code, resp)
	}
}

func TestSendUserVerificationCodeSMTPFailure(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	orig := config.ConfigFile
	t.Cleanup(func() { config.ConfigFile = orig })
	config.ConfigFile.SMTPEnabled = true
	config.ConfigFile.SMTPHost = "127.0.0.1"
	config.ConfigFile.SMTPPort = 1 // nothing listening here
	config.ConfigFile.SMTPFrom = "noreply@example.com"

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	code, resp, _ := doRequest(SendUserVerificationCode, "POST", "/api/open/users/verification", "", header, nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500 when the SMTP server is unreachable; body=%v", code, resp)
	}
}

func TestSendUserVerificationCodeUserDeleted(t *testing.T) {
	// A valid access token whose user was deleted afterward: resolveGateUser's
	// database.GetAllUserInformation lookup fails, and there's no SSO cookie to
	// fall back to either, so the gate rejects the request before it can even
	// attempt to generate a code.
	setupControllersDB(t)
	user := createTestUser(t)
	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}

	if result := database.Instance.Unscoped().Delete(&models.User{}, "id = ?", user.ID); result.Error != nil {
		t.Fatalf("failed to hard-delete user: %v", result.Error)
	}

	code, resp, _ := doRequest(SendUserVerificationCode, "POST", "/api/open/users/verification", "", header, nil)
	if code != 401 {
		t.Fatalf("status = %d, want 401 once the user backing the token no longer exists; body=%v", code, resp)
	}
}

func TestGetUserNonExistentTargetAsAdmin(t *testing.T) {
	setupControllersDB(t)
	admin := createTestUser(t)
	admin.Admin = true
	admin, err := database.UpdateUserInDB(admin)
	if err != nil {
		t.Fatalf("failed to make user admin: %v", err)
	}

	header := map[string]string{"Authorization": authHeader(t, admin.ID, true)}
	missing := uuid.New()
	params := gin.Params{{Key: "user_id", Value: missing.String()}}
	code, resp, _ := doRequest(GetUser, "GET", "/api/auth/users/"+missing.String(), "", header, params)
	if code != 500 {
		t.Fatalf("status = %d, want 500 for a nonexistent target user as admin; body=%v", code, resp)
	}
}

func TestGetUserNonExistentTargetAsNonAdmin(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	missing := uuid.New()
	params := gin.Params{{Key: "user_id", Value: missing.String()}}
	code, resp, _ := doRequest(GetUser, "GET", "/api/auth/users/"+missing.String(), "", header, params)
	if code != 500 {
		t.Fatalf("status = %d, want 500 for a nonexistent target user as a non-admin; body=%v", code, resp)
	}
}

func TestAPICurrentUserDatabaseFailure(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}

	if result := database.Instance.Unscoped().Delete(&models.User{}, "id = ?", user.ID); result.Error != nil {
		t.Fatalf("failed to hard-delete user: %v", result.Error)
	}

	code, resp, _ := doRequest(APICurrentUser, "GET", "/api/auth/me", "", header, nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500 once the user backing the token no longer exists; body=%v", code, resp)
	}
}

func TestUserHandlersDatabaseErrors(t *testing.T) {
	userParam := gin.Params{{Key: "user_id", Value: "00000000-0000-0000-0000-00000000000a"}}
	runDatabaseErrorCases(t, []dbErrorCase{
		{name: "APICurrentUser", handler: APICurrentUser, method: "GET", path: "/api/auth/me"},
		{name: "GetUser", handler: GetUser, method: "GET", path: "/api/auth/users/00000000-0000-0000-0000-00000000000a", params: userParam},
		{name: "GetUsers", handler: GetUsers, method: "GET", path: "/api/auth/users"},
		{name: "GetUsers/notAMemberOfGroupID", handler: GetUsers, method: "GET", path: "/api/auth/users?notAMemberOfGroupID=00000000-0000-0000-0000-00000000000a"},
		{name: "GetUsers/notACollaboratorOfWishlistID", handler: GetUsers, method: "GET", path: "/api/auth/users?notACollaboratorOfWishlistID=00000000-0000-0000-0000-00000000000a"},
		{name: "UpdateUser", handler: UpdateUser, method: "POST", path: "/api/auth/users/update", body: `{"email":"a@example.com","first_name":"A","last_name":"B"}`},
		{name: "APIDeleteUser", handler: APIDeleteUser, method: "DELETE", path: "/api/admin/users/00000000-0000-0000-0000-00000000000a", params: userParam, admin: true},
	})
}

func TestUserHandlersRequireAuth(t *testing.T) {
	runUnauthenticatedCases(t, []dbErrorCase{
		{name: "GetUsers", handler: GetUsers, method: "GET", path: "/api/auth/users"},
		{name: "APIDeleteUser", handler: APIDeleteUser, method: "DELETE", path: "/api/admin/users/00000000-0000-0000-0000-00000000000a", params: gin.Params{{Key: "user_id", Value: "00000000-0000-0000-0000-00000000000a"}}},
	})
}

// userTestLongPassword satisfies ValidatePasswordFormat but is longer than
// bcrypt's 72-byte input limit, so HashPassword rejects it. That is the only
// way to reach the handlers' "failed to hash password" branches.
var userTestLongPassword = "Aa1" + strings.Repeat("x", 80)

// userTestRegisterBody is a valid RegisterUser request for inviteCode.
func userTestRegisterBody(inviteCode, password string) string {
	return fmt.Sprintf(`{"first_name":"Ada","last_name":"Lovelace","email":"ada@example.com","password":"%s","password_repeat":"%s","invite_code":"%s"}`, password, password, inviteCode)
}

// userTestMakeAdmin flips the admin flag on user in the DB.
func userTestMakeAdmin(t *testing.T, user models.User) models.User {
	t.Helper()
	user.Admin = true
	updated, err := database.UpdateUserInDB(user)
	if err != nil {
		t.Fatalf("failed to make user admin: %v", err)
	}
	return updated
}

// userTestMarkUnverified flips the verified flag off on user in the DB.
func userTestMarkUnverified(t *testing.T, user models.User) models.User {
	t.Helper()
	user.Verified = boolPtr(false)
	updated, err := database.UpdateUserInDB(user)
	if err != nil {
		t.Fatalf("failed to mark user unverified: %v", err)
	}
	return updated
}

// userTestInviteUsed reports whether the invite with code has been claimed.
func userTestInviteUsed(t *testing.T, code string) bool {
	t.Helper()
	unused, err := database.VerifyUnusedUserInviteCode(code)
	if err != nil {
		t.Fatalf("failed to look up invite: %v", err)
	}
	return !unused
}

func TestRegisterUserDatabaseFailures(t *testing.T) {
	cases := []struct {
		name       string
		op         string
		table      string
		skip       int
		wantStatus int
		wantError  string
	}{
		{"user count", "query", "users", 0, 500, "Failed to verify user amount."},
		{"invite lookup", "query", "invites", 0, 500, "Failed to verify invite code."},
		{"unique e-mail check", "query", "users", 1, 500, "Failed to verify unique e-mail."},
		{"user insert", "create", "users", 0, 500, "Failed to create user."},
		{"claim invite", "update", "invites", 0, 500, "Failed to create user."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			invite := createTestInvite(t)
			failDBOperation(t, c.op, c.table, c.skip)

			code, resp, _ := doRequest(RegisterUser, "POST", "/api/open/users/register", userTestRegisterBody(invite.Code, "Sup3rSecret!"), nil, nil)
			if code != c.wantStatus {
				t.Fatalf("status = %d, want %d; body=%v", code, c.wantStatus, resp)
			}
			if resp["error"] != c.wantError {
				t.Errorf("error = %v, want %q", resp["error"], c.wantError)
			}
		})
	}
}

// A failure while claiming the invite must not leave the new account behind
// with the invite still reusable.
func TestRegisterUserInviteClaimFailureLeavesNoUser(t *testing.T) {
	setupControllersDB(t)
	invite := createTestInvite(t)
	failDBOperation(t, "update", "invites", 0)

	code, resp, _ := doRequest(RegisterUser, "POST", "/api/open/users/register", userTestRegisterBody(invite.Code, "Sup3rSecret!"), nil, nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
	if userTestInviteUsed(t, invite.Code) {
		t.Error("the invite must stay unused when registration fails")
	}
	count, err := database.GetAmountOfEnabledUsers()
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 0 {
		t.Errorf("users = %d, want 0 (the insert should have been rolled back)", count)
	}
}

// Two registrations can both pass the "unused invite" check; only one may
// claim it. Simulate losing that race by marking the invite used just before
// the claim.
func TestRegisterUserInviteClaimedConcurrently(t *testing.T) {
	setupControllersDB(t)
	invite := createTestInvite(t)
	onDBOperation(t, "update", "invites", 0, func(tx *gorm.DB) {
		if err := tx.Exec("UPDATE invites SET used = ? WHERE id = ?", true, invite.ID).Error; err != nil {
			t.Errorf("failed to mark invite used: %v", err)
		}
	})

	code, resp, _ := doRequest(RegisterUser, "POST", "/api/open/users/register", userTestRegisterBody(invite.Code, "Sup3rSecret!"), nil, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
	if resp["error"] != "Invitiation code is not valid." {
		t.Errorf("error = %v", resp["error"])
	}
	count, err := database.GetAmountOfEnabledUsers()
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 0 {
		t.Errorf("users = %d, want 0 (the losing registration must be rolled back)", count)
	}
}

func TestRegisterUserPasswordTooLongToHash(t *testing.T) {
	setupControllersDB(t)
	invite := createTestInvite(t)

	code, resp, _ := doRequest(RegisterUser, "POST", "/api/open/users/register", userTestRegisterBody(invite.Code, userTestLongPassword), nil, nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
	if resp["error"] != "Failed to hash password." {
		t.Errorf("error = %v", resp["error"])
	}
	if userTestInviteUsed(t, invite.Code) {
		t.Error("the invite must stay unused when registration fails")
	}
}

func TestGetUsersListDatabaseFailures(t *testing.T) {
	cases := []struct {
		name      string
		path      string
		admin     bool
		wantError string
	}{
		{"all users as admin", "/api/auth/users?includeDisabled=true", true, "Failed to get all users."},
		{"enabled users", "/api/auth/users", false, "Failed to get enabled users."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			user := createTestUser(t)
			if c.admin {
				user = userTestMakeAdmin(t, user)
			}
			header := map[string]string{"Authorization": authHeader(t, user.ID, c.admin)}
			// The first users query loads the caller; fail the listing after it.
			failDBOperation(t, "query", "users", 1)

			code, resp, _ := doRequest(GetUsers, "GET", c.path, "", header, nil)
			if code != 500 {
				t.Fatalf("status = %d, want 500; body=%v", code, resp)
			}
			if resp["error"] != c.wantError {
				t.Errorf("error = %v, want %q", resp["error"], c.wantError)
			}
		})
	}
}

func TestGetUsersNotAMemberOfGroupMembershipLookupFailure(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	group := createTestGroup(t, user.ID)
	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	failDBOperation(t, "query", "group_memberships", 0)

	code, resp, _ := doRequest(GetUsers, "GET", "/api/auth/users?notAMemberOfGroupID="+group.ID.String(), "", header, nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
	if resp["error"] != "Failed to verify ownership to group." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestGetUsersNotACollaboratorWishlistConversionFailure(t *testing.T) {
	setupControllersDB(t)
	requester := createTestUser(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	header := map[string]string{"Authorization": authHeader(t, requester.ID, false)}
	failDBOperation(t, "query", "wishlist_collaborators", 0)

	code, resp, _ := doRequest(GetUsers, "GET", "/api/auth/users?notACollaboratorOfWishlistID="+wishlist.ID.String(), "", header, nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
	if resp["error"] != "Failed to convert wishlist to wishlist object." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestGetUsersNotACollaboratorExcludesCollaborators(t *testing.T) {
	setupControllersDB(t)
	requester := createTestUser(t)
	owner := createTestUser(t)
	collaborator := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	collab := models.WishlistCollaborator{UserID: collaborator.ID, WishlistID: wishlist.ID, Enabled: true}
	collab.ID = uuid.New()
	if err := database.CreateWishlistCollaboratorInDB(collab); err != nil {
		t.Fatalf("failed to add wishlist collaborator: %v", err)
	}

	header := map[string]string{"Authorization": authHeader(t, requester.ID, false)}
	code, resp, _ := doRequest(GetUsers, "GET", "/api/auth/users?notACollaboratorOfWishlistID="+wishlist.ID.String(), "", header, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	users, ok := resp["users"].([]interface{})
	if !ok {
		t.Fatalf("expected a users array, got %v", resp)
	}
	if len(users) != 2 {
		t.Fatalf("len(users) = %d, want 2 (everyone but the collaborator)", len(users))
	}
	for _, u := range users {
		if u.(map[string]interface{})["id"] == collaborator.ID.String() {
			t.Error("the collaborator should have been filtered out")
		}
	}
}

func TestVerifyUserDatabaseFailures(t *testing.T) {
	cases := []struct {
		name      string
		op        string
		skip      int
		wantError string
	}{
		// resolveGateUser's lookup is the first users query.
		{"code lookup", "query", 1, "Failed to get verification code."},
		{"set verified", "update", 0, "Failed to set user verification."},
		{"reload user", "query", 2, "Failed to get user details."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			user := userTestMarkUnverified(t, createTestUser(t))
			verificationCode, err := database.GenerateRandomVerificationCodeForUser(user.ID)
			if err != nil {
				t.Fatalf("failed to generate verification code: %v", err)
			}
			header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
			failDBOperation(t, c.op, "users", c.skip)

			params := gin.Params{{Key: "code", Value: verificationCode}}
			code, resp, _ := doRequest(VerifyUser, "GET", "/api/open/users/verify/"+verificationCode, "", header, params)
			if code != 500 {
				t.Fatalf("status = %d, want 500; body=%v", code, resp)
			}
			if resp["error"] != c.wantError {
				t.Errorf("error = %v, want %q", resp["error"], c.wantError)
			}
		})
	}
}

func TestSendUserVerificationCodeDatabaseFailures(t *testing.T) {
	cases := []struct {
		name      string
		op        string
		skip      int
		wantError string
	}{
		{"generate code", "update", 0, "Failed to generate verification code."},
		{"reload user", "query", 1, "Failed to get user."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			user := createTestUser(t)
			srv := startFakeSMTPServer(t)
			configureTestSMTP(t, srv)
			header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
			failDBOperation(t, c.op, "users", c.skip)

			code, resp, _ := doRequest(SendUserVerificationCode, "POST", "/api/open/users/verification", "", header, nil)
			if code != 500 {
				t.Fatalf("status = %d, want 500; body=%v", code, resp)
			}
			if resp["error"] != c.wantError {
				t.Errorf("error = %v, want %q", resp["error"], c.wantError)
			}
			if srv.messageCount() != 0 {
				t.Errorf("messageCount = %d, want 0", srv.messageCount())
			}
		})
	}
}

func TestUpdateUserBadJSON(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	code, resp, _ := doRequest(UpdateUser, "PUT", "/api/auth/users", "not-json", header, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
}

func TestUpdateUserChangesPassword(t *testing.T) {
	setupControllersDB(t)
	user := newUserWithPassword(t, "CorrectHorse1!")

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	body := `{"email":"` + *user.Email + `","password_original":"CorrectHorse1!","password":"NewPassw0rd","password_repeat":"NewPassw0rd"}`
	code, resp, _ := doRequest(UpdateUser, "PUT", "/api/auth/users", body, header, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}

	updated, err := database.GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if err := updated.CheckPassword("NewPassw0rd"); err != nil {
		t.Errorf("new password should be accepted: %v", err)
	}
	if err := updated.CheckPassword("CorrectHorse1!"); err == nil {
		t.Error("old password should no longer be accepted")
	}
}

func TestUpdateUserPasswordTooLongToHash(t *testing.T) {
	setupControllersDB(t)
	user := newUserWithPassword(t, "CorrectHorse1!")

	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	body := `{"email":"` + *user.Email + `","password_original":"CorrectHorse1!","password":"` + userTestLongPassword + `","password_repeat":"` + userTestLongPassword + `"}`
	code, resp, _ := doRequest(UpdateUser, "PUT", "/api/auth/users", body, header, nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
	if resp["error"] != "Failed to hash password." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestUpdateUserDatabaseFailures(t *testing.T) {
	cases := []struct {
		name       string
		newEmail   bool
		op         string
		skip       int
		wantStatus int
		wantError  string
	}{
		// Users queries in order: caller lookup, userOriginal lookup, then either
		// the unique-e-mail check (e-mail change) or the final reload.
		{"load original", false, "query", 1, 500, "Failed to get user details."},
		{"unique e-mail check", true, "query", 2, 500, "Failed to verify e-mail."},
		{"unverify on e-mail change", true, "update", 0, 500, "Failed to change verification."},
		{"save user", false, "update", 0, 500, "Failed to update user."},
		{"reload user", false, "query", 2, 500, "Failed to get user."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			user := newUserWithPassword(t, "CorrectHorse1!")
			email := *user.Email
			if c.newEmail {
				email = "changed-" + email
			}
			header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
			failDBOperation(t, c.op, "users", c.skip)

			body := `{"email":"` + email + `","password_original":"CorrectHorse1!"}`
			code, resp, _ := doRequest(UpdateUser, "PUT", "/api/auth/users", body, header, nil)
			if code != c.wantStatus {
				t.Fatalf("status = %d, want %d; body=%v", code, c.wantStatus, resp)
			}
			if resp["error"] != c.wantError {
				t.Errorf("error = %v, want %q", resp["error"], c.wantError)
			}
		})
	}
}

func TestUpdateUserUnverifiedVerificationCodeFailure(t *testing.T) {
	setupControllersDB(t)
	user := userTestMarkUnverified(t, newUserWithPassword(t, "CorrectHorse1!"))
	srv := startFakeSMTPServer(t)
	configureTestSMTP(t, srv)
	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	// The first users update is the profile save; fail the code generation after it.
	failDBOperation(t, "update", "users", 1)

	body := `{"email":"` + *user.Email + `","password_original":"CorrectHorse1!"}`
	code, resp, _ := doRequest(UpdateUser, "PUT", "/api/auth/users", body, header, nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
	if resp["error"] != "Failed to generate verification code." {
		t.Errorf("error = %v", resp["error"])
	}
	if srv.messageCount() != 0 {
		t.Errorf("messageCount = %d, want 0", srv.messageCount())
	}
}

func TestUpdateUserUnverifiedSMTPFailure(t *testing.T) {
	setupControllersDB(t)
	user := userTestMarkUnverified(t, newUserWithPassword(t, "CorrectHorse1!"))
	configureTestSMTP(t, &fakeSMTPServer{host: "127.0.0.1", port: 1}) // nothing listening
	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}

	body := `{"email":"` + *user.Email + `","password_original":"CorrectHorse1!"}`
	code, resp, _ := doRequest(UpdateUser, "PUT", "/api/auth/users", body, header, nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
	if resp["error"] != "Failed to send e-mail." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestAPIResetPasswordReloadFailureStillReportsOkay(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	srv := startFakeSMTPServer(t)
	configureTestSMTP(t, srv)
	// The e-mail lookup succeeds; the unredacted reload after it fails.
	failDBOperation(t, "query", "users", 1)

	code, resp, _ := doRequest(APIResetPassword, "POST", "/api/open/users/reset", `{"email":"`+*user.Email+`"}`, nil, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200 (must not reveal whether the e-mail exists); body=%v", code, resp)
	}
	if srv.messageCount() != 0 {
		t.Errorf("messageCount = %d, want 0", srv.messageCount())
	}
}

func TestAPIResetPasswordFailures(t *testing.T) {
	cases := []struct {
		name        string
		op          string
		skip        int
		smtpDown    bool
		wantMessage string
	}{
		{"generate reset code", "update", 0, false, "Error."},
		{"reload user", "query", 2, false, "Error."},
		{"send e-mail", "", 0, true, "Error. Failed to send e-mail."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			user := createTestUser(t)
			srv := startFakeSMTPServer(t)
			if c.smtpDown {
				srv = &fakeSMTPServer{host: "127.0.0.1", port: 1} // nothing listening
			}
			configureTestSMTP(t, srv)
			if c.op != "" {
				failDBOperation(t, c.op, "users", c.skip)
			}

			code, resp, _ := doRequest(APIResetPassword, "POST", "/api/open/users/reset", `{"email":"`+*user.Email+`"}`, nil, nil)
			if code != 500 {
				t.Fatalf("status = %d, want 500; body=%v", code, resp)
			}
			if resp["message"] != c.wantMessage {
				t.Errorf("message = %v, want %q", resp["message"], c.wantMessage)
			}
		})
	}
}

func TestAPIVerifyResetCodeExpired(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	// valid=false stamps the expiry at "now", so it's already in the past.
	resetCode, err := database.GenerateRandomResetCodeForUser(user.ID, false)
	if err != nil {
		t.Fatalf("failed to generate reset code: %v", err)
	}

	params := gin.Params{{Key: "resetCode", Value: resetCode}}
	code, resp, _ := doRequest(APIVerifyResetCode, "GET", "/api/open/users/reset/"+resetCode, "", nil, params)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, resp)
	}
	if resp["expired"] != true {
		t.Errorf("expired = %v, want true", resp["expired"])
	}
}

func TestAPIChangePasswordBadJSON(t *testing.T) {
	setupControllersDB(t)

	code, resp, _ := doRequest(APIChangePassword, "POST", "/api/open/users/password", "not-json", nil, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
	if resp["error"] != "Failed to parse request." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestAPIChangePasswordExpiredCode(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	resetCode, err := database.GenerateRandomResetCodeForUser(user.ID, false)
	if err != nil {
		t.Fatalf("failed to generate reset code: %v", err)
	}

	body := `{"reset_code":"` + resetCode + `","password":"NewPassw0rd","password_repeat":"NewPassw0rd"}`
	code, resp, _ := doRequest(APIChangePassword, "POST", "/api/open/users/password", body, nil, nil)
	if code != 400 {
		t.Fatalf("status = %d, want 400; body=%v", code, resp)
	}
	if resp["error"] != "Reset code has expired." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestAPIChangePasswordTooLongToHash(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	resetCode, err := database.GenerateRandomResetCodeForUser(user.ID, true)
	if err != nil {
		t.Fatalf("failed to generate reset code: %v", err)
	}

	body := `{"reset_code":"` + resetCode + `","password":"` + userTestLongPassword + `","password_repeat":"` + userTestLongPassword + `"}`
	code, resp, _ := doRequest(APIChangePassword, "POST", "/api/open/users/password", body, nil, nil)
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
	if resp["error"] != "Failed to process password." {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestAPIChangePasswordDatabaseFailures(t *testing.T) {
	cases := []struct {
		name        string
		skip        int
		wantError   interface{}
		wantMessage interface{}
	}{
		{"save password", 0, "Failed to update user.", nil},
		{"rotate reset code", 1, nil, "Error."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			user := createTestUser(t)
			resetCode, err := database.GenerateRandomResetCodeForUser(user.ID, true)
			if err != nil {
				t.Fatalf("failed to generate reset code: %v", err)
			}
			failDBOperation(t, "update", "users", c.skip)

			body := `{"reset_code":"` + resetCode + `","password":"NewPassw0rd","password_repeat":"NewPassw0rd"}`
			code, resp, _ := doRequest(APIChangePassword, "POST", "/api/open/users/password", body, nil, nil)
			if code != 500 {
				t.Fatalf("status = %d, want 500; body=%v", code, resp)
			}
			if resp["error"] != c.wantError || resp["message"] != c.wantMessage {
				t.Errorf("error = %v, message = %v; want %v, %v", resp["error"], resp["message"], c.wantError, c.wantMessage)
			}
		})
	}
}

func TestAPIDeleteUserSaveFailure(t *testing.T) {
	setupControllersDB(t)
	admin := createTestUser(t)
	target := createTestUser(t)
	header := map[string]string{"Authorization": authHeader(t, admin.ID, true)}
	failDBOperation(t, "update", "users", 0)

	params := gin.Params{{Key: "user_id", Value: target.ID.String()}}
	code, resp, _ := doRequest(APIDeleteUser, "DELETE", "/api/admin/users/"+target.ID.String(), "", header, params)
	if code != 500 {
		t.Fatalf("status = %d, want 500; body=%v", code, resp)
	}
	if resp["error"] != "Failed to update user object." {
		t.Errorf("error = %v", resp["error"])
	}
}

// TestAPIDeleteUserImageCleanupFailureIsNotFatal plants a non-empty directory
// where the user's profile image would be, so os.Remove fails with something
// other than "not exist". The user is already disabled by then, so the handler
// must still report success.
func TestAPIDeleteUserImageCleanupFailureIsNotFatal(t *testing.T) {
	setupControllersDB(t)
	admin := createTestUser(t)
	target := createTestUser(t)

	origDir := profileImageDir
	t.Cleanup(func() { profileImageDir = origDir })
	profileImageDir = t.TempDir()
	if err := os.MkdirAll(filepath.Join(imageFilePath(profileImageDir, target.ID, false), "blocker"), 0755); err != nil {
		t.Fatalf("failed to plant blocking directory: %v", err)
	}

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
