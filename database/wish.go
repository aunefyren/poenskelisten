package database

import (
	"aunefyren/poenskelisten/models"
	"errors"

	"github.com/google/uuid"
)

// Get wishlist id from wish id
func GetWishlistIDFromWish(WishID uuid.UUID) (*uuid.UUID, error) {
	var wish models.Wish

	wishRecord := Instance.
		Where(&models.Wish{Enabled: true}).
		Where(&models.GormModel{ID: WishID}).
		Find(&wish)

	if wishRecord.Error != nil {
		return nil, wishRecord.Error
	} else if wishRecord.RowsAffected != 1 {
		return nil, nil
	}

	return &wish.WishlistID, nil
}

// Get wish by wish ID
func GetWishByWishID(wishID uuid.UUID) (*models.Wish, error) {
	var wish models.Wish

	wishRecord := Instance.
		Where(&models.Wish{Enabled: true}).
		Where(&models.GormModel{ID: wishID}).
		Find(&wish)

	if wishRecord.Error != nil {
		return nil, wishRecord.Error
	} else if wishRecord.RowsAffected != 1 {
		return nil, nil
	}

	return &wish, nil
}

func UpdateWishInDB(wishOriginal models.Wish) (wish models.Wish, err error) {
	wish = wishOriginal
	err = nil

	wishRecord := Instance.Save(wish)

	if wishRecord.Error != nil {
		return wish, wishRecord.Error
	}

	return
}

// Get wishes from wishlist
func GetWishesFromWishlist(WishlistID uuid.UUID) (bool, []models.Wish, error) {
	var wishes []models.Wish
	wishRecords := Instance.
		Order("wishes.created_at ASC").
		Where(&models.Wish{Enabled: true, WishlistID: WishlistID}).
		Joins("JOIN users ON users.id = wishes.owner_id").
		Where("users.enabled = ?", true).
		Find(&wishes)

	if wishRecords.Error != nil {
		return false, []models.Wish{}, wishRecords.Error
	} else if wishRecords.RowsAffected < 1 {
		return false, []models.Wish{}, nil
	}

	return true, wishes, nil
}

// get wish claims from wish, returns empty array without error if none are found.
func GetWishClaimFromWish(WishID uuid.UUID) ([]models.WishClaimObject, error) {
	var wishClaim models.WishClaim
	var wishWithUser models.WishClaimObject
	var wishArrayWithUser []models.WishClaimObject

	wishClaimRecords := Instance.
		Where(&models.WishClaim{Enabled: true, WishID: WishID}).
		Joins("JOIN users ON users.id = wish_claims.user_id").
		Where("users.enabled = ?", true).
		Find(&wishClaim)

	if wishClaimRecords.Error != nil {
		return []models.WishClaimObject{}, wishClaimRecords.Error
	} else if wishClaimRecords.RowsAffected < 1 {
		return []models.WishClaimObject{}, nil
	}

	userObject, err := GetUserInformation(wishClaim.UserID)
	if err != nil {
		return []models.WishClaimObject{}, err
	}

	userObjectMinimal := models.UserMinimal{
		GormModel: userObject.GormModel,
		FirstName: userObject.FirstName,
		LastName:  userObject.LastName,
		Email:     userObject.Email,
	}

	wishWithUser.User = userObjectMinimal
	wishWithUser.CreatedAt = wishClaim.CreatedAt
	wishWithUser.DeletedAt = wishClaim.DeletedAt
	wishWithUser.Enabled = wishClaim.Enabled
	wishWithUser.ID = wishClaim.ID
	wishWithUser.UpdatedAt = wishClaim.UpdatedAt
	wishWithUser.Wish = wishClaim.WishID

	wishArrayWithUser = append(wishArrayWithUser, wishWithUser)

	return wishArrayWithUser, err
}

// Verify if a user ID is an owner of a wish
func VerifyUserOwnershipToWish(UserID uuid.UUID, WishID uuid.UUID) (bool, error) {
	var wish models.Wish

	wishRecord := Instance.
		Where(&models.Wish{Enabled: true, OwnerID: UserID}).
		Where(&models.GormModel{ID: WishID}).
		Find(&wish)

	if wishRecord.Error != nil {
		return false, wishRecord.Error
	} else if wishRecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Verify if a user ID is an owner of a wish
func VerifyUserOwnershipToWishClaimByWish(UserID uuid.UUID, WishID uuid.UUID) (bool, error) {
	var wishClaim models.WishClaim

	wishClaimRecord := Instance.
		Where(&models.WishClaim{Enabled: true, WishID: WishID, UserID: UserID}).
		Find(&wishClaim)

	if wishClaimRecord.Error != nil {
		return false, wishClaimRecord.Error
	} else if wishClaimRecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Verify if a user ID is an owner of a wish
func VerifyWishIsClaimed(WishID uuid.UUID) (bool, error) {
	var wishClaim models.WishClaim

	wishClaimRecord := Instance.
		Where(&models.WishClaim{Enabled: true, WishID: WishID}).
		Joins("JOIN users ON users.id = wish_claims.user_id").
		Where("users.enabled = ?", true).
		Find(&wishClaim)

	if wishClaimRecord.Error != nil {
		return false, wishClaimRecord.Error
	} else if wishClaimRecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Set wish claim to disabled
func DeleteWishClaimByUserAndWish(WishID uuid.UUID, UserID uuid.UUID) error {
	var wishClaim models.WishClaim

	wishClaimRecords := Instance.
		Model(wishClaim).
		Where(&models.WishClaim{WishID: WishID, UserID: UserID}).
		Update("enabled", false)

	if wishClaimRecords.Error != nil {
		return wishClaimRecords.Error
	}
	if wishClaimRecords.RowsAffected != 1 {
		return errors.New("Failed to delete wish claim membership in database.")
	}
	return nil
}

// Get wish by wish ID
func GetWishlistByWishID(wishID uuid.UUID) (bool, models.Wishlist, error) {
	var wishlist models.Wishlist

	wishlistRecords := Instance.
		Where(&models.Wishlist{Enabled: true}).
		Joins("JOIN wishes ON wishlists.id = wishes.wishlist_id").
		Where("wishes.enabled = ? AND wishes.id = ?", true, wishID).
		Find(&wishlist)

	if wishlistRecords.Error != nil {
		return false, models.Wishlist{}, wishlistRecords.Error
	} else if wishlistRecords.RowsAffected != 1 {
		return false, models.Wishlist{}, nil
	}

	return true, wishlist, nil
}

func CreateWishInDB(wishDB models.Wish) (wish models.Wish, err error) {
	wish = models.Wish{}
	err = nil
	record := Instance.Create(&wishDB)

	if record.Error != nil {
		return wish, record.Error
	}

	if record.RowsAffected != 1 {
		return wish, errors.New("Wish not added to database.")
	}

	return wish, err
}

func CreateWishClaimInDB(wishClaimDB models.WishClaim) (wishClaim models.WishClaim, err error) {
	wishClaim = models.WishClaim{}
	err = nil
	record := Instance.Create(&wishClaimDB)

	if record.Error != nil {
		return wishClaim, record.Error
	}

	if record.RowsAffected != 1 {
		return wishClaim, errors.New("WishClaim not added to database.")
	}

	return wishClaim, err
}
