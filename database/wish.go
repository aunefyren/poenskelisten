package database

import (
	"aunefyren/poenskelisten/models"
	"errors"

	"github.com/google/uuid"
)

// Get wishlist id from wish id
func GetWishlistFromWish(WishID uuid.UUID) (bool, uuid.UUID, error) {
	var wish models.Wish
	wishrecord := Instance.Where("`wishes`.enabled = ?", 1).Where("`wishes`.id = ?", WishID).Find(&wish)
	if wishrecord.Error != nil {
		return false, uuid.UUID{}, wishrecord.Error
	} else if wishrecord.RowsAffected != 1 {
		return false, uuid.UUID{}, nil
	}

	return true, wish.WishlistID, nil
}

// Get wish by wish ID
func GetWishByWishID(wishID uuid.UUID) (bool, models.Wish, error) {

	var wish models.Wish

	wishRecord := Instance.Where("`wishes`.enabled = ?", 1).Where("`wishes`.id = ?", wishID).Find(&wish)

	if wishRecord.Error != nil {
		return false, models.Wish{}, wishRecord.Error
	} else if wishRecord.RowsAffected != 1 {
		return false, models.Wish{}, nil
	}

	return true, wish, nil

}

func UpdateWishValuesInDatabase(wishID uuid.UUID, wishName string, wishNote string, wishURL string, wishprice float64) error {

	var wish models.Wish

	wishRecord := Instance.Model(wish).Where("`wishes`.enabled = ?", 1).Where("`wishes`.ID = ?", wishID).Update("name", wishName)
	if wishRecord.Error != nil {
		return wishRecord.Error
	} else if wishRecord.RowsAffected != 1 {
		return errors.New("Name not changed in database.")
	}

	wishRecord = Instance.Model(wish).Where("`wishes`.enabled = ?", 1).Where("`wishes`.ID = ?", wishID).Update("note", wishNote)
	if wishRecord.Error != nil {
		return wishRecord.Error
	} else if wishRecord.RowsAffected != 1 {
		return errors.New("Note not changed in database.")
	}

	wishRecord = Instance.Model(wish).Where("`wishes`.enabled = ?", 1).Where("`wishes`.ID = ?", wishID).Update("url", wishURL)
	if wishRecord.Error != nil {
		return wishRecord.Error
	} else if wishRecord.RowsAffected != 1 {
		return errors.New("URL not changed in database.")
	}

	wishRecord = Instance.Model(wish).Where("`wishes`.enabled = ?", 1).Where("`wishes`.ID = ?", wishID).Update("price", wishprice)
	if wishRecord.Error != nil {
		return wishRecord.Error
	} else if wishRecord.RowsAffected != 1 {
		return errors.New("Price not changed in database.")
	}

	return nil

}

// Get wishes from wishlist
func GetWishesFromWishlist(WishlistID uuid.UUID) (bool, []models.Wish, error) {

	var wishes []models.Wish

	wishrecords := Instance.Where("`wishes`.enabled = ?", 1).Where("`wishes`.wishlist_id = ?", WishlistID).Joins("JOIN `users` on `users`.id = `wishes`.owner_id").Where("`users`.enabled = ?", 1).Find(&wishes)
	if wishrecords.Error != nil {
		return false, []models.Wish{}, wishrecords.Error
	} else if wishrecords.RowsAffected < 1 {
		return false, []models.Wish{}, nil
	}

	return true, wishes, nil
}

// get wish claims from wish, returns empty array without error if none are found.
func GetWishClaimFromWish(WishID uuid.UUID) ([]models.WishClaimObject, error) {
	var wish_claim models.WishClaim
	var wish_with_user models.WishClaimObject
	var wisharray_with_user []models.WishClaimObject

	wishclaimrecords := Instance.
		Where("`wish_claims`.enabled = ?", 1).
		Where("`wish_claims`.wish_id = ?", WishID).
		Joins("JOIN `users` on `users`.id = `wish_claims`.user_id").
		Where("`users`.enabled = ?", 1).Find(&wish_claim)

	if wishclaimrecords.Error != nil {
		return []models.WishClaimObject{}, wishclaimrecords.Error
	} else if wishclaimrecords.RowsAffected < 1 {
		return []models.WishClaimObject{}, nil
	}

	user_object, err := GetUserInformation(wish_claim.UserID)
	if err != nil {
		return []models.WishClaimObject{}, err
	}

	wish_with_user.User = user_object
	wish_with_user.CreatedAt = wish_claim.CreatedAt
	wish_with_user.DeletedAt = wish_claim.DeletedAt
	wish_with_user.Enabled = wish_claim.Enabled
	wish_with_user.ID = wish_claim.ID
	wish_with_user.UpdatedAt = wish_claim.UpdatedAt
	wish_with_user.Wish = wish_claim.WishID

	wisharray_with_user = append(wisharray_with_user, wish_with_user)

	return wisharray_with_user, err
}

// Verify if a user ID is an owner of a wish
func VerifyUserOwnershipToWish(UserID uuid.UUID, WishID uuid.UUID) (bool, error) {
	var wish models.Wish
	wishrecord := Instance.Where("`wishes`.enabled = ?", 1).Where("`wishes`.id = ?", WishID).Where("`wishes`.owner_id = ?", UserID).Find(&wish)
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
	wishclaimrecord := Instance.Where("`wish_claims`.enabled = ?", 1).Where("`wish_claims`.wish_id = ?", WishID).Where("`wish_claims`.user_id = ?", UserID).Find(&wishclaim)
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
	wishclaimrecord := Instance.Where("`wish_claims`.enabled = ?", 1).Where("`wish_claims`.wish_id = ?", WishID).Joins("JOIN `users` on `users`.id = `wish_claims`.user_id").Where("`users`.enabled = ?", 1).Find(&wishclaim)
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
	wishclaimrecords := Instance.Model(wishclaim).Where("`wish_claims`.wish_id = ?", WishID).Where("`wish_claims`.user_id = ?", UserID).Update("enabled", 0)
	if wishclaimrecords.Error != nil {
		return wishclaimrecords.Error
	}
	if wishclaimrecords.RowsAffected != 1 {
		return errors.New("Failed to delete wish claim membership in database.")
	}
	return nil
}

// Set wish to disabled
func DeleteWish(WishID uuid.UUID) error {
	var wish models.Wish
	wishrecords := Instance.Model(wish).Where("`wishes`.id = ?", WishID).Update("enabled", 0)
	if wishrecords.Error != nil {
		return wishrecords.Error
	}
	if wishrecords.RowsAffected != 1 {
		return errors.New("Failed to delete wish in database.")
	}
	return nil
}
