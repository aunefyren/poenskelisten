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
func VerifyUniqueWishlistNameinGroup(WishlistName string, GroupID int) (bool, error) {
	var wishlist models.Wishlist
	wishlistrecord := Instance.Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.group = ?", GroupID).Where("`wishlists`.name = ?", WishlistName).Find(&wishlist)
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
func GetUserMembersFromGroup(GroupID int) ([]models.User, error) {
	var users []models.User
	var group_memberships []models.GroupMembership

	membershiprecords := Instance.Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.group = ?", GroupID).Find(&group_memberships)
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

	return users, nil
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
