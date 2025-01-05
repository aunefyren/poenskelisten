package database

import (
	"aunefyren/poenskelisten/models"
	"errors"

	"github.com/google/uuid"
)

// Update values in wishlist object in DB
func UpdateWishlistInDB(wishlist models.Wishlist) (models.Wishlist, error) {
	wishlistRecord := Instance.Save(&wishlist)
	if wishlistRecord.Error != nil {
		return wishlist, wishlistRecord.Error
	}
	if wishlistRecord.RowsAffected != 1 {
		return wishlist, errors.New("Wishlist not changed in database.")
	}

	return wishlist, nil
}

// Create wishlist in DB
func CreateWishlistInDB(wishlistDB models.Wishlist) (wishlist models.Wishlist, err error) {
	wishlist = models.Wishlist{}
	err = nil
	record := Instance.Create(&wishlistDB)

	if record.Error != nil {
		return wishlistDB, record.Error
	}

	if record.RowsAffected != 1 {
		return wishlistDB, errors.New("Wishlist not added to database.")
	}

	return wishlistDB, err
}

// Get wishlist by wishlist ID
func GetWishlistByWishlistID(wishlistID uuid.UUID) (bool, models.Wishlist, error) {

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
func GetWishlistCollaboratorsFromWishlist(WishlistID uuid.UUID) (wishlistColab []models.WishlistCollaborator, err error) {
	wishlistColab = []models.WishlistCollaborator{}
	err = nil

	userRecords := Instance.Where("`wishlist_collaborators`.enabled = ?", 1).Joins("JOIN `users` on `wishlist_collaborators`.user_id = `users`.id").Where("`users`.enabled = ?", 1).Joins("JOIN `wishlists` on `wishlist_collaborators`.wishlist_id = `wishlists`.id").Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.id = ?", WishlistID).Find(&wishlistColab)
	if userRecords.Error != nil {
		return wishlistColab, userRecords.Error
	}

	return wishlistColab, nil
}

// Get wishlist collab by id
func GetWishlistCollaboratorByUserIDAndWishlistID(WishlistID uuid.UUID, UserID uuid.UUID) (wishlistColab models.WishlistCollaborator, err error) {
	wishlistColab = models.WishlistCollaborator{}
	err = nil

	userRecords := Instance.Where("`wishlist_collaborators`.enabled = ?", 1).Joins("JOIN `users` on `wishlist_collaborators`.user_id = `users`.id").Where("`users`.enabled = ?", 1).Joins("JOIN `wishlists` on `wishlist_collaborators`.wishlist_id = `wishlists`.id").Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.id = ?", WishlistID).Where("`users`.id = ?", UserID).Find(&wishlistColab)
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
func VerifyWishlistCollaboratorToWishlist(WishlistID uuid.UUID, UserID uuid.UUID) (verified bool, err error) {
	verified = false
	err = nil
	wishlistColab := models.WishlistCollaborator{}

	wishlistmembershipprecord := Instance.Where("`wishlist_collaborators`.enabled = ?", 1).Where("`wishlist_collaborators`.wishlist_id = ?", WishlistID).Where("`wishlist_collaborators`.user_id = ?", UserID).Joins("JOIN `users` on `wishlist_collaborators`.user_id = `users`.id").Where("`users`.enabled = ?", 1).Find(&wishlistColab)
	if wishlistmembershipprecord.Error != nil {
		return verified, wishlistmembershipprecord.Error
	} else if wishlistmembershipprecord.RowsAffected != 1 {
		return verified, err
	}

	return true, err
}

// Set wishlist membership to disabled
func DeleteWishlistCollaboratorByWishlistCollaboratorID(WishlistCollaboratorID uuid.UUID) (err error) {
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
func GetWishlistsByUserIDThroughWishlistCollaborations(UserID uuid.UUID) (wishlists []models.Wishlist, err error) {
	wishlists = []models.Wishlist{}
	err = nil

	wishlistRecords := Instance.Order("`wishlists`.date desc, `wishlists`.name").Where("`wishlists`.enabled = ?", 1).Joins("JOIN `wishlist_collaborators` on `wishlists`.id = `wishlist_collaborators`.wishlist_id").Where("`wishlist_collaborators`.enabled = ?", 1).Where("`wishlist_collaborators`.user_id = ?", UserID).Joins("JOIN `users` on `wishlist_collaborators`.user_id = `users`.id").Where("`users`.enabled = ?", 1).Find(&wishlists)

	if wishlistRecords.Error != nil {
		return []models.Wishlist{}, wishlistRecords.Error
	}

	return wishlists, err
}

// Get all wishlists in groups
func GetWishlistsFromGroup(GroupID uuid.UUID) ([]models.Wishlist, error) {
	var wishlists []models.Wishlist
	wishlistRecords := Instance.
		Where("`wishlists`.enabled = ?", 1).
		Joins("JOIN wishlist_memberships on wishlist_memberships.wishlist_id = wishlists.id").
		Where("`wishlist_memberships`.group_id = ?", GroupID).
		Where("`wishlist_memberships`.enabled = ?", 1).
		Joins("JOIN `groups` on `wishlist_memberships`.group_id = `groups`.ID").
		Where("`groups`.enabled = ?", 1).
		Joins("JOIN `users` on `wishlists`.owner_id = `users`.id").
		Where("`users`.enabled = ?", 1).
		Find(&wishlists)

	if wishlistRecords.Error != nil {
		return []models.Wishlist{}, wishlistRecords.Error
	} else if wishlistRecords.RowsAffected == 0 {
		return []models.Wishlist{}, nil
	}

	return wishlists, nil
}

// Get all wishlists a user is an owner of
func GetOwnedWishlists(UserID uuid.UUID) (wishlists []models.Wishlist, err error) {
	wishlists = []models.Wishlist{}
	err = nil

	wishlistRecords := Instance.Order("`wishlists`.date desc, `wishlists`.name").Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.owner_id = ?", UserID).Joins("JOIN users on `users`.id = `wishlists`.owner_id").Where("`users`.enabled = ?", 1).Find(&wishlists)

	if wishlistRecords.Error != nil {
		return []models.Wishlist{}, wishlistRecords.Error
	}

	return wishlists, err
}

// Get all wishlists a user is an owner of
func GetWishlist(WishlistID uuid.UUID) (models.Wishlist, error) {
	var wishlist models.Wishlist
	wishlistRecords := Instance.Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.id = ?", WishlistID).Find(&wishlist)

	if wishlistRecords.Error != nil {
		return models.Wishlist{}, wishlistRecords.Error
	} else if wishlistRecords.RowsAffected != 1 {
		return models.Wishlist{}, errors.New("Wishlist not found.")
	}

	return wishlist, nil
}

// Verify if a wish name in wishlist is unique
func VerifyUniqueWishNameinWishlist(WishName string, WishlistID uuid.UUID) (bool, error) {
	var wish models.Wish
	wishesrecord := Instance.Where("`wishes`.enabled = ?", 1).Where("`wishes`.wishlist_id = ?", WishlistID).Where("`wishes`.name = ?", WishName).Find(&wish)
	if wishesrecord.Error != nil {
		return false, wishesrecord.Error
	} else if wishesrecord.RowsAffected != 0 {
		return false, nil
	}
	return true, nil
}

// Verify if a wishlist name in group is unique
func VerifyUniqueWishlistNameForUser(WishlistName string, UserID uuid.UUID) (bool, error) {
	var wishlist models.Wishlist
	wishlistRecord := Instance.Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.owner_id = ?", UserID).Where("`wishlists`.name = ?", WishlistName).Find(&wishlist)
	if wishlistRecord.Error != nil {
		return false, wishlistRecord.Error
	} else if wishlistRecord.RowsAffected != 0 {
		return false, nil
	}
	return true, nil
}

// Get owner id of wishlist
func GetWishlistOwner(WishlistID uuid.UUID) (uuid.UUID, error) {
	var wishlist models.Wishlist
	wishlistRecord := Instance.Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.id = ?", WishlistID).Find(&wishlist)
	if wishlistRecord.Error != nil {
		return uuid.UUID{}, wishlistRecord.Error
	} else if wishlistRecord.RowsAffected != 1 {
		return uuid.UUID{}, errors.New("Failed to find correct wishlist in DB.")
	}

	return wishlist.OwnerID, nil
}

// Verify if a group ID is a member of a wishlist
func VerifyUserMembershipToGroupmembershipToWishlist(UserID uuid.UUID, WishlistID uuid.UUID) (bool, error) {
	var wishlistmembership models.WishlistMembership
	wishlistmembershiprecord := Instance.Where("`wishlist_memberships`.enabled = ?", 1).Where("`wishlist_memberships`.wishlist_id = ?", WishlistID).Joins("JOIN `groups` on `groups`.id = `wishlist_memberships`.group_id").Where("`groups`.enabled = ?", 1).Joins("JOIN `group_memberships` on `group_memberships`.group_id = `groups`.id").Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.member_id = ?", UserID).Find(&wishlistmembership)
	if wishlistmembershiprecord.Error != nil {
		return false, wishlistmembershiprecord.Error
	} else if wishlistmembershiprecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Verify if a user ID is an owner of a wishlist
func VerifyUserOwnershipToWishlist(UserID uuid.UUID, WishlistID uuid.UUID) (bool, error) {
	var wishlist models.Wishlist

	wishlistRecord := Instance.
		Where("`wishlists`.enabled = ?", 1).
		Where("`wishlists`.id = ?", WishlistID).
		Where("`wishlists`.owner_id = ?", UserID).
		Find(&wishlist)

	if wishlistRecord.Error != nil {
		return false, wishlistRecord.Error
	} else if wishlistRecord.RowsAffected != 1 {
		return false, nil
	}

	return true, nil
}

// Get user information from wishlist
func GetUserMembersFromWishlist(WishlistID uuid.UUID) ([]models.User, error) {
	var users []models.User
	var group_memberships []models.GroupMembership

	membershiprecords := Instance.Where("`group_memberships`.enabled = ?", 1).Joins("JOIN `groups` on `group_memberships`.group_id = `groups`.id").Where("`groups`.enabled = ?", 1).Joins("JOIN `wishlist_memberships` on `wishlist_memberships`.group_id = `groups`.id").Where("`wishlist_memberships`.enabled = ?", 1).Joins("JOIN `wishlists` on `wishlists`.id = `wishlist_memberships`.wishlist_id").Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.id = ?", WishlistID).Joins("JOIN `users` on `group_memberships`.group_id = `users`.id").Where("`users`.enabled = ?", 1).Where("`group_memberships`.group_id != `wishlists`.owner_id").Find(&group_memberships)
	if membershiprecords.Error != nil {
		return []models.User{}, membershiprecords.Error
	}

	for _, membership := range group_memberships {
		user_object, err := GetUserInformation(membership.MemberID)
		if err != nil {
			return []models.User{}, err
		}
		users = append(users, user_object)
	}

	if len(users) == 0 {
		users = []models.User{}
	}

	return users, nil
}

func GetMembershipIDForGroupToWishlist(WishlistID uuid.UUID, GroupID uuid.UUID) (membershipFound bool, wishlistMembership models.WishlistMembership, err error) {
	wishlistMembership = models.WishlistMembership{}
	membershipFound = false
	err = nil

	wishlistmembershiprecord := Instance.Where("`wishlist_memberships`.enabled = ?", 1).Where("`wishlist_memberships`.wishlist_id = ?", WishlistID).Where("`wishlist_memberships`.group_id = ?", GroupID).Find(&wishlistMembership)
	if wishlistmembershiprecord.Error != nil {
		return membershipFound, wishlistMembership, wishlistmembershiprecord.Error
	} else if wishlistmembershiprecord.RowsAffected != 1 {
		return false, wishlistMembership, errors.New("Failed to find membership.")
	}

	return true, wishlistMembership, err
}

// Get wishlist by wishlist ID
func GetPublicWishListByWishlistHash(wishlistHash uuid.UUID) (bool, models.Wishlist, error) {
	var wishlist models.Wishlist

	wishlistRecord := Instance.Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.public = ?", 1).Where("`wishlists`.public_hash = ?", wishlistHash).Find(&wishlist)

	if wishlistRecord.Error != nil {
		return false, models.Wishlist{}, wishlistRecord.Error
	} else if wishlistRecord.RowsAffected != 1 {
		return false, models.Wishlist{}, nil
	}

	return true, wishlist, nil
}

// Create wishlist in DB
func CreateWishlistMembershipInDB(wishlistMembership models.WishlistMembership) (newWishlistMembership models.WishlistMembership, err error) {
	err = nil
	record := Instance.Create(&wishlistMembership)

	if record.Error != nil {
		return wishlistMembership, record.Error
	}

	if record.RowsAffected != 1 {
		return wishlistMembership, errors.New("Wishlist not added to database.")
	}

	return wishlistMembership, err
}
