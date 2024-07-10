package database

import (
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Update values in wishlist object in DB
func UpdateWishlistValuesByID(wishlistID uuid.UUID, wishlistName string, wishlistDesc string, wishlistExpiration time.Time, wishlistClaimable bool, wishlistExpires bool, wishlistPublic bool, wishlistPublicHash uuid.UUID) error {

	var wishlist models.Wishlist

	wishlistRecord := Instance.Model(wishlist).Where(&models.Wishlist{GormModel: models.GormModel{ID: wishlistID}, Enabled: true}).Update("name", wishlistName)
	if wishlistRecord.Error != nil {
		return wishlistRecord.Error
	}
	if wishlistRecord.RowsAffected != 1 {
		return errors.New("Name not changed in database.")
	}

	wishlistRecord = Instance.Model(wishlist).Where(&models.Wishlist{Enabled: true}).Where(&models.GormModel{ID: wishlistID}).Update("description", wishlistDesc)
	if wishlistRecord.Error != nil {
		return wishlistRecord.Error
	}
	if wishlistRecord.RowsAffected != 1 {
		return errors.New("Description not changed in database.")
	}

	wishlistRecord = Instance.Model(wishlist).Where(&models.Wishlist{Enabled: true}).Where(&models.GormModel{ID: wishlistID}).Update("date", wishlistExpiration)
	if wishlistRecord.Error != nil {
		return wishlistRecord.Error
	}
	if wishlistRecord.RowsAffected != 1 {
		return errors.New("Expiration not changed in database.")
	}

	wishlistRecord = Instance.Model(wishlist).Where(&models.Wishlist{Enabled: true}).Where(&models.GormModel{ID: wishlistID}).Update("claimable", wishlistClaimable)
	if wishlistRecord.Error != nil {
		return wishlistRecord.Error
	}
	if wishlistRecord.RowsAffected != 1 {
		return errors.New("Claimability not changed in database.")
	}

	wishlistRecord = Instance.Model(wishlist).Where(&models.Wishlist{Enabled: true}).Where(&models.GormModel{ID: wishlistID}).Update("expires", wishlistExpires)
	if wishlistRecord.Error != nil {
		return wishlistRecord.Error
	}
	if wishlistRecord.RowsAffected != 1 {
		return errors.New("Expiration not changed in database.")
	}

	wishlistRecord = Instance.Model(wishlist).Where(&models.Wishlist{Enabled: true}).Where(&models.GormModel{ID: wishlistID}).Update("public", wishlistPublic)
	if wishlistRecord.Error != nil {
		return wishlistRecord.Error
	}
	if wishlistRecord.RowsAffected != 1 {
		return errors.New("Public state not changed in database.")
	}

	wishlistRecord = Instance.Model(wishlist).Where(&models.Wishlist{Enabled: true}).Where(&models.GormModel{ID: wishlistID}).Update("public_hash", wishlistPublicHash)
	if wishlistRecord.Error != nil {
		return wishlistRecord.Error
	}
	if wishlistRecord.RowsAffected != 1 {
		return errors.New("Public hash not changed in database.")
	}

	return nil

}

// Create wishlist in DB
func CreateWishlistInDB(wishlistdb models.Wishlist) (wishlist models.Wishlist, err error) {
	wishlist = models.Wishlist{}
	err = nil
	record := Instance.Create(&wishlistdb)

	if record.Error != nil {
		return wishlistdb, record.Error
	}

	if record.RowsAffected != 1 {
		return wishlistdb, errors.New("Wishlist not added to database.")
	}

	return wishlistdb, err
}

// Get wishlist by wishlist ID
func GetWishlistByWishlistID(wishlistID uuid.UUID) (bool, models.Wishlist, error) {

	var wishlist models.Wishlist

	wishlistRecord := Instance.Where(&models.Wishlist{Enabled: true}).Where(&models.GormModel{ID: wishlistID}).Find(&wishlist)

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

	userRecords := Instance.
		Where(&models.WishlistCollaborator{Enabled: true}).
		Joins("JOIN users ON wishlist_collaborators.user_id = users.id", Instance.Where(&models.User{Enabled: true})).
		Joins("JOIN wishlists ON wishlist_collaborators.wishlist_id = wishlists.id", Instance.Where(&models.Wishlist{Enabled: true}).Where(&models.GormModel{ID: WishlistID})).
		Find(&wishlistColab)
	if userRecords.Error != nil {
		return wishlistColab, userRecords.Error
	}

	return wishlistColab, nil
}

// Get wishlist collab by id
func GetWishlistCollaboratorByUserIDAndWishlistID(WishlistID uuid.UUID, UserID uuid.UUID) (wishlistColab models.WishlistCollaborator, err error) {
	wishlistColab = models.WishlistCollaborator{}

	userRecords := Instance.
		Where(&models.WishlistCollaborator{Enabled: true}).
		Joins("JOIN users ON wishlist_collaborators.user_id = users.id", Instance.Where(&models.User{Enabled: true}).Where(&models.GormModel{ID: UserID})).
		Joins("JOIN wishlists ON wishlist_collaborators.wishlist_id = wishlists.id", Instance.Where(&models.Wishlist{Enabled: true}).Where(&models.GormModel{ID: WishlistID})).
		Find(&wishlistColab)
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

	wishlistmembershipprecord := Instance.
		Where(&models.WishlistCollaborator{Enabled: true, WishlistID: WishlistID, UserID: UserID}).
		Joins("JOIN users ON wishlist_collaborators.user_id = users.id", Instance.Where(&models.User{Enabled: true})).
		Find(&wishlistColab)
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

	wishlistmembershiprecords := Instance.Model(wishlistCollaborator).Where(&models.GormModel{ID: WishlistCollaboratorID}).Update("enabled", 0)
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

	// Order("wishlists.date desc, wishlists.name").
	wishlistRecords := Instance.
		Where(&models.Wishlist{Enabled: true}).
		Joins("JOIN wishlist_collaborators ON wishlists.id = wishlist_collaborators.wishlist_id", Instance.Where(&models.WishlistCollaborator{Enabled: true, UserID: UserID})).
		Joins("JOIN users ON wishlist_collaborators.user_id = users.id", Instance.Where(&models.User{Enabled: true})).
		Find(&wishlists)

	if wishlistRecords.Error != nil {
		return []models.Wishlist{}, wishlistRecords.Error
	}

	return wishlists, err
}

// Get all wishlists in groups
func GetWishlistsFromGroup(GroupID uuid.UUID) ([]models.Wishlist, error) {
	var wishlists []models.Wishlist
	wishlistrecords := Instance.
		Where(&models.Wishlist{Enabled: true}).
		Joins("JOIN wishlist_memberships ON wishlist_memberships.wishlist_id = wishlists.id", Instance.Where(&models.WishlistMembership{Enabled: true, GroupID: GroupID})).
		Joins("JOIN groups ON wishlist_memberships.group_id = groups.ID", Instance.Where(&models.Group{Enabled: true})).
		Joins("JOIN users ON wishlists.owner_id = users.id", Instance.Where(&models.User{Enabled: true})).
		Find(&wishlists)

	if wishlistrecords.Error != nil {
		return []models.Wishlist{}, wishlistrecords.Error
	} else if wishlistrecords.RowsAffected == 0 {
		return []models.Wishlist{}, nil
	}

	return wishlists, nil
}

// Get all wishlists a user is an owner of
func GetOwnedWishlists(UserID uuid.UUID) (wishlists []models.Wishlist, err error) {
	wishlists = []models.Wishlist{}
	err = nil

	wishlistrecords := Instance.
		Where(&models.Wishlist{Enabled: true, OwnerID: UserID}).
		Joins("JOIN users ON users.id = wishlists.owner_id", Instance.Where(&models.User{Enabled: true})).
		Order("wishlists.date desc, wishlists.name").
		Find(&wishlists)

	if wishlistrecords.Error != nil {
		return []models.Wishlist{}, wishlistrecords.Error
	}

	return wishlists, err
}

// Get all wishlists a user is an owner of
func GetWishlist(WishlistID uuid.UUID) (models.Wishlist, error) {
	var wishlist models.Wishlist
	wishlistrecords := Instance.Where(&models.Wishlist{Enabled: true}).Where(&models.GormModel{ID: WishlistID}).Find(&wishlist)

	if wishlistrecords.Error != nil {
		return models.Wishlist{}, wishlistrecords.Error
	} else if wishlistrecords.RowsAffected != 1 {
		return models.Wishlist{}, errors.New("Wishlist not found.")
	}

	return wishlist, nil
}

// Verify if a wish name in wishlist is unique
func VerifyUniqueWishNameinWishlist(WishName string, WishlistID uuid.UUID) (bool, error) {
	var wish models.Wish
	wishesrecord := Instance.Where(&models.Wish{Enabled: true, WishlistID: WishlistID, Name: WishName}).Find(&wish)
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
	wishlistrecord := Instance.Where(&models.Wishlist{Enabled: true, OwnerID: UserID, Name: WishlistName}).Find(&wishlist)
	if wishlistrecord.Error != nil {
		return false, wishlistrecord.Error
	} else if wishlistrecord.RowsAffected != 0 {
		return false, nil
	}
	return true, nil
}

// Get owner id of wishlist
func GetWishlistOwner(WishlistID uuid.UUID) (uuid.UUID, error) {
	var wishlist models.Wishlist
	wishlistrecord := Instance.Where(&models.Wishlist{Enabled: true}).Where(&models.GormModel{ID: WishlistID}).Find(&wishlist)
	if wishlistrecord.Error != nil {
		return uuid.UUID{}, wishlistrecord.Error
	} else if wishlistrecord.RowsAffected != 1 {
		return uuid.UUID{}, errors.New("Failed to find correct wishlist in DB.")
	}

	return wishlist.OwnerID, nil
}

// Verify if a group ID is a member of a wishlist
func VerifyUserMembershipToGroupmembershipToWishlist(UserID uuid.UUID, WishlistID uuid.UUID) (bool, error) {
	var wishlistmembership models.WishlistMembership
	wishlistmembershiprecord := Instance.
		Where(&models.WishlistMembership{WishlistID: WishlistID, Enabled: true}).
		Joins("JOIN groups ON groups.id = wishlist_memberships.group_id", Instance.Where(&models.Group{Enabled: true})).
		Joins("JOIN group_memberships ON group_memberships.group_id = groups.id", Instance.Where(&models.GroupMembership{Enabled: true, MemberID: UserID})).
		Find(&wishlistmembership)
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
	wishlistrecord := Instance.Where(&models.Wishlist{Enabled: true, OwnerID: UserID}).Where(&models.GormModel{ID: WishlistID}).Find(&wishlist)
	if wishlistrecord.Error != nil {
		return false, wishlistrecord.Error
	} else if wishlistrecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Get user information from wishlist
func GetUserMembersFromWishlist(WishlistID uuid.UUID) ([]models.User, error) {
	var users []models.User
	var group_memberships []models.GroupMembership

	membershiprecords := Instance.
		Where(&models.GroupMembership{Enabled: true}).
		Joins("JOIN groups ON group_memberships.group_id = groups.id", Instance.Where(&models.Group{Enabled: true})).
		Joins("JOIN wishlist_memberships ON wishlist_memberships.group_id = groups.id", Instance.Where(&models.GroupMembership{Enabled: true})).
		Joins("JOIN wishlists ON wishlists.id = wishlist_memberships.wishlist_id", Instance.Where(&models.Wishlist{Enabled: true}).Where(&models.GormModel{ID: WishlistID})).
		Joins("JOIN users ON group_memberships.group_id = users.id", Instance.Where(&models.User{Enabled: true})).
		Where("group_memberships.group_id != wishlists.owner_id").
		Find(&group_memberships)
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

	wishlistmembershiprecord := Instance.Where(&models.WishlistMembership{WishlistID: WishlistID, Enabled: true, GroupID: GroupID}).Find(&wishlistMembership)
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

	wishlistRecord := Instance.Where(&models.Wishlist{Enabled: true, PublicHash: wishlistHash, Public: &utilities.DBTrue}).Find(&wishlist)

	if wishlistRecord.Error != nil {
		return false, models.Wishlist{}, wishlistRecord.Error
	} else if wishlistRecord.RowsAffected != 1 {
		return false, models.Wishlist{}, nil
	}

	return true, wishlist, nil
}
