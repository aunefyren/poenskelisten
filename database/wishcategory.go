package database

import (
	"aunefyren/poenskelisten/models"
	"errors"

	"github.com/google/uuid"
)

// Create a wish category
func CreateWishCategoryInDB(category models.WishCategory) error {
	record := Instance.Create(&category)

	if record.Error != nil {
		return record.Error
	}
	if record.RowsAffected != 1 {
		return errors.New("wish category not added to database")
	}

	return nil
}

// Get all enabled categories for a wishlist, ordered for stable display
func GetWishCategoriesFromWishlist(wishlistID uuid.UUID) ([]models.WishCategory, error) {
	var categories []models.WishCategory

	record := Instance.
		Order("wish_categories.sort_order ASC").
		Order("wish_categories.created_at ASC").
		Where(&models.WishCategory{Enabled: true, WishlistID: wishlistID}).
		Find(&categories)

	if record.Error != nil {
		return []models.WishCategory{}, record.Error
	}

	return categories, nil
}

// Get a single enabled category by ID. Returns nil without error when not found.
func GetWishCategoryByID(categoryID uuid.UUID) (*models.WishCategory, error) {
	var category models.WishCategory

	record := Instance.
		Where(&models.WishCategory{Enabled: true}).
		Where(&models.GormModel{ID: categoryID}).
		Find(&category)

	if record.Error != nil {
		return nil, record.Error
	} else if record.RowsAffected != 1 {
		return nil, nil
	}

	return &category, nil
}

// Get an enabled category by name within a wishlist. Returns nil without error
// when no match exists. Used to re-use a category when a wish is created with a
// category name that already exists, rather than duplicating it.
func GetWishCategoryByNameInWishlist(name string, wishlistID uuid.UUID) (*models.WishCategory, error) {
	var category models.WishCategory

	record := Instance.
		Where(&models.WishCategory{Enabled: true, WishlistID: wishlistID, Name: name}).
		Limit(1).
		Find(&category)

	if record.Error != nil {
		return nil, record.Error
	} else if record.RowsAffected == 0 {
		return nil, nil
	}

	return &category, nil
}

// Get the sort order to assign to the next category added to a wishlist, so new
// categories sort after existing ones.
func GetNextWishCategorySortOrder(wishlistID uuid.UUID) (int, error) {
	var maxOrder *int

	record := Instance.
		Model(&models.WishCategory{}).
		Where(&models.WishCategory{Enabled: true, WishlistID: wishlistID}).
		Select("MAX(sort_order)").
		Scan(&maxOrder)

	if record.Error != nil {
		return 0, record.Error
	}
	if maxOrder == nil {
		return 0, nil
	}

	return *maxOrder + 1, nil
}

// Count enabled wishes currently assigned to a category. Used to auto-clean up
// categories that have become empty.
func CountEnabledWishesInCategory(categoryID uuid.UUID) (int64, error) {
	var count int64

	record := Instance.
		Model(&models.Wish{}).
		Where(&models.Wish{Enabled: true, CategoryID: &categoryID}).
		Count(&count)

	if record.Error != nil {
		return 0, record.Error
	}

	return count, nil
}

// Detach every wish from a category by nulling their CategoryID.
func ClearWishCategoryFromWishes(categoryID uuid.UUID) error {
	record := Instance.
		Model(&models.Wish{}).
		Where("category_id = ?", categoryID).
		Update("category_id", nil)

	return record.Error
}

// Soft-delete a category and detach any wishes still pointing at it.
func DeleteWishCategory(categoryID uuid.UUID) error {
	// Detach wishes first so no wish is left referencing a disabled category.
	err := ClearWishCategoryFromWishes(categoryID)
	if err != nil {
		return err
	}

	var category models.WishCategory

	record := Instance.
		Model(category).
		Where(&models.GormModel{ID: categoryID}).
		Update("enabled", false)

	if record.Error != nil {
		return record.Error
	}
	if record.RowsAffected != 1 {
		return errors.New("failed to delete wish category in database")
	}

	return nil
}
