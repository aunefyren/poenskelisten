package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ResolveWishCategoryForWish turns the category fields of a wish create/update
// request into a concrete *CategoryID to store on the wish.
//
//   - categoryName set  -> re-use an existing category with that name in the
//     wishlist, otherwise create a new one.
//   - categoryID set    -> validate it belongs to this wishlist and use it.
//   - neither set        -> the wish is uncategorized (nil).
//
// A non-empty userError is a validation problem to surface to the caller as a
// 400; a non-nil err is an internal failure.
func ResolveWishCategoryForWish(wishlistID uuid.UUID, ownerID uuid.UUID, categoryID *uuid.UUID, categoryName string) (resolved *uuid.UUID, userError string, err error) {
	categoryName = strings.TrimSpace(categoryName)

	// Inline creation / re-use by name takes precedence over a passed ID.
	if categoryName != "" {
		if len(categoryName) < 2 {
			return nil, "The name of the category must be two or more letters.", nil
		}

		stringMatch, requirements, err := utilities.ValidateTextCharacters(categoryName)
		if err != nil {
			return nil, "", err
		} else if !stringMatch {
			return nil, requirements, nil
		}

		// Re-use an existing category with the same name instead of duplicating.
		existing, err := database.GetWishCategoryByNameInWishlist(categoryName, wishlistID)
		if err != nil {
			return nil, "", err
		}
		if existing != nil {
			return &existing.ID, "", nil
		}

		sortOrder, err := database.GetNextWishCategorySortOrder(wishlistID)
		if err != nil {
			return nil, "", err
		}

		category := models.WishCategory{
			Name:       categoryName,
			WishlistID: wishlistID,
			OwnerID:    ownerID,
			SortOrder:  sortOrder,
			Enabled:    true,
		}
		category.ID = uuid.New()

		err = database.CreateWishCategoryInDB(category)
		if err != nil {
			return nil, "", err
		}

		return &category.ID, "", nil
	}

	// Assign to an existing category by ID.
	if categoryID != nil {
		category, err := database.GetWishCategoryByID(*categoryID)
		if err != nil {
			return nil, "", err
		}
		if category == nil || category.WishlistID != wishlistID {
			return nil, "The selected category does not belong to this wishlist.", nil
		}
		return &category.ID, "", nil
	}

	// No category requested.
	return nil, "", nil
}

// CleanupWishCategoryIfEmpty soft-deletes a category once it no longer holds any
// enabled wishes. Because categories are managed inline (there is no dedicated
// management panel), this keeps empty categories from lingering after their last
// wish is deleted or moved out. Failures are logged, not fatal to the caller.
func CleanupWishCategoryIfEmpty(categoryID uuid.UUID) {
	count, err := database.CountEnabledWishesInCategory(categoryID)
	if err != nil {
		logger.Log.Warn("Failed to count wishes in category '" + categoryID.String() + "'. Skipping cleanup. Error: " + err.Error())
		return
	}

	if count == 0 {
		err = database.DeleteWishCategory(categoryID)
		if err != nil {
			logger.Log.Warn("Failed to auto-remove empty category '" + categoryID.String() + "'. Error: " + err.Error())
		}
	}
}

func APIGetWishlistCategories(context *gin.Context) {
	var wishlistIDString = context.Param("wishlist_id")

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	// Parse wishlist id
	wishlistID, err := uuid.Parse(wishlistIDString)
	if err != nil {
		logger.Log.Error("Failed to parse wishlist ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wishlist ID."})
		context.Abort()
		return
	}

	// Only users who can add wishes need the category picker: owners and collaborators.
	WishlistOwnership, err := database.VerifyUserOwnershipToWishlist(userID, wishlistID)
	if err != nil {
		logger.Log.Error("Failed to verify ownership of wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership of wishlist."})
		context.Abort()
		return
	}

	collaborationStatus, err := database.VerifyWishlistCollaboratorToWishlist(wishlistID, userID)
	if err != nil {
		logger.Log.Error("Failed to verify wishlist collaborator status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist collaborator status."})
		context.Abort()
		return
	}

	if !WishlistOwnership && !collaborationStatus {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You are not an owner or collaborator of this wishlist."})
		context.Abort()
		return
	}

	categories, err := database.GetWishCategoriesFromWishlist(wishlistID)
	if err != nil {
		logger.Log.Error("Failed to get categories from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get categories from database."})
		context.Abort()
		return
	}

	categoryObjects := []models.WishCategoryObject{}
	for _, category := range categories {
		categoryObjects = append(categoryObjects, models.WishCategoryObject{
			GormModel:  category.GormModel,
			Name:       category.Name,
			WishlistID: category.WishlistID,
			SortOrder:  category.SortOrder,
		})
	}

	context.JSON(http.StatusOK, gin.H{"categories": categoryObjects, "message": "Categories retrieved."})
}
