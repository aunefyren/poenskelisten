package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestGetNewsRequiresAuth(t *testing.T) {
	setupControllersDB(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/news", nil)
	GetNews(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 without auth", w.Code)
	}
}

func TestGetNewsFiltersFutureAndExpiredForNonAdmin(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	news := createTestNews(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/news", nil)
	ctx.Request.Header.Set("Authorization", authHeader(t, user.ID, false))
	GetNews(ctx)

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}

	var body struct {
		News []struct {
			ID string `json:"ID"`
		} `json:"news"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	found := false
	for _, n := range body.News {
		if n.ID == news.ID.String() {
			found = true
		}
	}
	if !found {
		t.Errorf("expected the past-dated news post to be included, got %+v", body.News)
	}
}

func TestGetNewsUserNotFound(t *testing.T) {
	setupControllersDB(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/news", nil)
	ctx.Request.Header.Set("Authorization", authHeader(t, uuid.New(), false))
	GetNews(ctx)

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the token's user doesn't exist", w.Code)
	}
}

func TestGetNewsAdminSeesFuturePosts(t *testing.T) {
	setupControllersDB(t)
	// GetNews checks the DB user's Admin field, not the token's admin claim.
	admin := createTestUser(t)
	admin.Admin = true
	admin, err := database.UpdateUserInDB(admin)
	if err != nil {
		t.Fatalf("failed to promote user to admin: %v", err)
	}

	future := models.News{
		Title:   "Future post",
		Body:    "Body",
		Enabled: true,
		Date:    time.Now().Add(24 * time.Hour),
	}
	future.ID = uuid.New()
	if _, err := database.CreateNewsPostInDB(future); err != nil {
		t.Fatalf("failed to create future news post: %v", err)
	}

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/news", nil)
	ctx.Request.Header.Set("Authorization", authHeader(t, admin.ID, true))
	GetNews(ctx)

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	var body struct {
		News []struct {
			ID string `json:"ID"`
		} `json:"news"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	found := false
	for _, n := range body.News {
		if n.ID == future.ID.String() {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an admin to see a future-dated post, got %+v", body.News)
	}
}

func TestGetNewsExcludesFuturePostsForNonAdmin(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	future := models.News{
		Title:   "Future post",
		Body:    "Body",
		Enabled: true,
		Date:    time.Now().Add(24 * time.Hour),
	}
	future.ID = uuid.New()
	if _, err := database.CreateNewsPostInDB(future); err != nil {
		t.Fatalf("failed to create future news post: %v", err)
	}

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/news", nil)
	ctx.Request.Header.Set("Authorization", authHeader(t, user.ID, false))
	GetNews(ctx)

	var body struct {
		News []struct {
			ID string `json:"ID"`
		} `json:"news"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	for _, n := range body.News {
		if n.ID == future.ID.String() {
			t.Error("did not expect a non-admin to see a future-dated post")
		}
	}
}

func TestGetNewsExcludesExpiredEvenForAdmin(t *testing.T) {
	setupControllersDB(t)
	admin := createTestUser(t)

	past := time.Now().Add(-48 * time.Hour)
	expired := time.Now().Add(-24 * time.Hour)
	expiredPost := models.News{
		Title:      "Expired post",
		Body:       "Body",
		Enabled:    true,
		Date:       past,
		ExpiryDate: &expired,
	}
	expiredPost.ID = uuid.New()
	if _, err := database.CreateNewsPostInDB(expiredPost); err != nil {
		t.Fatalf("failed to create expired news post: %v", err)
	}

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/news", nil)
	ctx.Request.Header.Set("Authorization", authHeader(t, admin.ID, true))
	GetNews(ctx)

	var body struct {
		News []struct {
			ID string `json:"ID"`
		} `json:"news"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	for _, n := range body.News {
		if n.ID == expiredPost.ID.String() {
			t.Error("did not expect an admin to see an expired post")
		}
	}
}

func TestGetNewsPostInvalidID(t *testing.T) {
	setupControllersDB(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/news/not-a-uuid", nil)
	ctx.Params = gin.Params{{Key: "news_id", Value: "not-a-uuid"}}
	GetNewsPost(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed news_id", w.Code)
	}
}

func TestGetNewsPostSuccess(t *testing.T) {
	setupControllersDB(t)
	news := createTestNews(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/news/"+news.ID.String(), nil)
	ctx.Params = gin.Params{{Key: "news_id", Value: news.ID.String()}}
	GetNewsPost(ctx)

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
}

func postNewsPost(t *testing.T, authHeaderValue string, body string) (int, map[string]interface{}) {
	t.Helper()
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/api/news", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	if authHeaderValue != "" {
		ctx.Request.Header.Set("Authorization", authHeaderValue)
	}
	RegisterNewsPost(ctx)

	var parsed map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &parsed)
	return w.Code, parsed
}

func TestRegisterNewsPostRequiresAuth(t *testing.T) {
	setupControllersDB(t)

	code, _ := postNewsPost(t, "", `{"title":"Hello world","body":"Some news body"}`)
	if code != 400 {
		t.Errorf("status = %d, want 400 without auth", code)
	}
}

func TestRegisterNewsPostUserNotFound(t *testing.T) {
	setupControllersDB(t)

	code, _ := postNewsPost(t, authHeader(t, uuid.New(), true), `{"title":"Hello world","body":"Some news body"}`)
	if code != 500 {
		t.Errorf("status = %d, want 500 when the token's user doesn't exist", code)
	}
}

func TestRegisterNewsPostSuccess(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	code, body := postNewsPost(t, authHeader(t, user.ID, true), `{"title":"Hello world","body":"Some news body"}`)
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, body)
	}
	if body["message"] != "News post created." {
		t.Errorf("message = %v", body["message"])
	}
}

func TestRegisterNewsPostTitleTooShort(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	code, _ := postNewsPost(t, authHeader(t, user.ID, true), `{"title":"Hi","body":"Some news body"}`)
	if code != 400 {
		t.Errorf("status = %d, want 400 for a too-short title", code)
	}
}

func TestRegisterNewsPostBodyTooShort(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	code, _ := postNewsPost(t, authHeader(t, user.ID, true), `{"title":"Hello world","body":"Hi"}`)
	if code != 400 {
		t.Errorf("status = %d, want 400 for a too-short body", code)
	}
}

func TestRegisterNewsPostInvalidTitleCharacters(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	code, _ := postNewsPost(t, authHeader(t, user.ID, true), `{"title":"<script>bad</script>","body":"Some news body"}`)
	if code != 400 {
		t.Errorf("status = %d, want 400 for an invalid title", code)
	}
}

func TestRegisterNewsPostInvalidBodyCharacters(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	code, _ := postNewsPost(t, authHeader(t, user.ID, true), `{"title":"Hello world","body":"<script>bad</script>"}`)
	if code != 400 {
		t.Errorf("status = %d, want 400 for an invalid body", code)
	}
}

func TestRegisterNewsPostMalformedJSON(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	code, _ := postNewsPost(t, authHeader(t, user.ID, true), `not-json`)
	if code != 400 {
		t.Errorf("status = %d, want 400 for malformed JSON", code)
	}
}

func TestDeleteNewsPostInvalidID(t *testing.T) {
	setupControllersDB(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("DELETE", "/api/news/not-a-uuid", nil)
	ctx.Params = gin.Params{{Key: "news_id", Value: "not-a-uuid"}}
	DeleteNewsPost(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed news_id", w.Code)
	}
}

func TestDeleteNewsPostNotFound(t *testing.T) {
	setupControllersDB(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	missingID := "00000000-0000-0000-0000-000000000000"
	ctx.Request = httptest.NewRequest("DELETE", "/api/news/"+missingID, nil)
	ctx.Params = gin.Params{{Key: "news_id", Value: missingID}}
	DeleteNewsPost(ctx)

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 for a news post that doesn't exist", w.Code)
	}
}

func TestDeleteNewsPostSuccess(t *testing.T) {
	setupControllersDB(t)
	news := createTestNews(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("DELETE", "/api/news/"+news.ID.String(), nil)
	ctx.Params = gin.Params{{Key: "news_id", Value: news.ID.String()}}
	DeleteNewsPost(ctx)

	if w.Code != 201 {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
}

func putNewsPost(t *testing.T, newsID string, body string) (int, map[string]interface{}) {
	t.Helper()
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("PUT", "/api/news/"+newsID, strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "news_id", Value: newsID}}
	APIEditNewsPost(ctx)

	var parsed map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &parsed)
	return w.Code, parsed
}

func TestEditNewsPostInvalidID(t *testing.T) {
	setupControllersDB(t)

	code, _ := putNewsPost(t, "not-a-uuid", `{"title":"Hello world","body":"Some news body"}`)
	if code != 400 {
		t.Errorf("status = %d, want 400 for a malformed news_id", code)
	}
}

func TestEditNewsPostNotFound(t *testing.T) {
	setupControllersDB(t)

	missingID := "00000000-0000-0000-0000-000000000000"
	code, _ := putNewsPost(t, missingID, `{"title":"Hello world","body":"Some news body"}`)
	if code != 400 {
		t.Errorf("status = %d, want 400 for a news post that doesn't exist", code)
	}
}

func TestEditNewsPostMalformedJSON(t *testing.T) {
	setupControllersDB(t)
	news := createTestNews(t)

	code, _ := putNewsPost(t, news.ID.String(), `not-json`)
	if code != 400 {
		t.Errorf("status = %d, want 400 for malformed JSON", code)
	}
}

func TestEditNewsPostTitleTooShort(t *testing.T) {
	setupControllersDB(t)
	news := createTestNews(t)

	code, _ := putNewsPost(t, news.ID.String(), `{"title":"Hi","body":"Some news body"}`)
	if code != 400 {
		t.Errorf("status = %d, want 400 for a too-short title", code)
	}
}

func TestEditNewsPostBodyTooShort(t *testing.T) {
	setupControllersDB(t)
	news := createTestNews(t)

	code, _ := putNewsPost(t, news.ID.String(), `{"title":"Hello world","body":"Hi"}`)
	if code != 400 {
		t.Errorf("status = %d, want 400 for a too-short body", code)
	}
}

func TestEditNewsPostInvalidTitleCharacters(t *testing.T) {
	setupControllersDB(t)
	news := createTestNews(t)

	code, _ := putNewsPost(t, news.ID.String(), `{"title":"<script>bad</script>","body":"Some news body"}`)
	if code != 400 {
		t.Errorf("status = %d, want 400 for an invalid title", code)
	}
}

func TestEditNewsPostInvalidBodyCharacters(t *testing.T) {
	setupControllersDB(t)
	news := createTestNews(t)

	code, _ := putNewsPost(t, news.ID.String(), `{"title":"Hello world","body":"<script>bad</script>"}`)
	if code != 400 {
		t.Errorf("status = %d, want 400 for an invalid body", code)
	}
}

func TestEditNewsPostSuccess(t *testing.T) {
	setupControllersDB(t)
	news := createTestNews(t)
	expiry := time.Now().Add(24 * time.Hour)
	body, err := json.Marshal(map[string]interface{}{
		"title":       "Updated title",
		"body":        "Updated news body",
		"date":        time.Now(),
		"expiry_date": expiry,
	})
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	code, parsed := putNewsPost(t, news.ID.String(), string(body))
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, parsed)
	}
	if parsed["message"] != "News post updated." {
		t.Errorf("message = %v", parsed["message"])
	}
}

func TestNewsHandlersDatabaseErrors(t *testing.T) {
	newsParam := gin.Params{{Key: "news_id", Value: "00000000-0000-0000-0000-00000000000a"}}
	runDatabaseErrorCases(t, []dbErrorCase{
		{name: "GetNews", handler: GetNews, method: "GET", path: "/api/auth/news"},
		{name: "GetNewsPost", handler: GetNewsPost, method: "GET", path: "/api/auth/news/00000000-0000-0000-0000-00000000000a", params: newsParam},
		{name: "RegisterNewsPost", handler: RegisterNewsPost, method: "POST", path: "/api/admin/news", body: `{"title":"Title","body":"Body"}`, admin: true},
		{name: "DeleteNewsPost", handler: DeleteNewsPost, method: "DELETE", path: "/api/admin/news/00000000-0000-0000-0000-00000000000a", params: newsParam, admin: true},
		{name: "APIEditNewsPost", handler: APIEditNewsPost, method: "POST", path: "/api/admin/news/00000000-0000-0000-0000-00000000000a", body: `{"title":"Title","body":"Body"}`, params: newsParam, admin: true},
	})
}
