package database

import (
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
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

	wishlistRecord := Instance.
		Where(&models.Wishlist{Enabled: true}).
		Where(&models.GormModel{ID: wishlistID}).
		Find(&wishlist)

	if wishlistRecord.Error != nil {
		return false, models.Wishlist{}, wishlistRecord.Error
	} else if wishlistRecord.RowsAffected != 1 {
		return false, models.Wishlist{}, nil
	}

	return true, wishlist, nil

}

// Get wishlist collaborators who are members of wishlist
func GetWishlistCollaboratorsFromWishlist(WishlistID uuid.UUID) (WishlistCollaborator []models.WishlistCollaborator, err error) {
	WishlistCollaborator = []models.WishlistCollaborator{}

	userRecords := Instance.
		Where(&models.WishlistCollaborator{Enabled: true}).
		Joins("JOIN users ON wishlist_collaborators.user_id = users.id").
		Where("users.enabled = ?", true).
		Joins("JOIN wishlists ON wishlist_collaborators.wishlist_id = wishlists.id").
		Where("wishlists.enabled = ? AND wishlists.id = ?", true, WishlistID).
		Find(&WishlistCollaborator)

	if userRecords.Error != nil {
		return WishlistCollaborator, userRecords.Error
	}

	return WishlistCollaborator, nil
}

// Get wishlist collaborator by id
func GetWishlistCollaboratorByUserIDAndWishlistID(WishlistID uuid.UUID, UserID uuid.UUID) (WishlistCollaborator models.WishlistCollaborator, err error) {
	WishlistCollaborator = models.WishlistCollaborator{}

	userRecords := Instance.
		Where(&models.WishlistCollaborator{Enabled: true}).
		Joins("JOIN users ON wishlist_collaborators.user_id = users.id").
		Where("users.enabled = ? AND users.id = ?", true, UserID).
		Joins("JOIN wishlists ON wishlist_collaborators.wishlist_id = wishlists.id").
		Where("wishlists.enabled = ? AND wishlists.id = ?", true, WishlistID).
		Find(&WishlistCollaborator)

	if userRecords.Error != nil {
		return WishlistCollaborator, userRecords.Error
	}

	return WishlistCollaborator, nil
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
	WishlistCollaborator := models.WishlistCollaborator{}

	wishlistMembershipRecord := Instance.
		Where(&models.WishlistCollaborator{Enabled: true, WishlistID: WishlistID, UserID: UserID}).
		Joins("JOIN users ON wishlist_collaborators.user_id = users.id").
		Where("users.enabled = ?", true).
		Find(&WishlistCollaborator)

	if wishlistMembershipRecord.Error != nil {
		return verified, wishlistMembershipRecord.Error
	} else if wishlistMembershipRecord.RowsAffected != 1 {
		return verified, err
	}

	return true, err
}

// Set wishlist membership to disabled
func DeleteWishlistCollaboratorByWishlistCollaboratorID(WishlistCollaboratorID uuid.UUID) (err error) {
	wishlistCollaborator := models.WishlistCollaborator{}

	wishlistMembershipRecords := Instance.
		Model(wishlistCollaborator).
		Where(&models.GormModel{ID: WishlistCollaboratorID}).
		Update("enabled", 0)

	if wishlistMembershipRecords.Error != nil {
		return wishlistMembershipRecords.Error
	}
	if wishlistMembershipRecords.RowsAffected != 1 {
		return errors.New("Failed to delete wishlist collaboration in database.")
	}

	return nil
}

// Get all wishlists a user is an owner of
func GetWishlistsByUserIDThroughWishlistCollaborations(UserID uuid.UUID) (wishlists []models.Wishlist, err error) {
	wishlists = []models.Wishlist{}
	err = nil

	// Order("wishlists.date desc, wishlists.name").
	wishlistRecords := Instance.
		Where(&models.Wishlist{Enabled: true}).
		Joins("JOIN wishlist_collaborators ON wishlists.id = wishlist_collaborators.wishlist_id").
		Where("wishlist_collaborators.enabled = ? AND wishlist_collaborators.user_id = ?", true, UserID).
		Joins("JOIN users ON wishlist_collaborators.user_id = users.id").
		Where("users.enabled = ?", true).
		Find(&wishlists)

	if wishlistRecords.Error != nil {
		return []models.Wishlist{}, wishlistRecords.Error
	}

	return wishlists, err
}

