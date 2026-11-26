package mcpserver

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const writeScope = "mcp:wishlists.read mcp:wishlists.write"

func createMCPTestWishlist(t *testing.T, ownerID uuid.UUID) models.Wishlist {
	t.Helper()
	now := time.Now()
	wishlist := models.Wishlist{Name: "Birthday", Enabled: true, OwnerID: ownerID, Date: &now}
	wishlist.ID = uuid.New()
	if r := database.Instance.Create(&wishlist); r.Error != nil {
		t.Fatalf("failed to create wishlist: %v", r.Error)
	}
	return wishlist
}

func createMCPTestWish(t *testing.T, ownerID uuid.UUID, wishlistID uuid.UUID, name string) models.Wish {
	t.Helper()
	price := 10.0
	wish := models.Wish{Name: name, Note: "Blue", URL: "https://example.com/item", Price: &price, Enabled: true, OwnerID: ownerID, WishlistID: wishlistID}
	wish.ID = uuid.New()
	if r := database.Instance.Create(&wish); r.Error != nil {
		t.Fatalf("failed to create wish: %v", r.Error)
	}
	return wish
}

func addMCPTestCollaborator(t *testing.T, wishlistID uuid.UUID, userID uuid.UUID) {
	t.Helper()
	collaborator := models.WishlistCollaborator{WishlistID: wishlistID, UserID: userID, Enabled: true}
	collaborator.ID = uuid.New()
	if r := database.Instance.Create(&collaborator); r.Error != nil {
		t.Fatalf("failed to create collaborator: %v", r.Error)
	}
}

// loadMCPTestWish reads a wish straight from the database, deleted or not.
func loadMCPTestWish(t *testing.T, wishID uuid.UUID) models.Wish {
	t.Helper()
	var wish models.Wish
	if r := database.Instance.Where("wishes.id = ?", wishID).First(&wish); r.Error != nil {
		t.Fatalf("failed to load wish: %v", r.Error)
	}
	return wish
}

func countMCPTestWishes(t *testing.T, wishlistID uuid.UUID) int64 {
	t.Helper()
	var count int64
	if r := database.Instance.Model(&models.Wish{}).Where("wishes.wishlist_id = ? AND wishes.enabled = ?", wishlistID, true).Count(&count); r.Error != nil {
		t.Fatalf("failed to count wishes: %v", r.Error)
	}
	return count
}

func callMCPTool(t *testing.T, session *sdkmcp.ClientSession, name string, args map[string]any) *sdkmcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool %s transport-level error: %v", name, err)
	}
	return result
}

func decodeWishOut(t *testing.T, result *sdkmcp.CallToolResult) wishOut {
	t.Helper()
	if result.IsError {
		t.Fatalf("tool call reported an error: %s", firstText(t, result))
	}
	var got wishOut
	if err := json.Unmarshal([]byte(firstText(t, result)), &got); err != nil {
		t.Fatalf("failed to parse tool result: %v (raw: %s)", err, firstText(t, result))
	}
	return got
}

func TestAddWishTool(t *testing.T) {
	setupMCPDB(t)
	owner := createMCPTestUser(t, "Ada")
	wishlist := createMCPTestWishlist(t, owner.ID)
	session := startMCPTestServer(t, owner.ID, writeScope)

	got := decodeWishOut(t, callMCPTool(t, session, "add_wish", map[string]any{
		"wishlist_id": wishlist.ID.String(),
		"name":        "  Mouse Logitech MX Master 4  ",
		"note":        "Nero",
		"url":         "https://www.amazon.it/dp/B0FHHSZ8WM",
		"price":       99.9,
	}))

	if got.Name != "Mouse Logitech MX Master 4" || got.Note != "Nero" || got.URL != "https://www.amazon.it/dp/B0FHHSZ8WM" || got.Price == nil || *got.Price != 99.9 {
		t.Errorf("returned wish = %+v, want the trimmed name, the note, the link and the price", got)
	}

	stored := loadMCPTestWish(t, uuid.MustParse(got.ID))
	if !stored.Enabled || stored.OwnerID != owner.ID || stored.WishlistID != wishlist.ID {
		t.Errorf("stored wish = %+v, want an enabled wish owned by the caller on the wishlist", stored)
	}
}

