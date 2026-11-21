package mcpserver

import (
	"aunefyren/poenskelisten/controllers"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The write tools hold wishes to the same rules as the web app's wish endpoints
// (controllers/wish.go), so a wish added by an assistant can't be one the web
// app would have refused: only the wishlist's owner or a collaborator may change
// it, a name has at least five characters and is unique on its wishlist, a link
// is an absolute http(s) URL, and text shown to other users carries no markup
// characters.

const scopeWishlistsWrite = "mcp:wishlists.write"

type addWishArgs struct {
	WishlistID string   `json:"wishlist_id" jsonschema:"the ID of the wishlist to add the wish to (from list_wishlists)"`
	Name       string   `json:"name" jsonschema:"the name of the wish: at least five characters, and not already used on this wishlist"`
	Note       string   `json:"note,omitempty" jsonschema:"an optional note, such as a size or a colour"`
	URL        string   `json:"url,omitempty" jsonschema:"an optional http or https link to the item"`
	Price      *float64 `json:"price,omitempty" jsonschema:"an optional price"`
}

type updateWishArgs struct {
	WishID     string   `json:"wish_id" jsonschema:"the ID of the wish to change (from list_wishes)"`
	Name       *string  `json:"name,omitempty" jsonschema:"a new name; leave it out to keep the current one"`
	Note       *string  `json:"note,omitempty" jsonschema:"a new note; an empty string removes the note"`
	URL        *string  `json:"url,omitempty" jsonschema:"a new http or https link; an empty string removes the link"`
	Price      *float64 `json:"price,omitempty" jsonschema:"a new price"`
	ClearPrice bool     `json:"clear_price,omitempty" jsonschema:"true removes the price; can't be combined with price"`
}

type deleteWishArgs struct {
	WishID string `json:"wish_id" jsonschema:"the ID of the wish to delete (from list_wishes)"`
}

type deletedWishOut struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// errWishlistNotAccessible is the one answer for a wishlist that doesn't exist
// and one the caller may not change, so a tool call can't be used to find out
// which wishlist IDs exist.
var errWishlistNotAccessible = errors.New("wishlist not found or not accessible")

// errWishNotAccessible does the same for wishes.
var errWishNotAccessible = errors.New("wish not found or not accessible")

// wishDeletedNotifier tells the claimers of a wish delete_wish removed; tests
// replace it.
var wishDeletedNotifier = controllers.NotifyWishDeleted

func registerWriteTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_wish",
		Description: "Add a wish to a wishlist you own or collaborate on. Provide the wishlist_id (from list_wishlists) and a name of at least five characters; note, url and price are optional. Returns the new wish.",
	}, addWish)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_wish",
		Description: "Change a wish's name, note, link or price. Provide the wish_id (from list_wishes) and only the fields to change: an empty note or url removes it, and clear_price removes the price. Returns the updated wish.",
	}, updateWish)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_wish",
		Description: "Delete a wish from a wishlist. Provide the wish_id (from list_wishes). If someone had claimed it, they are told by e-mail when the server has e-mail set up.",
	}, deleteWish)
}

func addWish(ctx context.Context, _ *mcp.CallToolRequest, args addWishArgs) (*mcp.CallToolResult, any, error) {
	userID, err := requireScope(ctx, scopeWishlistsWrite)
	if err != nil {
		return nil, nil, err
	}

	wishlistID, err := utilities.ParseID(strings.TrimSpace(args.WishlistID))
	if err != nil {
		return nil, nil, errors.New("invalid wishlist_id")
	}

	if err := requireWishlistEditor(userID, wishlistID); err != nil {
		return nil, nil, err
	}

	name := strings.TrimSpace(args.Name)
	note := strings.TrimSpace(args.Note)
	wishURL := strings.TrimSpace(args.URL)
	if err := validateWishName(name); err != nil {
		return nil, nil, err
	}
	if err := validateWishText("note", note); err != nil {
		return nil, nil, err
	}
	if err := validateWishURL(wishURL); err != nil {
		return nil, nil, err
	}

	unique, err := database.VerifyUniqueWishNameInWishlist(name, wishlistID)
	if err != nil {
		logger.Log.Error("Failed to verify unique wish name. Error: " + err.Error())
		return nil, nil, errors.New("failed to check the wish name")
	} else if !unique {
		return nil, nil, errors.New("there is already a wish with that name on this wishlist")
	}

	wish := models.Wish{
		OwnerID:    userID,
		WishlistID: wishlistID,
		Name:       name,
		Note:       note,
		URL:        wishURL,
		Price:      args.Price,
	}
	wish.ID = uuid.New()

	created, err := database.CreateWishInDB(wish)
	if err != nil {
		logger.Log.Error("Failed to create wish. Error: " + err.Error())
		return nil, nil, errors.New("failed to save the wish")
	}
	return jsonResult(newWishOut(created)), nil, nil
}

