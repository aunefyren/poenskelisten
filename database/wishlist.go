package database

import (
	"aunefyren/poenskelisten/models"
	"errors"
	"time"
)

// Update values in wishlist object in DB
func UpdateWishlistValuesByID(wishlistID int, wishlistName string, wishlistDesc string, wishlistExpiration time.Time) error {

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

	return nil

}