func TestAddWishToolOnlyNeedsAName(t *testing.T) {
	setupMCPDB(t)
	owner := createMCPTestUser(t, "Ada")
	wishlist := createMCPTestWishlist(t, owner.ID)
	session := startMCPTestServer(t, owner.ID, writeScope)

	got := decodeWishOut(t, callMCPTool(t, session, "add_wish", map[string]any{
		"wishlist_id": wishlist.ID.String(),
		"name":        "Tappetino bagno",
	}))
	if got.Note != "" || got.URL != "" || got.Price != nil {
		t.Errorf("returned wish = %+v, want no note, link or price", got)
	}
}

func TestAddWishToolAsCollaborator(t *testing.T) {
	setupMCPDB(t)
	owner := createMCPTestUser(t, "Ada")
	collaborator := createMCPTestUser(t, "Bob")
	wishlist := createMCPTestWishlist(t, owner.ID)
	addMCPTestCollaborator(t, wishlist.ID, collaborator.ID)
	session := startMCPTestServer(t, collaborator.ID, writeScope)

	got := decodeWishOut(t, callMCPTool(t, session, "add_wish", map[string]any{
		"wishlist_id": wishlist.ID.String(),
		"name":        "Headphones",
	}))
	if stored := loadMCPTestWish(t, uuid.MustParse(got.ID)); stored.OwnerID != collaborator.ID {
		t.Errorf("wish owner = %s, want the collaborator who added it", stored.OwnerID)
	}
}

// Every refused add leaves the wishlist as it was.
func TestAddWishToolRefuses(t *testing.T) {
	cases := []struct {
		name    string
		scope   string
		asOther bool
		args    func(wishlist models.Wishlist) map[string]any
		want    string
	}{
		{
			name:  "token without the write scope",
			scope: "mcp:wishlists.read",
			args: func(w models.Wishlist) map[string]any {
				return map[string]any{"wishlist_id": w.ID.String(), "name": "Headphones"}
			},
			want: "missing required scope: mcp:wishlists.write",
		},
		{
			name:    "someone else's wishlist",
			asOther: true,
			args: func(w models.Wishlist) map[string]any {
				return map[string]any{"wishlist_id": w.ID.String(), "name": "Headphones"}
			},
			want: "wishlist not found or not accessible",
		},
		{
			name: "a wishlist that doesn't exist",
			args: func(models.Wishlist) map[string]any {
				return map[string]any{"wishlist_id": uuid.NewString(), "name": "Headphones"}
			},
			want: "wishlist not found or not accessible",
		},
		{
			name: "malformed wishlist_id",
			args: func(models.Wishlist) map[string]any {
				return map[string]any{"wishlist_id": "not-a-uuid", "name": "Headphones"}
			},
			want: "invalid wishlist_id",
		},
		{
			name: "nil wishlist_id",
			args: func(models.Wishlist) map[string]any {
				return map[string]any{"wishlist_id": uuid.Nil.String(), "name": "Headphones"}
			},
			want: "invalid wishlist_id",
		},
		{
			name: "name shorter than five characters",
			args: func(w models.Wishlist) map[string]any {
				return map[string]any{"wishlist_id": w.ID.String(), "name": " Hat "}
			},
			want: "the name of the wish must be five or more characters",
		},
		{
			name: "name already on the wishlist",
			args: func(w models.Wishlist) map[string]any {
				return map[string]any{"wishlist_id": w.ID.String(), "name": "Existing wish"}
			},
			want: "there is already a wish with that name on this wishlist",
		},
		{
			name: "name already on the wishlist in other capitals",
			args: func(w models.Wishlist) map[string]any {
				return map[string]any{"wishlist_id": w.ID.String(), "name": "existing WISH"}
			},
			want: "there is already a wish with that name on this wishlist",
		},
		{
			name: "name of two characters that take more than five bytes",
			args: func(w models.Wishlist) map[string]any {
				return map[string]any{"wishlist_id": w.ID.String(), "name": "🎁🎁"}
			},
			want: "the name of the wish must be five or more characters",
		},
		{
			name: "negative price",
			args: func(w models.Wishlist) map[string]any {
				return map[string]any{"wishlist_id": w.ID.String(), "name": "Headphones", "price": -50}
			},
			want: "invalid price: it can't be negative",
		},
		{
			name: "markup in the name",
			args: func(w models.Wishlist) map[string]any {
				return map[string]any{"wishlist_id": w.ID.String(), "name": "<b>Headphones</b>"}
			},
			want: `invalid name: Text must not contain <, >, or ".`,
		},
		{
			name: "markup in the note",
			args: func(w models.Wishlist) map[string]any {
				return map[string]any{"wishlist_id": w.ID.String(), "name": "Headphones", "note": `9" screen`}
			},
			want: `invalid note: Text must not contain <, >, or ".`,
		},
		{
			name: "javascript link",
			args: func(w models.Wishlist) map[string]any {
				return map[string]any{"wishlist_id": w.ID.String(), "name": "Headphones", "url": "javascript:alert(1)"}
			},
			want: "invalid url: it must be an absolute http or https link",
		},
		{
			name: "link without a scheme",
			args: func(w models.Wishlist) map[string]any {
				return map[string]any{"wishlist_id": w.ID.String(), "name": "Headphones", "url": "www.amazon.it/dp/B0FHHSZ8WM"}
			},
			want: "invalid url: it must be an absolute http or https link",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupMCPDB(t)
			owner := createMCPTestUser(t, "Ada")
			other := createMCPTestUser(t, "Bob")
			wishlist := createMCPTestWishlist(t, owner.ID)
			createMCPTestWish(t, owner.ID, wishlist.ID, "Existing wish")

			caller, scope := owner.ID, writeScope
			if c.asOther {
				caller = other.ID
			}
			if c.scope != "" {
				scope = c.scope
			}
			session := startMCPTestServer(t, caller, scope)

			result := callMCPTool(t, session, "add_wish", c.args(wishlist))
			if !result.IsError || firstText(t, result) != c.want {
				t.Errorf("result = (error %v) %q, want error %q", result.IsError, firstText(t, result), c.want)
			}
			if got := countMCPTestWishes(t, wishlist.ID); got != 1 {
				t.Errorf("wishlist has %d wishes, want only the existing one", got)
			}
		})
	}
}

