package database

import (
	"errors"
	"log"
	"poenskelisten/models"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var Instance *gorm.DB
var dbError error

func Connect(connectionString string) {
	Instance, dbError = gorm.Open(mysql.Open(connectionString), &gorm.Config{})
	if dbError != nil {
		log.Fatal(dbError)
		panic("Cannot connect to DB")
	}
	log.Println("Connected to Database!")
}

func Migrate() {
	Instance.AutoMigrate(&models.User{})
	Instance.AutoMigrate(&models.Invite{})
	Instance.AutoMigrate(&models.Group{})
	Instance.AutoMigrate(&models.GroupMembership{})
	Instance.AutoMigrate(&models.Wishlist{})
	Instance.AutoMigrate(&models.WishlistMembership{})
	Instance.AutoMigrate(&models.Wish{})
	log.Println("Database Migration Completed!")
}

// Verify e-mail is not in use
func VerifyUniqueUserEmail(providedEmail string) (bool, error) {
	var user models.User
	userrecords := Instance.Where("`users`.email= ?", providedEmail).Find(&user)
	if userrecords.Error != nil {
		return false, userrecords.Error
	}
	if userrecords.RowsAffected != 0 {
		return false, nil
	}
	return true, nil
}

// Verify unsued invite code exists
func VerifyUnusedUserInviteCode(providedCode string) (bool, error) {
	var invitestruct models.Invite
	inviterecords := Instance.Where("`invites`.invite_enabled = ?", 1).Where("`invites`.invite_used= ?", 0).Where("`invites`.invite_code = ?", providedCode).Find(&invitestruct)
	if inviterecords.Error != nil {
		return false, inviterecords.Error
	}
	if inviterecords.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Set invite code to used
func SetUsedUserInviteCode(providedCode string) error {
	var invitestruct models.Invite
	inviterecords := Instance.Model(invitestruct).Where("`invites`.invite_code= ?", providedCode).Update("invite_used", 1)
	if inviterecords.Error != nil {
		return inviterecords.Error
	}
	if inviterecords.RowsAffected != 1 {
		return errors.New("Code not changed in database.")
	}
	return nil
}

// Set group to disabled
func DeleteGroup(GroupID int) error {
	var group models.Group
	grouprecords := Instance.Model(group).Where("`groups`.ID= ?", GroupID).Update("enabled", 0)
	if grouprecords.Error != nil {
		return grouprecords.Error
	}
	if grouprecords.RowsAffected != 1 {
		return errors.New("Failed to delete group in database.")
	}
	return nil
}

// Set group membership to disabled
func DeleteGroupMembership(GroupMembershipID int) error {
	var groupmembership models.GroupMembership
	grouprecords := Instance.Model(groupmembership).Where("`group_memberships`.ID= ?", GroupMembershipID).Update("enabled", 0)
	if grouprecords.Error != nil {
		return grouprecords.Error
	}
	if grouprecords.RowsAffected != 1 {
		return errors.New("Failed to delete group membership in database.")
	}
	return nil
}

// Set wishlist to disabled
func DeleteWishlist(WishlistID int) error {
	var wishlist models.Wishlist
	wishlistrecords := Instance.Model(wishlist).Where("`wishlists`.ID= ?", WishlistID).Update("enabled", 0)
	if wishlistrecords.Error != nil {
		return wishlistrecords.Error
	}
	if wishlistrecords.RowsAffected != 1 {
		return errors.New("Failed to delete wishlist in database.")
	}
	return nil
}

// Set wish to disabled
func DeleteWish(WishID int) error {
	var wish models.Wish
	wishrecords := Instance.Model(wish).Where("`wishes`.ID= ?", WishID).Update("enabled", 0)
	if wishrecords.Error != nil {
		return wishrecords.Error
	}
	if wishrecords.RowsAffected != 1 {
		return errors.New("Failed to delete wish in database.")
	}
	return nil
}

// Verify if a user ID is a member of a group
func VerifyUserMembershipToGroup(UserID int, GroupID int) (bool, error) {
	var groupmembership models.GroupMembership
	groupmembershiprecord := Instance.Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.group = ?", GroupID).Where("`group_memberships`.member = ?", UserID).Find(&groupmembership)
	if groupmembershiprecord.Error != nil {
		return false, groupmembershiprecord.Error
	} else if groupmembershiprecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Verify if a group ID is a member of a wishlist
func VerifyGroupmembershipToWishlist(GroupID int, WishlistID int) (bool, error) {
	var wishlistmembership models.WishlistMembership
	wishlistmembershiprecord := Instance.Where("`wishlist_memberships`.enabled = ?", 1).Where("`wishlist_memberships`.group = ?", GroupID).Where("`wishlist_memberships`.wishlist = ?", WishlistID).Find(&wishlistmembership)
	if wishlistmembershiprecord.Error != nil {
		return false, wishlistmembershiprecord.Error
	} else if wishlistmembershiprecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Verify if a group ID is a member of a wishlist
func VerifyUserMembershipToGroupmembershipToWishlist(UserID int, WishlistID int) (bool, error) {
	var wishlistmembership models.WishlistMembership
	wishlistmembershiprecord := Instance.Where("`wishlist_memberships`.enabled = ?", 1).Where("`wishlist_memberships`.wishlist = ?", WishlistID).Joins("JOIN `groups` on `groups`.id = `wishlist_memberships`.group").Where("`groups`.enabled = ?", 1).Joins("JOIN `group_memberships` on `group_memberships`.group = `groups`.id").Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.member = ?", UserID).Find(&wishlistmembership)
	if wishlistmembershiprecord.Error != nil {
		return false, wishlistmembershiprecord.Error
	} else if wishlistmembershiprecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Verify if a user ID is an owner of a group
func VerifyUserOwnershipToGroup(UserID int, GroupID int) (bool, error) {
	var group models.Group
	grouprecord := Instance.Where("`groups`.enabled = ?", 1).Where("`groups`.id = ?", GroupID).Where("`groups`.owner = ?", UserID).Find(&group)
	if grouprecord.Error != nil {
		return false, grouprecord.Error
	} else if grouprecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Verify if a user ID is an owner of a wishlist
func VerifyUserOwnershipToWishlist(UserID int, WishlistID int) (bool, error) {
	var wishlist models.Wishlist
	wishlistrecord := Instance.Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.id = ?", WishlistID).Where("`wishlists`.owner = ?", UserID).Find(&wishlist)
	if wishlistrecord.Error != nil {
		return false, wishlistrecord.Error
	} else if wishlistrecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Verify if a wish name in wishlist is unique
func VerifyUniqueWishNameinWishlist(WishName string, WishlistID int) (bool, error) {
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
func VerifyUniqueWishlistNameForUser(WishlistName string, UserID int) (bool, error) {
	var wishlist models.Wishlist
	wishlistrecord := Instance.Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.owner = ?", UserID).Where("`wishlists`.name = ?", WishlistName).Find(&wishlist)
	if wishlistrecord.Error != nil {
		return false, wishlistrecord.Error
	} else if wishlistrecord.RowsAffected != 0 {
		return false, nil
	}
	return true, nil
}

// Get user information
func GetUserInformation(UserID int) (models.User, error) {
	var user models.User
	userrecord := Instance.Where("`users`.enabled = ?", 1).Where("`users`.id = ?", UserID).Find(&user)
	if userrecord.Error != nil {
		return models.User{}, userrecord.Error
	} else if userrecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

	// Redact user information
	user.Password = "REDACTED"
	user.Email = "REDACTED"

	return user, nil
}

// Get owner id of wishlist
func GetWishlistOwner(WishlistID int) (int, error) {
	var wishlist models.Wishlist
	wishlistrecord := Instance.Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.id = ?", WishlistID).Find(&wishlist)
	if wishlistrecord.Error != nil {
		return 0, wishlistrecord.Error
	} else if wishlistrecord.RowsAffected != 1 {
		return 0, errors.New("Failed to find correct wishlist in DB.")
	}

	return wishlist.Owner, nil
}

// Get user information
func GetUserMembersFromWishlist(WishlistID int) ([]models.User, error) {
	var users []models.User
	var group_memberships []models.GroupMembership

	membershiprecords := Instance.Where("`group_memberships`.enabled = ?", 1).Joins("JOIN `groups` on `group_memberships`.group = `groups`.id").Where("`groups`.enabled = ?", 1).Joins("JOIN `wishlist_memberships` on `wishlist_memberships`.group = `groups`.id").Where("`wishlist_memberships`.enabled = ?", 1).Joins("JOIN `wishlists` on `wishlists`.id = `wishlist_memberships`.wishlist").Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.id = ?", WishlistID).Joins("JOIN `users` on `group_memberships`.member = `users`.id").Where("`users`.enabled = ?", 1).Where("`group_memberships`.member != `wishlists`.owner").Find(&group_memberships)
	if membershiprecords.Error != nil {
		return []models.User{}, membershiprecords.Error
	}

	for _, membership := range group_memberships {
		user_object, err := GetUserInformation(membership.Member)
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

// Get all wishlists in groups
func GetWishlistsFromGroup(GroupID int) ([]models.Wishlist, error) {
	var wishlists []models.Wishlist
	wishlistrecords := Instance.Where("`wishlists`.enabled = ?", 1).Joins("JOIN wishlist_memberships on wishlist_memberships.wishlist = wishlists.id").Where("`wishlist_memberships`.group = ?", GroupID).Where("`wishlist_memberships`.enabled = ?", 1).Find(&wishlists)

	if wishlistrecords.Error != nil {
		return []models.Wishlist{}, wishlistrecords.Error
	} else if wishlistrecords.RowsAffected == 0 {
		return []models.Wishlist{}, nil
	}

	return wishlists, nil
}

// Get all wishlists a user is an owner of
func GetOwnedWishlists(UserID int) ([]models.Wishlist, error) {
	var wishlists []models.Wishlist
	wishlistrecords := Instance.Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.owner = ?", UserID).Joins("JOIN users on users.id = wishlists.owner").Where("`users`.enabled = ?", 1).Find(&wishlists)

	if wishlistrecords.Error != nil {
		return []models.Wishlist{}, wishlistrecords.Error
	} else if wishlistrecords.RowsAffected == 0 {
		return []models.Wishlist{}, nil
	}

	return wishlists, nil
}

// Get all wishlists a user is an owner of
func GetWishlist(WishlistID int) (models.Wishlist, error) {
	var wishlist models.Wishlist
	wishlistrecords := Instance.Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.id = ?", WishlistID).Find(&wishlist)

	if wishlistrecords.Error != nil {
		return models.Wishlist{}, wishlistrecords.Error
	} else if wishlistrecords.RowsAffected != 1 {
		return models.Wishlist{}, errors.New("Wishlist not found.")
	}

	return wishlist, nil
}

// Get user information
func GetWishesFromWishlist(WishlistID int) ([]models.WishUser, error) {
	var wishes []models.Wish
	var wishes_with_owner []models.WishUser

	wishrecords := Instance.Where("`wishes`.enabled = ?", 1).Where("`wishes`.wishlist_id = ?", WishlistID).Find(&wishes)
	if wishrecords.Error != nil {
		return []models.WishUser{}, wishrecords.Error
	} else if wishrecords.RowsAffected < 1 {
		return []models.WishUser{}, nil
	}

	for _, wish := range wishes {
		user_object, err := GetUserInformation(wish.Owner)
		if err != nil {
			return []models.WishUser{}, err
		}

		var wish_with_owner models.WishUser
		wish_with_owner.CreatedAt = wish.CreatedAt
		wish_with_owner.DeletedAt = wish.DeletedAt
		wish_with_owner.Enabled = wish.Enabled
		wish_with_owner.ID = wish.ID
		wish_with_owner.Model = wish.Model
		wish_with_owner.Name = wish.Name
		wish_with_owner.Note = wish.Note
		wish_with_owner.Owner = user_object
		wish_with_owner.URL = wish.URL
		wish_with_owner.UpdatedAt = wish.UpdatedAt
		wish_with_owner.WishlistID = wish.WishlistID

		wishes_with_owner = append(wishes_with_owner, wish_with_owner)
	}

	return wishes_with_owner, nil
}

// get wishlist id from wish id
func GetWishlistFromWish(WishID int) (int, error) {
	var wish models.Wish
	wishrecord := Instance.Where("`wishes`.enabled = ?", 1).Where("`wishes`.id = ?", WishID).Find(&wish)
	if wishrecord.Error != nil {
		return 0, wishrecord.Error
	} else if wishrecord.RowsAffected != 1 {
		return 0, errors.New("Failed to find correct wish in DB.")
	}

	return wish.WishlistID, nil
}
