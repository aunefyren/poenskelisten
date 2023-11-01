package database

import (
	"aunefyren/poenskelisten/models"
	"errors"
	"time"
)

// Update values in wishlist object in DB
func UpdateWishlistValuesByID(wishlistID int, wishlistName string, wishlistDesc string, wishlistExpiration time.Time, wishlistClaimable bool, wishlistExpires bool) error {

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

	wishlistRecord = Instance.Model(wishlist).Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.ID = ?", wishlistID).Update("expires", wishlistExpires)
	if wishlistRecord.Error != nil {
		return wishlistRecord.Error
	}
	if wishlistRecord.RowsAffected != 1 {
		return errors.New("Expiration not changed in database.")
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

// Get wishlist collabs who are members of wishlist
func GetWishlistCollaboratorsFromWishlist(WishlistID int) (wishlistColab []models.WishlistCollaborator, err error) {
	wishlistColab = []models.WishlistCollaborator{}
	err = nil

	userRecords := Instance.Where("`wishlist_collaborators`.enabled = ?", 1).Joins("JOIN `users` on `wishlist_collaborators`.user = `users`.id").Where("`users`.enabled = ?", 1).Joins("JOIN `wishlists` on `wishlist_collaborators`.wishlist = `wishlists`.id").Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.id = ?", WishlistID).Find(&wishlistColab)
	if userRecords.Error != nil {
		return wishlistColab, userRecords.Error
	}

	return wishlistColab, nil
}

// Get wishlist collab by id
func GetWishlistCollaboratorByUserIDAndWishlistID(WishlistID int, UserID int) (wishlistColab models.WishlistCollaborator, err error) {
	wishlistColab = models.WishlistCollaborator{}
	err = nil

	userRecords := Instance.Where("`wishlist_collaborators`.enabled = ?", 1).Joins("JOIN `users` on `wishlist_collaborators`.user = `users`.id").Where("`users`.enabled = ?", 1).Joins("JOIN `wishlists` on `wishlist_collaborators`.wishlist = `wishlists`.id").Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.id = ?", WishlistID).Where("`users`.id = ?", UserID).Find(&wishlistColab)
	if userRecords.Error != nil {
		return wishlistColab, userRecords.Error
	}

	return wishlistColab, nil
}

// Create wishlist collaborator in DB
func CreateWishlistCollaboratorInDB(wishlistCollaborator models.WishlistCollaborator) error {
	record := Instance.Create(&wishlistCollaborator)

	if record.Error != nil {
		return record.Error
	}

	if record.RowsAffected != 1 {
		return errors.New("Wishlist not added to database.")
	}

	return nil
}

// Verify if a group id is a member of a wishlist
func VerifyWishlistCollaboratorToWishlist(WishlistID int, UserID int) (verified bool, err error) {
	verified = false
	err = nil
	wishlistColab := models.WishlistCollaborator{}

	wishlistmembershipprecord := Instance.Where("`wishlist_collaborators`.enabled = ?", 1).Where("`wishlist_collaborators`.wishlist = ?", WishlistID).Where("`wishlist_collaborators`.user = ?", UserID).Joins("JOIN `users` on `wishlist_collaborators`.user = `users`.id").Where("`users`.enabled = ?", 1).Find(&wishlistColab)
	if wishlistmembershipprecord.Error != nil {
		return verified, wishlistmembershipprecord.Error
	} else if wishlistmembershipprecord.RowsAffected != 1 {
		return verified, err
	}

	return true, err
}

// Set wishlist membership to disabled
func DeleteWishlistCollaboratorByWishlistCollaboratorID(WishlistCollaboratorID int) (err error) {
	wishlistCollaborator := models.WishlistCollaborator{}
	err = nil

	wishlistmembershiprecords := Instance.Model(wishlistCollaborator).Where("`wishlist_collaborators`.ID= ?", WishlistCollaboratorID).Update("enabled", 0)
	if wishlistmembershiprecords.Error != nil {
		return wishlistmembershiprecords.Error
	}
	if wishlistmembershiprecords.RowsAffected != 1 {
		return errors.New("Failed to delete wishlist collaboration in database.")
	}

	return nil
}

// Get all wishlists a user is an owner of
func GetWishlistsByUserIDThroughWishlistCollaborations(UserID int) (wishlists []models.Wishlist, err error) {
	wishlists = []models.Wishlist{}
	err = nil

	wishlistRecords := Instance.Order("`wishlists`.date desc, `wishlists`.name").Where("`wishlists`.enabled = ?", 1).Joins("JOIN `wishlist_collaborators` on `wishlists`.id = `wishlist_collaborators`.wishlist").Where("`wishlist_collaborators`.enabled = ?", 1).Joins("JOIN users on wishlist_collaborators.user = users.id").Where("`users`.enabled = ?", 1).Find(&wishlists)

	if wishlistRecords.Error != nil {
		return []models.Wishlist{}, wishlistRecords.Error
	}

	return wishlists, err
}