func TestUpdateWishTool(t *testing.T) {
	setupMCPDB(t)
	owner := createMCPTestUser(t, "Ada")
	wishlist := createMCPTestWishlist(t, owner.ID)
	wish := createMCPTestWish(t, owner.ID, wishlist.ID, "Headphones")
	session := startMCPTestServer(t, owner.ID, writeScope)

	got := decodeWishOut(t, callMCPTool(t, session, "update_wish", map[string]any{
		"wish_id": wish.ID.String(),
		"name":    " Cuffie wireless ",
		"note":    "",
		"price":   24.99,
	}))

	if got.Name != "Cuffie wireless" || got.Note != "" || got.URL != wish.URL || got.Price == nil || *got.Price != 24.99 {
		t.Errorf("returned wish = %+v, want the new name and price, no note, and the link unchanged", got)
	}
	stored := loadMCPTestWish(t, wish.ID)
	if stored.Name != "Cuffie wireless" || stored.Note != "" || stored.URL != wish.URL || stored.Price == nil || *stored.Price != 24.99 || !stored.Enabled {
		t.Errorf("stored wish = %+v, want the update saved and the rest unchanged", stored)
	}
}

// Leaving every field out changes nothing, and keeping the current name doesn't
// trip the duplicate-name check against the wish itself.
func TestUpdateWishToolKeepsWhatIsLeftOut(t *testing.T) {
	setupMCPDB(t)
	owner := createMCPTestUser(t, "Ada")
	wishlist := createMCPTestWishlist(t, owner.ID)
	wish := createMCPTestWish(t, owner.ID, wishlist.ID, "Headphones")
	session := startMCPTestServer(t, owner.ID, writeScope)

	got := decodeWishOut(t, callMCPTool(t, session, "update_wish", map[string]any{
		"wish_id": wish.ID.String(),
		"name":    "Headphones",
	}))
	if got.Name != wish.Name || got.Note != wish.Note || got.URL != wish.URL || got.Price == nil || *got.Price != *wish.Price {
		t.Errorf("returned wish = %+v, want it unchanged from %+v", got, wish)
	}
}