func updateWish(ctx context.Context, _ *mcp.CallToolRequest, args updateWishArgs) (*mcp.CallToolResult, any, error) {
	userID, err := requireScope(ctx, scopeWishlistsWrite)
	if err != nil {
		return nil, nil, err
	}

	wishID, err := utilities.ParseID(strings.TrimSpace(args.WishID))
	if err != nil {
		return nil, nil, errors.New("invalid wish_id")
	}

	if args.ClearPrice && args.Price != nil {
		return nil, nil, errors.New("give either price or clear_price, not both")
	}

	wish, err := loadEditableWish(userID, wishID)
	if err != nil {
		return nil, nil, err
	}

	if args.Name != nil {
		name := strings.TrimSpace(*args.Name)
		if name != wish.Name {
			if err := validateWishName(name); err != nil {
				return nil, nil, err
			}
			unique, err := database.VerifyUniqueWishNameInWishlistExcludingWish(name, wish.WishlistID, wish.ID)
			if err != nil {
				logger.Log.Error("Failed to verify unique wish name. Error: " + err.Error())
				return nil, nil, errors.New("failed to check the wish name")
			} else if !unique {
				return nil, nil, errors.New("there is already a wish with that name on this wishlist")
			}
			wish.Name = name
		}
	}

	if args.Note != nil {
		note := strings.TrimSpace(*args.Note)
		if err := validateWishText("note", note); err != nil {
			return nil, nil, err
		}
		wish.Note = note
	}

	if args.URL != nil {
		wishURL := strings.TrimSpace(*args.URL)
		if err := validateWishURL(wishURL); err != nil {
			return nil, nil, err
		}
		wish.URL = wishURL
	}

	if args.ClearPrice {
		wish.Price = nil
	} else if args.Price != nil {
		wish.Price = args.Price
	}

	updated, err := database.UpdateWishInDB(*wish)
	if err != nil {
		logger.Log.Error("Failed to update wish. Error: " + err.Error())
		return nil, nil, errors.New("failed to save the wish")
	}
	return jsonResult(newWishOut(updated)), nil, nil
}

func deleteWish(ctx context.Context, _ *mcp.CallToolRequest, args deleteWishArgs) (*mcp.CallToolResult, any, error) {
	userID, err := requireScope(ctx, scopeWishlistsWrite)
	if err != nil {
		return nil, nil, err
	}

	wishID, err := utilities.ParseID(strings.TrimSpace(args.WishID))
	if err != nil {
		return nil, nil, errors.New("invalid wish_id")
	}

	wish, err := loadEditableWish(userID, wishID)
	if err != nil {
		return nil, nil, err
	}

	// Deleting disables the wish, as the web app does, so its claims and
	// history stay behind for the claimer notification below.
	wish.Enabled = false
	deleted, err := database.UpdateWishInDB(*wish)
	if err != nil {
		logger.Log.Error("Failed to delete wish. Error: " + err.Error())
		return nil, nil, errors.New("failed to delete the wish")
	}

	// The wish is already gone from its wishlist, so tidying up after it only
	// logs a failure instead of reporting the delete as failed.
	if deleted.CategoryID != nil {
		controllers.CleanupWishCategoryIfEmpty(*deleted.CategoryID)
	}
	if err := controllers.DeleteWishImage(deleted.ID); err != nil {
		logger.Log.Error("Failed to delete image of deleted wish. Error: " + err.Error())
	}
	wishDeletedNotifier(deleted)

	return jsonResult(deletedWishOut{ID: deleted.ID.String(), Deleted: true}), nil, nil
}

// requireWishlistEditor checks that the user may change the wishes on a
// wishlist: its owner or one of its collaborators, as in the web app.
func requireWishlistEditor(userID uuid.UUID, wishlistID uuid.UUID) error {
	owns, err := database.VerifyUserOwnershipToWishlist(userID, wishlistID)
	if err != nil {
		logger.Log.Error("Failed to verify ownership of wishlist. Error: " + err.Error())
		return errors.New("failed to check access to the wishlist")
	} else if owns {
		return nil
	}

	collaborates, err := database.VerifyWishlistCollaboratorToWishlist(wishlistID, userID)
	if err != nil {
		logger.Log.Error("Failed to verify wishlist collaborator status. Error: " + err.Error())
		return errors.New("failed to check access to the wishlist")
	} else if !collaborates {
		return errWishlistNotAccessible
	}
	return nil
}

// loadEditableWish returns an enabled wish the user may change. A wish that
// doesn't exist, was deleted, or is on a wishlist the user can't change all
// get the same error.
func loadEditableWish(userID uuid.UUID, wishID uuid.UUID) (*models.Wish, error) {
	wish, err := database.GetWishByWishID(wishID)
	if err != nil {
		logger.Log.Error("Failed to get wish. Error: " + err.Error())
		return nil, errors.New("failed to load the wish")
	} else if wish == nil {
		return nil, errWishNotAccessible
	}

	if err := requireWishlistEditor(userID, wish.WishlistID); errors.Is(err, errWishlistNotAccessible) {
		return nil, errWishNotAccessible
	} else if err != nil {
		return nil, err
	}
	return wish, nil
}

func validateWishName(name string) error {
	if len(name) < 5 {
		return errors.New("the name of the wish must be five or more characters")
	}
	return validateWishText("name", name)
}

// validateWishText applies the check the web app's `text` sanitize tag does:
// wish text is shown to other users, so it may not carry markup characters.
func validateWishText(field string, value string) error {
	if valid, requirements := utilities.ValidateTextCharacters(value); !valid {
		return fmt.Errorf("invalid %s: %s", field, requirements)
	}
	return nil
}

func validateWishURL(wishURL string) error {
	if wishURL == "" {
		return nil
	}
	if err := validateWishText("url", wishURL); err != nil {
		return err
	}
	if !controllers.IsValidWishURL(wishURL) {
		return errors.New("invalid url: it must be an absolute http or https link")
	}
	return nil
}
