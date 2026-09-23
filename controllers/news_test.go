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

// newsCreatePost inserts an enabled news post with the given dates.
func newsCreatePost(t *testing.T, title string, date time.Time, expiry *time.Time) models.News {
	t.Helper()
	news := models.News{Title: title, Body: "Body text", Enabled: true, Date: date, ExpiryDate: expiry}
	news.ID = uuid.New()
	created, err := database.CreateNewsPostInDB(news)
	if err != nil {
		t.Fatalf("failed to create news post: %v", err)
	}
	return created
}

// newsResponseIDs pulls the IDs out of a handler's "news" array, in order.
func newsResponseIDs(t *testing.T, body map[string]interface{}) []string {
	t.Helper()
	raw, ok := body["news"].([]interface{})
	if !ok {
		t.Fatalf("response has no news array: %v", body)
	}
	ids := make([]string, 0, len(raw))
	for _, item := range raw {
		post, _ := item.(map[string]interface{})
		id, _ := post["id"].(string)
		ids = append(ids, id)
	}
	return ids
}

func TestGetNewsSortsNewestFirst(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	older := newsCreatePost(t, "Older post", time.Now().Add(-48*time.Hour), nil)
	newer := newsCreatePost(t, "Newer post", time.Now().Add(-1*time.Hour), nil)
	oldest := newsCreatePost(t, "Oldest post", time.Now().Add(-96*time.Hour), nil)

	status, body, _ := doRequest(GetNews, "GET", "/api/auth/news", "", map[string]string{"Authorization": authHeader(t, user.ID, false)}, nil)
	if status != 201 {
		t.Fatalf("status = %d, want 201; body=%v", status, body)
	}
	got := newsResponseIDs(t, body)
	want := []string{newer.ID.String(), older.ID.String(), oldest.ID.String()}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("news order = %v, want %v", got, want)
	}
}

func TestGetNewsListFails(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	header := map[string]string{"Authorization": authHeader(t, user.ID, false)}
	failDBOperation(t, "query", "news", 0)

	status, body, _ := doRequest(GetNews, "GET", "/api/auth/news", "", header, nil)
	if status != 500 || body["error"] != "Failed to get news." {
		t.Errorf("status = %d body=%v, want 500 'Failed to get news.'", status, body)
	}
}

func TestRegisterNewsPostAdminResponseSkipsExpired(t *testing.T) {
	setupControllersDB(t)
	admin := createTestUser(t)
	admin.Admin = true
	admin, err := database.UpdateUserInDB(admin)
	if err != nil {
		t.Fatalf("failed to promote user to admin: %v", err)
	}
	expiredAt := time.Now().Add(-time.Hour)
	expired := newsCreatePost(t, "Expired post", time.Now().Add(-48*time.Hour), &expiredAt)
	live := newsCreatePost(t, "Live post", time.Now().Add(-24*time.Hour), nil)

	status, body := postNewsPost(t, authHeader(t, admin.ID, true), `{"title":"Hello world","body":"Some news body"}`)
	if status != 201 {
		t.Fatalf("status = %d, want 201; body=%v", status, body)
	}
	ids := strings.Join(newsResponseIDs(t, body), ",")
	if strings.Contains(ids, expired.ID.String()) {
		t.Error("expired post must not be in the response")
	}
	if !strings.Contains(ids, live.ID.String()) {
		t.Error("live post should be in an admin's response")
	}
	if len(newsResponseIDs(t, body)) != 2 {
		t.Errorf("response news = %v, want the live post plus the new one", ids)
	}
}

func TestRegisterNewsPostDatabaseFailures(t *testing.T) {
	cases := []struct {
		name      string
		op        string
		wantError string
	}{
		{"create", "create", "Failed to create news post."},
		{"reload list", "query", "Failed to get news posts."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			user := createTestUser(t)
			header := authHeader(t, user.ID, true)
			failDBOperation(t, c.op, "news", 0)

			status, body := postNewsPost(t, header, `{"title":"Hello world","body":"Some news body"}`)
			if status != 500 || body["error"] != c.wantError {
				t.Errorf("status = %d body=%v, want 500 %q", status, body, c.wantError)
			}
		})
	}
}

func TestDeleteNewsPostDatabaseFailures(t *testing.T) {
	cases := []struct {
		name      string
		op        string
		skip      int
		wantError string
	}{
		{"disable", "update", 0, "Failed to delete news post."},
		{"reload list", "query", 1, "Failed to get news posts."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			news := createTestNews(t)
			failDBOperation(t, c.op, "news", c.skip)

			status, body, _ := doRequest(DeleteNewsPost, "DELETE", "/api/admin/news/"+news.ID.String(), "", nil, gin.Params{{Key: "news_id", Value: news.ID.String()}})
			if status != 500 || body["error"] != c.wantError {
				t.Errorf("status = %d body=%v, want 500 %q", status, body, c.wantError)
			}
		})
	}
}

func TestEditNewsPostSaveFails(t *testing.T) {
	setupControllersDB(t)
	news := createTestNews(t)
	failDBOperation(t, "update", "news", 0)

	status, body := putNewsPost(t, news.ID.String(), `{"title":"Updated title","body":"Updated news body"}`)
	if status != 500 || body["error"] != "Failed to create news post." {
		t.Errorf("status = %d body=%v, want 500", status, body)
	}

	stored, err := database.GetNewsPostByNewsID(news.ID)
	if err != nil {
		t.Fatalf("failed to reload news post: %v", err)
	}
	if stored.Title != news.Title {
		t.Errorf("title = %q, want unchanged %q", stored.Title, news.Title)
	}
}

func TestVisibleNewsPosts(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	older := now.Add(-2 * time.Hour)
	future := now.Add(time.Hour)

	post := func(title string, date time.Time, expiry *time.Time) models.News {
		return models.News{Title: title, Date: date, ExpiryDate: expiry}
	}
	posts := []models.News{
		post("older", older, nil),
		post("scheduled", future, nil),
		post("expired", older, &past),
		post("live", past, &future),
	}

	cases := []struct {
		name  string
		admin bool
		want  []string
	}{
		{"non-admin sees only published, unexpired posts", false, []string{"live", "older"}},
		{"admin also sees scheduled posts", true, []string{"scheduled", "live", "older"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := visibleNewsPosts(posts, c.admin, now)
			titles := []string{}
			for _, p := range got {
				titles = append(titles, p.Title)
			}
			if strings.Join(titles, ",") != strings.Join(c.want, ",") {
				t.Errorf("titles = %v, want %v (newest first)", titles, c.want)
			}
		})
	}
}