func TestUpdateWishToolClearsPriceAndLink(t *testing.T) {
	setupMCPDB(t)
	owner := createMCPTestUser(t, "Ada")
	wishlist := createMCPTestWishlist(t, owner.ID)
	wish := createMCPTestWish(t, owner.ID, wishlist.ID, "Headphones")
	session := startMCPTestServer(t, owner.ID, writeScope)

	got := decodeWishOut(t, callMCPTool(t, session, "update_wish", map[string]any{
		"wish_id":     wish.ID.String(),
		"url":         "",
		"clear_price": true,
	}))
	if got.URL != "" || got.Price != nil {
		t.Errorf("returned wish = %+v, want no link and no price", got)
	}
	if stored := loadMCPTestWish(t, wish.ID); stored.URL != "" || stored.Price != nil {
		t.Errorf("stored wish = %+v, want no link and no price", stored)
	}
}

func TestUpdateWishToolAsCollaborator(t *testing.T) {
	setupMCPDB(t)
	owner := createMCPTestUser(t, "Ada")
	collaborator := createMCPTestUser(t, "Bob")
	wishlist := createMCPTestWishlist(t, owner.ID)
	addMCPTestCollaborator(t, wishlist.ID, collaborator.ID)
	wish := createMCPTestWish(t, owner.ID, wishlist.ID, "Headphones")
	session := startMCPTestServer(t, collaborator.ID, writeScope)

	got := decodeWishOut(t, callMCPTool(t, session, "update_wish", map[string]any{
		"wish_id": wish.ID.String(),
		"note":    "Black",
	}))
	if got.Note != "Black" {
		t.Errorf("note = %q, want Black", got.Note)
	}
}

// Every refused update leaves the wish as it was.
func TestUpdateWishToolRefuses(t *testing.T) {
	cases := []struct {
		name    string
		scope   string
		asOther bool
		deleted bool
		args    func(wish models.Wish) map[string]any
		want    string
	}{
		{
			name:  "token without the write scope",
			scope: "mcp:wishlists.read",
			args:  func(w models.Wish) map[string]any { return map[string]any{"wish_id": w.ID.String(), "note": "Black"} },
			want:  "missing required scope: mcp:wishlists.write",
		},
		{
			name:    "a wish on someone else's wishlist",
			asOther: true,
			args:    func(w models.Wish) map[string]any { return map[string]any{"wish_id": w.ID.String(), "note": "Black"} },
			want:    "wish not found or not accessible",
		},
		{
			name:    "a deleted wish",
			deleted: true,
			args:    func(w models.Wish) map[string]any { return map[string]any{"wish_id": w.ID.String(), "note": "Black"} },
			want:    "wish not found or not accessible",
		},
		{
			name: "a wish that doesn't exist",
			args: func(models.Wish) map[string]any { return map[string]any{"wish_id": uuid.NewString(), "note": "Black"} },
			want: "wish not found or not accessible",
		},
		{
			name: "malformed wish_id",
			args: func(models.Wish) map[string]any { return map[string]any{"wish_id": "not-a-uuid", "note": "Black"} },
			want: "invalid wish_id",
		},
		{
			name: "price and clear_price together",
			args: func(w models.Wish) map[string]any {
				return map[string]any{"wish_id": w.ID.String(), "price": 5, "clear_price": true}
			},
			want: "give either price or clear_price, not both",
		},
		{
			name: "name shorter than five characters",
			args: func(w models.Wish) map[string]any { return map[string]any{"wish_id": w.ID.String(), "name": "Hat"} },
			want: "the name of the wish must be five or more characters",
		},
		{
			name: "name of another wish on the wishlist",
			args: func(w models.Wish) map[string]any {
				return map[string]any{"wish_id": w.ID.String(), "name": "Other wish"}
			},
			want: "there is already a wish with that name on this wishlist",
		},
		{
			name: "name of another wish in other capitals",
			args: func(w models.Wish) map[string]any {
				return map[string]any{"wish_id": w.ID.String(), "name": "OTHER wish"}
			},
			want: "there is already a wish with that name on this wishlist",
		},
		{
			name: "negative price",
			args: func(w models.Wish) map[string]any { return map[string]any{"wish_id": w.ID.String(), "price": -0.01} },
			want: "invalid price: it can't be negative",
		},
		{
			name: "markup in the note",
			args: func(w models.Wish) map[string]any {
				return map[string]any{"wish_id": w.ID.String(), "note": "<i>Black</i>"}
			},
			want: "invalid note: Text must not contain <, >, or \".",
		},
		{
			name: "javascript link",
			args: func(w models.Wish) map[string]any {
				return map[string]any{"wish_id": w.ID.String(), "url": "javascript:alert(1)"}
			},
			want: "invalid url: it must be an absolute http or https link",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupMCPDB(t)
			owner := createMCPTestUser(t, "Ada")
			other := createMCPTestUser(t, "Bob")
			wishlist := createMCPTestWishlist(t, owner.ID)
			wish := createMCPTestWish(t, owner.ID, wishlist.ID, "Headphones")
			createMCPTestWish(t, owner.ID, wishlist.ID, "Other wish")
			if c.deleted {
				if r := database.Instance.Model(&models.Wish{}).Where("wishes.id = ?", wish.ID).Update("enabled", false); r.Error != nil {
					t.Fatalf("failed to delete wish: %v", r.Error)
				}
			}

			caller, scope := owner.ID, writeScope
			if c.asOther {
				caller = other.ID
			}
			if c.scope != "" {
				scope = c.scope
			}
			session := startMCPTestServer(t, caller, scope)

			result := callMCPTool(t, session, "update_wish", c.args(wish))
			if !result.IsError || firstText(t, result) != c.want {
				t.Errorf("result = (error %v) %q, want error %q", result.IsError, firstText(t, result), c.want)
			}
			stored := loadMCPTestWish(t, wish.ID)
			if stored.Name != wish.Name || stored.Note != wish.Note || stored.URL != wish.URL || stored.Price == nil || *stored.Price != *wish.Price {
				t.Errorf("stored wish = %+v, want it unchanged", stored)
			}
		})
	}
}

