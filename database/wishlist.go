package database

import (
	"aunefyren/poenskelisten/models"
	"errors"
	"time"
)

// Update values in wishlist object in DB
func UpdateWishlistValuesByID(wishlistID int, wishlistName string, wishlistDesc string, wishlistExpiration time.Time, wishlistClaimable bool) error {

	var wishlist models.Wishlist

	wishlistRecord := Instance.Model(wishlist).Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.ID = ?", wishlistID).Update("name", wishlistName)
	if wishlistRecord.Error != nil {
		return wishlistRecord.Error
	}
	if wishlistRecord.RowsAffected != 1 {
		return errors.New("Name not changed in database.")
	}

	wishlistRecord = Instance.Model(wishlist).Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.ID = ?", wishlistID).Update("description", wishlistDesc)
	if wishlistRecord.Error != nil {
		return wishlistRecord.Error
	}
	if wishlistRecord.RowsAffected != 1 {
		return errors.New("Description not changed in database.")
	}

	wishlistRecord = Instance.Model(wishlist).Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.ID = ?", wishlistID).Update("date", wishlistExpiration)
	if wishlistRecord.Error != nil {
		return wishlistRecord.Error
	}
	if wishlistRecord.RowsAffected != 1 {
		return errors.New("Expiration not changed in database.")
	}

	wishlistRecord = Instance.Model(wishlist).Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.ID = ?", wishlistID).Update("claimable", wishlistClaimable)
	if wishlistRecord.Error != nil {
		return wishlistRecord.Error
	}
	if wishlistRecord.RowsAffected != 1 {
		return errors.New("Claimability not changed in database.")
	}

	return nil

}

// Create wishlist in DB
func CreateWishlistInDB(wishlistdb models.Wishlist) error {

	record := Instance.Create(&wishlistdb)

	if record.Error != nil {
		return record.Error
	}

	if record.RowsAffected != 1 {
		return errors.New("Wishlist not added to database.")
	}

	return nil

}

// Get wishlist by wishlist ID
func GetWishlistByWishlistID(wishlistID int) (bool, models.Wishlist, error) {

	var wishlist models.Wishlist

	wishlistRecord := Instance.Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.id = ?", wishlistID).Find(&wishlist)

	if wishlistRecord.Error != nil {
		return false, models.Wishlist{}, wishlistRecord.Error
	} else if wishlistRecord.RowsAffected != 1 {
		return false, models.Wishlist{}, nil
	}

	return true, wishlist, nil

}