// Get all wishlists a user is a member of
func GetWishlistsByUserIDThroughWishlistMemberships(UserID uuid.UUID) (wishlists []models.Wishlist, err error) {
	wishlists = []models.Wishlist{}

	wishlistRecords := Instance.
		Distinct().
		Where(&models.Wishlist{Enabled: true}).
		Joins("JOIN wishlist_memberships ON wishlists.id = wishlist_memberships.wishlist_id").
		Where("wishlist_memberships.enabled = ?", true).
		Joins("JOIN groups ON wishlist_memberships.group_id = groups.id").
		Where("groups.enabled = ?", true).
		Joins("JOIN group_memberships ON groups.id = group_memberships.group_id").
		Where("group_memberships.enabled = ?", true).
		Joins("JOIN users ON group_memberships.member_id = users.id").
		Where("users.enabled = ? AND users.id = ?", true, UserID).
		Order("wishlists.date desc, wishlists.name").
		Find(&wishlists)

	if wishlistRecords.Error != nil {
		return []models.Wishlist{}, wishlistRecords.Error
	}

	return wishlists, err
}

// Get all wishlists in groups
func GetWishlistsFromGroup(GroupID uuid.UUID) ([]models.Wishlist, error) {
	var wishlists []models.Wishlist

	wishlistRecords := Instance.
		Where(&models.Wishlist{Enabled: true}).
		Joins("JOIN wishlist_memberships ON wishlist_memberships.wishlist_id = wishlists.id").
		Where("wishlist_memberships.enabled = ? AND wishlist_memberships.group_id = ?", true, GroupID).
		Joins("JOIN groups ON wishlist_memberships.group_id = groups.ID").
		Where("groups.enabled = ?", true).
		Joins("JOIN users ON wishlists.owner_id = users.id").
		Where("users.enabled = ?", true).
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

	wishlistRecords := Instance.
		Distinct().
		Where(&models.Wishlist{Enabled: true, OwnerID: UserID}).
		Joins("JOIN users ON users.id = wishlists.owner_id").
		Where("users.enabled = ?", true).
		Order("wishlists.date desc, wishlists.name").
		Find(&wishlists)

	if wishlistRecords.Error != nil {
		return []models.Wishlist{}, wishlistRecords.Error
	}

	return wishlists, err
}

// Get all wishlists a user is an owner of
func GetWishlist(WishlistID uuid.UUID) (models.Wishlist, error) {
	var wishlist models.Wishlist

	wishlistRecords := Instance.
		Where(&models.Wishlist{Enabled: true}).
		Where(&models.GormModel{ID: WishlistID}).
		Find(&wishlist)

	if wishlistRecords.Error != nil {
		return models.Wishlist{}, wishlistRecords.Error
	} else if wishlistRecords.RowsAffected != 1 {
		return models.Wishlist{}, errors.New("Wishlist not found.")
	}

	return wishlist, nil
}

// Verify if a wish name in wishlist is unique
func VerifyUniqueWishNameInWishlist(WishName string, WishlistID uuid.UUID) (bool, error) {
	var wish models.Wish

	wishesRecord := Instance.
		Where(&models.Wish{Enabled: true, WishlistID: WishlistID, Name: WishName}).
		Find(&wish)

	if wishesRecord.Error != nil {
		return false, wishesRecord.Error
	} else if wishesRecord.RowsAffected != 0 {
		return false, nil
	}
	return true, nil
}

// Verify if a wishlist name in group is unique
func VerifyUniqueWishlistNameForUser(WishlistName string, UserID uuid.UUID) (bool, error) {
	var wishlist models.Wishlist

	wishlistRecord := Instance.
		Where(&models.Wishlist{Enabled: true, OwnerID: UserID, Name: WishlistName}).
		Find(&wishlist)

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

	wishlistRecord := Instance.
		Where(&models.Wishlist{Enabled: true}).
		Where(&models.GormModel{ID: WishlistID}).
		Find(&wishlist)

	if wishlistRecord.Error != nil {
		return uuid.UUID{}, wishlistRecord.Error
	} else if wishlistRecord.RowsAffected != 1 {
		return uuid.UUID{}, errors.New("Failed to find correct wishlist in DB.")
	}

	return wishlist.OwnerID, nil
}