// A name's length counts characters, so five letters outside ASCII are
// enough, and a free wish is not a negative price.
func TestAddWishToolCountsCharactersAndAllowsAFreeWish(t *testing.T) {
	setupMCPDB(t)
	owner := createMCPTestUser(t, "Ada")
	wishlist := createMCPTestWishlist(t, owner.ID)
	session := startMCPTestServer(t, owner.ID, writeScope)

	got := decodeWishOut(t, callMCPTool(t, session, "add_wish", map[string]any{
		"wishlist_id": wishlist.ID.String(),
		"name":        "Ærøgå",
		"price":       0,
	}))
	if got.Name != "Ærøgå" || got.Price == nil || *got.Price != 0 {
		t.Errorf("returned wish = %+v, want the name and a price of 0", got)
	}
}

// Names are unique ignoring case, but that doesn't stop a wish from being
// renamed to a different capitalisation of its own name.
func TestUpdateWishToolChangesTheCaseOfItsName(t *testing.T) {
	setupMCPDB(t)
	owner := createMCPTestUser(t, "Ada")
	wishlist := createMCPTestWishlist(t, owner.ID)
	wish := createMCPTestWish(t, owner.ID, wishlist.ID, "Headphones")
	session := startMCPTestServer(t, owner.ID, writeScope)

	got := decodeWishOut(t, callMCPTool(t, session, "update_wish", map[string]any{"wish_id": wish.ID.String(), "name": "HEADPHONES"}))
	if got.Name != "HEADPHONES" || loadMCPTestWish(t, wish.ID).Name != "HEADPHONES" {
		t.Errorf("returned wish = %+v, want it renamed to HEADPHONES", got)
	}
}

