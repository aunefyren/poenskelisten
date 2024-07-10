package database

import (
	"aunefyren/poenskelisten/models"
	"errors"

	"github.com/google/uuid"
)

// Get wishlist id from wish id
func GetWishlistIDFromWish(WishID uuid.UUID) (*uuid.UUID, error) {
	var wish models.Wish

	wishRecord := Instance.Where(&models.Wish{Enabled: true}).Where(&models.GormModel{ID: WishID}).Find(&wish)

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

	wishRecord := Instance.Where(&models.Wish{Enabled: true}).Where(&models.GormModel{ID: wishID}).Find(&wish)

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
	wishrecords := Instance.
		Order("created_at ASC").
		Where(&models.Wish{Enabled: true, WishlistID: WishlistID}).
		Joins("JOIN users ON users.id = wishes.owner_id").
		Where("users.enabled = ?", true).
		Find(&wishes)

	if wishrecords.Error != nil {
		return false, []models.Wish{}, wishrecords.Error
	} else if wishrecords.RowsAffected < 1 {
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
	wishrecord := Instance.Where(&models.Wish{Enabled: true, OwnerID: UserID}).Where(&models.GormModel{ID: WishID}).Find(&wish)
	if wishrecord.Error != nil {
		return false, wishrecord.Error
	} else if wishrecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Verify if a user ID is an owner of a wish
func VerifyUserOwnershipToWishClaimByWish(UserID uuid.UUID, WishID uuid.UUID) (bool, error) {
	var wishclaim models.WishClaim
	wishclaimrecord := Instance.Where(&models.WishClaim{Enabled: true, WishID: WishID, UserID: UserID}).Find(&wishclaim)
	if wishclaimrecord.Error != nil {
		return false, wishclaimrecord.Error
	} else if wishclaimrecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Verify if a user ID is an owner of a wish
func VerifyWishIsClaimed(WishID uuid.UUID) (bool, error) {
	var wishclaim models.WishClaim
	wishclaimrecord := Instance.
		Where(&models.WishClaim{Enabled: true, WishID: WishID}).
		Joins("JOIN users ON users.id = wish_claims.user_id").
		Where("users.enabled = ?", true).
		Find(&wishclaim)
	if wishclaimrecord.Error != nil {
		return false, wishclaimrecord.Error
	} else if wishclaimrecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Set wish claim to disabled
func DeleteWishClaimByUserAndWish(WishID uuid.UUID, UserID uuid.UUID) error {
	var wishclaim models.WishClaim
	wishclaimrecords := Instance.Model(wishclaim).Where(&models.WishClaim{WishID: WishID, UserID: UserID}).Update("enabled", 0)
	if wishclaimrecords.Error != nil {
		return wishclaimrecords.Error
	}
	if wishclaimrecords.RowsAffected != 1 {
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