// Verify if a group ID is a member of a wishlist
func VerifyUserMembershipToGroupMembershipToWishlist(UserID uuid.UUID, WishlistID uuid.UUID) (bool, error) {
	var wishlistMembership models.WishlistMembership

	wishlistMembershipRecord := Instance.
		Where(&models.WishlistMembership{WishlistID: WishlistID, Enabled: true}).
		Joins("JOIN groups ON groups.id = wishlist_memberships.group_id").
		Where("groups.enabled = ?", true).
		Joins("JOIN group_memberships ON group_memberships.group_id = groups.id").
		Where("group_memberships.enabled = ? AND group_memberships.member_id = ?", true, UserID).
		Find(&wishlistMembership)

	if wishlistMembershipRecord.Error != nil {
		return false, wishlistMembershipRecord.Error
	} else if wishlistMembershipRecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Verify if a user ID is an owner of a wishlist
func VerifyUserOwnershipToWishlist(UserID uuid.UUID, WishlistID uuid.UUID) (bool, error) {
	var wishlist models.Wishlist

	wishlistRecord := Instance.
		Where(&models.Wishlist{Enabled: true, OwnerID: UserID}).
		Where(&models.GormModel{ID: WishlistID}).
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
	var groupMemberships []models.GroupMembership

	membershipRecords := Instance.
		Where(&models.GroupMembership{Enabled: true}).
		Joins("JOIN groups ON group_memberships.group_id = groups.id").
		Where("groups.enabled = ?", true).
		Joins("JOIN wishlist_memberships ON wishlist_memberships.group_id = groups.id").
		Where("group_memberships.enabled = ?", true).
		Joins("JOIN wishlists ON wishlists.id = wishlist_memberships.wishlist_id").
		Where("wishlists.enabled = ? AND wishlists.id = ?", true, WishlistID).
		Joins("JOIN users ON group_memberships.group_id = users.id").
		Where("users.enabled = ?", true).
		Where("group_memberships.group_id != wishlists.owner_id").
		Find(&groupMemberships)

	if membershipRecords.Error != nil {
		return []models.User{}, membershipRecords.Error
	}

	for _, membership := range groupMemberships {
		userObject, err := GetUserInformation(membership.MemberID)
		if err != nil {
			return []models.User{}, err
		}
		users = append(users, userObject)
	}

	if len(users) == 0 {
		users = []models.User{}
	}

	return users, nil
}

func GetMembershipIDForGroupToWishlist(WishlistID uuid.UUID, GroupID uuid.UUID) (membershipFound bool, wishlistMembership models.WishlistMembership, err error) {
	wishlistMembership = models.WishlistMembership{}

	wishlistMembershipRecord := Instance.
		Where(&models.WishlistMembership{WishlistID: WishlistID, Enabled: true, GroupID: GroupID}).
		Find(&wishlistMembership)

	if wishlistMembershipRecord.Error != nil {
		return false, wishlistMembership, wishlistMembershipRecord.Error
	} else if wishlistMembershipRecord.RowsAffected != 1 {
		return false, wishlistMembership, errors.New("Failed to find membership.")
	}

	return true, wishlistMembership, err
}

// Get wishlist by wishlist ID
func GetPublicWishListByWishlistHash(wishlistHash uuid.UUID) (bool, models.Wishlist, error) {
	var wishlist models.Wishlist

	wishlistRecord := Instance.
		Where(&models.Wishlist{Enabled: true, PublicHash: wishlistHash, Public: &utilities.DBTrue}).
		Find(&wishlist)

	if wishlistRecord.Error != nil {
		return false, models.Wishlist{}, wishlistRecord.Error
	} else if wishlistRecord.RowsAffected != 1 {
		return false, models.Wishlist{}, nil
	}

	return true, wishlist, nil
}

// Create wishlist in DB
func CreateWishlistMembershipInDB(wishlistMembership models.WishlistMembership) (newWishlistMembership models.WishlistMembership, err error) {
	record := Instance.Create(&wishlistMembership)

	if record.Error != nil {
		return wishlistMembership, record.Error
	}

	if record.RowsAffected != 1 {
		return wishlistMembership, errors.New("Wishlist not added to database.")
	}

	return wishlistMembership, err
}