// Deleting a wishlist leaves its collaborators and wishes enabled, so every
// write tool must still refuse it, for its collaborators as well as its owner.
func TestWriteToolsRefuseADeletedWishlist(t *testing.T) {
	for _, role := range []string{"owner", "collaborator"} {
		calls := []struct {
			tool string
			args func(wishlist models.Wishlist, wish models.Wish) map[string]any
			want string
		}{
			{"add_wish", func(wl models.Wishlist, _ models.Wish) map[string]any {
				return map[string]any{"wishlist_id": wl.ID.String(), "name": "Headphones"}
			}, "wishlist not found or not accessible"},
			{"update_wish", func(_ models.Wishlist, w models.Wish) map[string]any {
				return map[string]any{"wish_id": w.ID.String(), "note": "Black"}
			}, "wish not found or not accessible"},
			{"delete_wish", func(_ models.Wishlist, w models.Wish) map[string]any {
				return map[string]any{"wish_id": w.ID.String()}
			}, "wish not found or not accessible"},
		}
		for _, call := range calls {
			t.Run(role+" "+call.tool, func(t *testing.T) {
				setupMCPDB(t)
				owner := createMCPTestUser(t, "Ada")
				collaborator := createMCPTestUser(t, "Bob")
				wishlist := createMCPTestWishlist(t, owner.ID)
				addMCPTestCollaborator(t, wishlist.ID, collaborator.ID)
				wish := createMCPTestWish(t, owner.ID, wishlist.ID, "Existing wish")
				if r := database.Instance.Model(&models.Wishlist{}).Where("wishlists.id = ?", wishlist.ID).Update("enabled", false); r.Error != nil {
					t.Fatalf("failed to delete wishlist: %v", r.Error)
				}
				notified := replaceWishDeletedNotifier(t)

				caller := owner.ID
				if role == "collaborator" {
					caller = collaborator.ID
				}
				session := startMCPTestServer(t, caller, writeScope)

				result := callMCPTool(t, session, call.tool, call.args(wishlist, wish))
				if !result.IsError || firstText(t, result) != call.want {
					t.Errorf("result = (error %v) %q, want error %q", result.IsError, firstText(t, result), call.want)
				}
				stored := loadMCPTestWish(t, wish.ID)
				if !stored.Enabled || stored.Note != wish.Note || countMCPTestWishes(t, wishlist.ID) != 1 {
					t.Errorf("stored wish = %+v and %d wishes, want the wishlist unchanged", stored, countMCPTestWishes(t, wishlist.ID))
				}
				if len(*notified) != 0 {
					t.Errorf("claimers notified for %v", *notified)
				}
			})
		}
	}
}

func replaceWishDeletedNotifier(t *testing.T) *[]uuid.UUID {
	t.Helper()
	notified := []uuid.UUID{}
	original := wishDeletedNotifier
	t.Cleanup(func() { wishDeletedNotifier = original })
	wishDeletedNotifier = func(wish models.Wish) { notified = append(notified, wish.ID) }
	return &notified
}

func TestDeleteWishTool(t *testing.T) {
	setupMCPDB(t)
	owner := createMCPTestUser(t, "Ada")
	wishlist := createMCPTestWishlist(t, owner.ID)
	wish := createMCPTestWish(t, owner.ID, wishlist.ID, "Headphones")
	notified := replaceWishDeletedNotifier(t)
	session := startMCPTestServer(t, owner.ID, writeScope)

	result := callMCPTool(t, session, "delete_wish", map[string]any{"wish_id": wish.ID.String()})
	if result.IsError {
		t.Fatalf("tool call reported an error: %s", firstText(t, result))
	}
	var got deletedWishOut
	if err := json.Unmarshal([]byte(firstText(t, result)), &got); err != nil || got.ID != wish.ID.String() || !got.Deleted {
		t.Errorf("result = %s (%v), want the deleted wish's ID", firstText(t, result), err)
	}

	if loadMCPTestWish(t, wish.ID).Enabled {
		t.Error("the wish is still enabled after delete_wish")
	}
	if len(*notified) != 1 || (*notified)[0] != wish.ID {
		t.Errorf("claimers notified for %v, want once for the deleted wish", *notified)
	}

	// A second delete finds nothing, so claimers aren't told twice.
	again := callMCPTool(t, session, "delete_wish", map[string]any{"wish_id": wish.ID.String()})
	if !again.IsError || firstText(t, again) != "wish not found or not accessible" {
		t.Errorf("second delete = (error %v) %q, want 'wish not found or not accessible'", again.IsError, firstText(t, again))
	}
	if len(*notified) != 1 {
		t.Errorf("claimers notified %d times, want once", len(*notified))
	}
}

func TestDeleteWishToolRemovesItsEmptyCategory(t *testing.T) {
	setupMCPDB(t)
	owner := createMCPTestUser(t, "Ada")
	wishlist := createMCPTestWishlist(t, owner.ID)
	replaceWishDeletedNotifier(t)
	category := models.WishCategory{Name: "Kitchen", WishlistID: wishlist.ID, OwnerID: owner.ID, Enabled: true}
	category.ID = uuid.New()
	if r := database.Instance.Create(&category); r.Error != nil {
		t.Fatalf("failed to create category: %v", r.Error)
	}
	wish := createMCPTestWish(t, owner.ID, wishlist.ID, "Coffee pods")
	if r := database.Instance.Model(&models.Wish{}).Where("wishes.id = ?", wish.ID).Update("category_id", category.ID); r.Error != nil {
		t.Fatalf("failed to put the wish in the category: %v", r.Error)
	}
	session := startMCPTestServer(t, owner.ID, writeScope)

	if result := callMCPTool(t, session, "delete_wish", map[string]any{"wish_id": wish.ID.String()}); result.IsError {
		t.Fatalf("tool call reported an error: %s", firstText(t, result))
	}

	var stored models.WishCategory
	if r := database.Instance.Where("wish_categories.id = ?", category.ID).First(&stored); r.Error != nil {
		t.Fatalf("failed to load category: %v", r.Error)
	}
	if stored.Enabled {
		t.Error("the category is still enabled after its last wish was deleted")
	}
}

func TestDeleteWishToolAsCollaborator(t *testing.T) {
	setupMCPDB(t)
	owner := createMCPTestUser(t, "Ada")
	collaborator := createMCPTestUser(t, "Bob")
	wishlist := createMCPTestWishlist(t, owner.ID)
	addMCPTestCollaborator(t, wishlist.ID, collaborator.ID)
	wish := createMCPTestWish(t, owner.ID, wishlist.ID, "Headphones")
	replaceWishDeletedNotifier(t)
	session := startMCPTestServer(t, collaborator.ID, writeScope)

	if result := callMCPTool(t, session, "delete_wish", map[string]any{"wish_id": wish.ID.String()}); result.IsError {
		t.Fatalf("tool call reported an error: %s", firstText(t, result))
	}
	if loadMCPTestWish(t, wish.ID).Enabled {
		t.Error("the wish is still enabled after delete_wish")
	}
}

func TestDeleteWishToolRefuses(t *testing.T) {
	cases := []struct {
		name    string
		scope   string
		asOther bool
		wishID  func(wish models.Wish) string
		want    string
	}{
		{name: "token without the write scope", scope: "mcp:wishlists.read", wishID: func(w models.Wish) string { return w.ID.String() }, want: "missing required scope: mcp:wishlists.write"},
		{name: "a wish on someone else's wishlist", asOther: true, wishID: func(w models.Wish) string { return w.ID.String() }, want: "wish not found or not accessible"},
		{name: "malformed wish_id", wishID: func(models.Wish) string { return "not-a-uuid" }, want: "invalid wish_id"},
		{name: "nil wish_id", wishID: func(models.Wish) string { return uuid.Nil.String() }, want: "invalid wish_id"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupMCPDB(t)
			owner := createMCPTestUser(t, "Ada")
			other := createMCPTestUser(t, "Bob")
			wishlist := createMCPTestWishlist(t, owner.ID)
			wish := createMCPTestWish(t, owner.ID, wishlist.ID, "Headphones")
			notified := replaceWishDeletedNotifier(t)

			caller, scope := owner.ID, writeScope
			if c.asOther {
				caller = other.ID
			}
			if c.scope != "" {
				scope = c.scope
			}
			session := startMCPTestServer(t, caller, scope)

			result := callMCPTool(t, session, "delete_wish", map[string]any{"wish_id": c.wishID(wish)})
			if !result.IsError || firstText(t, result) != c.want {
				t.Errorf("result = (error %v) %q, want error %q", result.IsError, firstText(t, result), c.want)
			}
			if !loadMCPTestWish(t, wish.ID).Enabled {
				t.Error("a refused delete_wish deleted the wish")
			}
			if len(*notified) != 0 {
				t.Errorf("claimers notified for %v after a refused delete", *notified)
			}
		})
	}
}

// TestWriteToolsSurfaceDatabaseFailures breaks the query each step needs and
// checks the tools answer with their generic error and change nothing.
func TestWriteToolsSurfaceDatabaseFailures(t *testing.T) {
	setup := func(t *testing.T) (models.User, models.Wishlist, models.Wish, context.Context) {
		setupMCPDB(t)
		owner := createMCPTestUser(t, "Ada")
		wishlist := createMCPTestWishlist(t, owner.ID)
		wish := createMCPTestWish(t, owner.ID, wishlist.ID, "Headphones")
		ctx := contextWithTokenInfo(t, &mcpauth.TokenInfo{UserID: owner.ID.String(), Scopes: []string{scopeWishlistsWrite}, Expiration: time.Now().Add(time.Hour)})
		return owner, wishlist, wish, ctx
	}
	dropTable := func(t *testing.T, model any) {
		if err := database.Instance.Migrator().DropTable(model); err != nil {
			t.Fatalf("failed to drop table: %v", err)
		}
	}
	wantError := func(t *testing.T, res *sdkmcp.CallToolResult, err error, want string) {
		t.Helper()
		if err == nil || err.Error() != want || res != nil {
			t.Errorf("result = (%v, %v), want %q", res, err, want)
		}
	}

	t.Run("add_wish ownership check", func(t *testing.T) {
		_, wishlist, _, ctx := setup(t)
		dropTable(t, &models.Wishlist{})
		res, _, err := addWish(ctx, nil, addWishArgs{WishlistID: wishlist.ID.String(), Name: "Headphones 2"})
		wantError(t, res, err, "failed to check access to the wishlist")
	})

	t.Run("add_wish collaborator check", func(t *testing.T) {
		_, _, _, ctx := setup(t)
		dropTable(t, &models.WishlistCollaborator{})
		res, _, err := addWish(ctx, nil, addWishArgs{WishlistID: uuid.NewString(), Name: "Headphones 2"})
		wantError(t, res, err, "failed to check access to the wishlist")
	})

	t.Run("update_wish wish lookup", func(t *testing.T) {
		_, _, wish, ctx := setup(t)
		dropTable(t, &models.Wish{})
		note := "Black"
		res, _, err := updateWish(ctx, nil, updateWishArgs{WishID: wish.ID.String(), Note: &note})
		wantError(t, res, err, "failed to load the wish")
	})

	t.Run("update_wish access check", func(t *testing.T) {
		_, _, wish, ctx := setup(t)
		dropTable(t, &models.Wishlist{})
		note := "Black"
		res, _, err := updateWish(ctx, nil, updateWishArgs{WishID: wish.ID.String(), Note: &note})
		wantError(t, res, err, "failed to check access to the wishlist")
	})

	t.Run("delete_wish wish lookup", func(t *testing.T) {
		_, _, wish, ctx := setup(t)
		notified := replaceWishDeletedNotifier(t)
		dropTable(t, &models.Wish{})
		res, _, err := deleteWish(ctx, nil, deleteWishArgs{WishID: wish.ID.String()})
		wantError(t, res, err, "failed to load the wish")
		if len(*notified) != 0 {
			t.Errorf("claimers notified for %v after a failed delete", *notified)
		}
	})
}

// TestBuildServerRegistersWriteTools checks the write tools are offered next to
// the read tools, so a connected client can discover them.
func TestBuildServerRegistersWriteTools(t *testing.T) {
	setupMCPDB(t)
	owner := createMCPTestUser(t, "Ada")
	session := startMCPTestServer(t, owner.ID, writeScope)

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	names := map[string]bool{}
	for _, tool := range tools.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"list_wishlists", "list_wishes", "list_groups", "add_wish", "update_wish", "delete_wish"} {
		if !names[want] {
			t.Errorf("tool %q is not registered; got %v", want, names)
		}
	}
}
