package database

import (
	"aunefyren/poenskelisten/models"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/thanhpk/randstr"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var Instance *gorm.DB
var dbError error

func Connect(dbType string, timezone string, dbUsername string, dbPassword string, dbIP string, dbPort int, dbName string) error {

	if strings.ToLower(dbType) == "postgresql" {
		return errors.New("Database type not supported.")
	} else if strings.ToLower(dbType) == "mysql" {
		connStrDb := dbUsername + ":" + dbPassword + "@tcp(" + dbIP + ":" + strconv.Itoa(dbPort) + ")/" + dbName + "?parseTime=True&loc=Local&charset=utf8mb4"

		// Connect to DB without DB Name
		Instance, dbError = gorm.Open(mysql.Open(connStrDb), &gorm.Config{})
		if dbError != nil {

			if strings.Contains(dbError.Error(), "Unknown database '"+dbName+"'") {
				err := CreateTable(dbUsername, dbPassword, dbIP, dbPort, dbName)
				if err != nil {
					return err
				} else {
					Instance, dbError = gorm.Open(mysql.Open(connStrDb), &gorm.Config{})
					if dbError != nil {
						return dbError
					}
				}
			} else {
				return dbError
			}
		}
	} else {
		return errors.New("Database type not recognized.")
	}

	log.Println("Connected to database.")
	fmt.Println("Connected to database.")

	return nil
}

func CreateTable(dbUsername string, dbPassword string, dbIP string, dbPort int, dbName string) error {
	url := fmt.Sprintf("host=%s port=%s user=%s password=%s sslmode=disable TimeZone=%s", dbIP, strconv.Itoa(dbPort), dbUsername, dbUsername, "local")
	db, err := sql.Open("mysql", url)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s;", dbName))
	if err != nil {
		panic(err)
	}

	return nil
}

func Migrate() {
	Instance.AutoMigrate(&models.User{})
	Instance.AutoMigrate(&models.Invite{})
	Instance.AutoMigrate(&models.Group{})
	Instance.AutoMigrate(&models.GroupMembership{})
	Instance.AutoMigrate(&models.Wishlist{})
	Instance.AutoMigrate(&models.WishlistMembership{})
	Instance.AutoMigrate(&models.WishlistCollaborator{})
	Instance.AutoMigrate(&models.Wish{})
	Instance.AutoMigrate(&models.WishClaim{})
	Instance.AutoMigrate(&models.News{})
	log.Println("Database Migration Completed!")
}

// Genrate a random invite code an return ut
func GenrateRandomInvite() (string, error) {
	var invite models.Invite

	randomString := randstr.String(16)
	invite.InviteCode = strings.ToUpper(randomString)

	record := Instance.Create(&invite)
	if record.Error != nil {
		return "", record.Error
	}

	return invite.InviteCode, nil
}

// Genrate a random verification code an return ut
func GenrateRandomVerificationCodeForuser(userID int) (string, error) {

	randomString := randstr.String(8)
	verificationCode := strings.ToUpper(randomString)

	var user models.User
	userrecord := Instance.Model(user).Where("`users`.enabled = ?", 1).Where("`users`.ID = ?", userID).Update("verification_code", verificationCode)
	if userrecord.Error != nil {
		return "", userrecord.Error
	}
	if userrecord.RowsAffected != 1 {
		return "", errors.New("Verification code not changed in database.")
	}

	return verificationCode, nil

}

// Verify e-mail is not in use
func VerifyUniqueUserEmail(providedEmail string) (bool, error) {
	var user models.User
	userrecords := Instance.Where("`users`.enabled = ?", 1).Where("`users`.email= ?", providedEmail).Find(&user)
	if userrecords.Error != nil {
		return false, userrecords.Error
	}
	if userrecords.RowsAffected != 0 {
		return false, nil
	}
	return true, nil
}

// Verify if user has a verification code set
func VerifyUserHasVerfificationCode(userID int) (bool, error) {
	var user models.User
	userrecords := Instance.Where("`users`.enabled = ?", 1).Where("`users`.ID= ?", userID).Find(&user)
	if userrecords.Error != nil {
		return false, userrecords.Error
	}
	if userrecords.RowsAffected != 1 {
		return false, errors.New("Couldn't find the user.")
	}

	if user.VerificationCode == "" {
		return false, nil
	} else {
		return true, nil
	}
}

// Verify if user has a verification code set
func VerifyUserVerfificationCodeMatches(userID int, verificationCode string) (bool, error) {

	var user models.User

	userrecords := Instance.Where("`users`.enabled = ?", 1).Where("`users`.ID= ?", userID).Where("`users`.verification_code = ?", verificationCode).Find(&user)

	if userrecords.Error != nil {
		return false, userrecords.Error
	}

	if userrecords.RowsAffected != 1 {
		return false, nil
	} else {
		return true, nil
	}

}

// Verify if user is verified
func VerifyUserIsVerified(userID int) (bool, error) {

	var user models.User
	userrecords := Instance.Where("`users`.id= ?", userID).Find(&user)
	if userrecords.Error != nil {
		return false, userrecords.Error
	}
	if userrecords.RowsAffected != 1 {
		return false, errors.New("No user found.")
	}

	return *user.Verified, nil
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
func SetUsedUserInviteCode(providedCode string, userIDClaimer int) error {
	var invitestruct models.Invite
	inviterecords := Instance.Model(invitestruct).Where("`invites`.invite_code= ?", providedCode).Update("invite_used", 1)
	if inviterecords.Error != nil {
		return inviterecords.Error
	}
	if inviterecords.RowsAffected != 1 {
		return errors.New("Code not changed in database.")
	}

	inviterecords = Instance.Model(invitestruct).Where("`invites`.invite_code= ?", providedCode).Update("invite_recipient", userIDClaimer)
	if inviterecords.Error != nil {
		return inviterecords.Error
	}
	if inviterecords.RowsAffected != 1 {
		return errors.New("Recipient not changed in database.")
	}

	return nil
}

// Set user to verified
func SetUserVerification(userID int, verified bool) error {

	var user models.User
	var verInt int

	if verified {
		verInt = 1
	} else {
		verInt = 0
	}

	userrecords := Instance.Model(user).Where("`users`.enabled = ?", 1).Where("`users`.ID = ?", userID).Update("verified", verInt)
	if userrecords.Error != nil {
		return userrecords.Error
	}
	if userrecords.RowsAffected != 1 {
		return errors.New("Verification not changed in database.")
	}

	return nil
}

// Update user values
func UpdateUserValuesByUserID(userID int, email string, password string) error {

	var user models.User

	userrecords := Instance.Model(user).Where("`users`.enabled = ?", 1).Where("`users`.ID = ?", userID).Update("email", email)
	if userrecords.Error != nil {
		return userrecords.Error
	}
	if userrecords.RowsAffected != 1 {
		return errors.New("Email not changed in database.")
	}

	userrecords = Instance.Model(user).Where("`users`.enabled = ?", 1).Where("`users`.ID = ?", userID).Update("password", password)
	if userrecords.Error != nil {
		return userrecords.Error
	}
	if userrecords.RowsAffected != 1 {
		return errors.New("Password not changed in database.")
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

// Set wishlist membership to disabled
func DeleteWishlistMembership(WishlistMembershipID int) error {
	var wishlistmembership models.WishlistMembership
	wishlistmembershiprecords := Instance.Model(wishlistmembership).Where("`wishlist_memberships`.ID= ?", WishlistMembershipID).Update("enabled", 0)
	if wishlistmembershiprecords.Error != nil {
		return wishlistmembershiprecords.Error
	}
	if wishlistmembershiprecords.RowsAffected != 1 {
		return errors.New("Failed to delete wishlist membership in database.")
	}
	return nil
}

// Set wish claim to disabled
func DeleteWishClaimByUserAndWish(WishID int, UserID int) error {
	var wishclaim models.WishClaim
	wishclaimrecords := Instance.Model(wishclaim).Where("`wish_claims`.wish= ?", WishID).Where("`wish_claims`.user= ?", UserID).Update("enabled", 0)
	if wishclaimrecords.Error != nil {
		return wishclaimrecords.Error
	}
	if wishclaimrecords.RowsAffected != 1 {
		return errors.New("Failed to delete wish claim membership in database.")
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

// Verify if a group id is a member of a wishlist
func VerifyGroupMembershipToWishlist(WishlistID int, GroupID int) (bool, error) {
	var wishlistmembership models.WishlistMembership
	wishlistmembershipprecord := Instance.Where("`wishlist_memberships`.enabled = ?", 1).Where("`wishlist_memberships`.wishlist = ?", WishlistID).Where("`wishlist_memberships`.group = ?", GroupID).Find(&wishlistmembership)
	if wishlistmembershipprecord.Error != nil {
		return false, wishlistmembershipprecord.Error
	} else if wishlistmembershipprecord.RowsAffected != 1 {
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

// Verify if a user ID is an owner of a wish
func VerifyUserOwnershipToWish(UserID int, WishID int) (bool, error) {
	var wish models.Wish
	wishrecord := Instance.Where("`wishes`.enabled = ?", 1).Where("`wishes`.id = ?", WishID).Where("`wishes`.owner = ?", UserID).Find(&wish)
	if wishrecord.Error != nil {
		return false, wishrecord.Error
	} else if wishrecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Verify if a user ID is an owner of a wish
func VerifyUserOwnershipToWishClaimByWish(UserID int, WishID int) (bool, error) {
	var wishclaim models.WishClaim
	wishclaimrecord := Instance.Where("`wish_claims`.enabled = ?", 1).Where("`wish_claims`.wish = ?", WishID).Where("`wish_claims`.user = ?", UserID).Find(&wishclaim)
	if wishclaimrecord.Error != nil {
		return false, wishclaimrecord.Error
	} else if wishclaimrecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Verify if a user ID is an owner of a wish
func VerifyWishIsClaimed(WishID int) (bool, error) {
	var wishclaim models.WishClaim
	wishclaimrecord := Instance.Where("`wish_claims`.enabled = ?", 1).Where("`wish_claims`.wish = ?", WishID).Joins("JOIN `users` on `users`.id = `wish_claims`.user").Where("`users`.enabled = ?", 1).Find(&wishclaim)
	if wishclaimrecord.Error != nil {
		return false, wishclaimrecord.Error
	} else if wishclaimrecord.RowsAffected != 1 {
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

// Get ALL user information
func GetAllUserInformation(UserID int) (models.User, error) {
	var user models.User
	userrecord := Instance.Where("`users`.enabled = ?", 1).Where("`users`.id = ?", UserID).Find(&user)
	if userrecord.Error != nil {
		return models.User{}, userrecord.Error
	} else if userrecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

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

// Get user information from wishlist
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

// Get user information from group
func GetUserMembersFromGroup(GroupID int) ([]models.User, error) {
	var users []models.User
	var group_memberships []models.GroupMembership

	membershiprecords := Instance.Where("`group_memberships`.enabled = ?", 1).Joins("JOIN `groups` on `group_memberships`.group = `groups`.id").Where("`groups`.enabled = ?", 1).Where("`groups`.id = ?", GroupID).Joins("JOIN `users` on `group_memberships`.member = `users`.id").Where("`users`.enabled = ?", 1).Find(&group_memberships)
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

// Get groups who are members of wishlist
func GetGroupMembersFromWishlist(WishlistID int) ([]models.Group, error) {

	var groups []models.Group

	groupsrecords := Instance.Where("`groups`.enabled = ?", 1).Joins("JOIN `group_memberships` on `groups`.id = `group_memberships`.group").Where("`group_memberships`.enabled = ?", 1).Joins("JOIN `users` on `group_memberships`.member = `users`.id").Where("`users`.enabled = ?", 1).Joins("JOIN `wishlist_memberships` on `groups`.id = `wishlist_memberships`.group").Where("`wishlist_memberships`.enabled = ?", 1).Where("`wishlist_memberships`.wishlist = ?", WishlistID).Group("groups.ID").Find(&groups)
	if groupsrecords.Error != nil {
		return []models.Group{}, groupsrecords.Error
	}

	if len(groups) == 0 {
		groups = []models.Group{}
	}

	return groups, nil
}

func GetGroupsAUserIsAMemberOf(UserID int) ([]models.Group, error) {

	var groups []models.Group

	// Retrieve groups that the user is a member of
	groupRecords := Instance.Where("`groups`.enabled = ?", 1).Joins("JOIN group_memberships on group_memberships.group = groups.id").Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.member = ?", UserID).Find(&groups)

	if groupRecords.Error != nil {
		return []models.Group{}, groupRecords.Error
	} else if groupRecords.RowsAffected == 0 {
		return []models.Group{}, nil
	}

	if len(groups) == 0 {
		groups = []models.Group{}
	}

	return groups, nil

}

// Retrieve memberships from group using group ID. Check that users are enabled.
func GetGroupMembershipsFromGroup(GroupID int) ([]models.GroupMembership, error) {

	var groupMemberships []models.GroupMembership

	groupmembershipRecords := Instance.Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.group = ?", GroupID).Joins("JOIN `users` on `group_memberships`.member = `users`.ID").Where("`users`.enabled = ?", 1).Find(&groupMemberships)

	if groupmembershipRecords.Error != nil {
		return []models.GroupMembership{}, groupmembershipRecords.Error
	} else if groupmembershipRecords.RowsAffected == 0 {
		return []models.GroupMembership{}, nil
	}

	if len(groupMemberships) == 0 {
		groupMemberships = []models.GroupMembership{}
	}

	return groupMemberships, nil

}

// Get all wishlists in groups
func GetWishlistsFromGroup(GroupID int) ([]models.Wishlist, error) {
	var wishlists []models.Wishlist
	wishlistrecords := Instance.Where("`wishlists`.enabled = ?", 1).Joins("JOIN wishlist_memberships on wishlist_memberships.wishlist = wishlists.id").Where("`wishlist_memberships`.group = ?", GroupID).Where("`wishlist_memberships`.enabled = ?", 1).Joins("JOIN `groups` on `wishlist_memberships`.group = `groups`.ID").Where("`groups`.enabled = ?", 1).Joins("JOIN `users` on `wishlists`.owner = `users`.id").Where("`users`.enabled = ?", 1).Find(&wishlists)

	if wishlistrecords.Error != nil {
		return []models.Wishlist{}, wishlistrecords.Error
	} else if wishlistrecords.RowsAffected == 0 {
		return []models.Wishlist{}, nil
	}

	return wishlists, nil
}

// Get all wishlists a user is an owner of
func GetOwnedWishlists(UserID int) (wishlists []models.Wishlist, err error) {
	wishlists = []models.Wishlist{}
	err = nil

	wishlistrecords := Instance.Order("`wishlists`.date desc, `wishlists`.name").Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.owner = ?", UserID).Joins("JOIN users on `users`.id = `wishlists`.owner").Where("`users`.enabled = ?", 1).Find(&wishlists)

	if wishlistrecords.Error != nil {
		return []models.Wishlist{}, wishlistrecords.Error
	}

	return wishlists, err
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

// Get wishes from wishlist
func GetWishesFromWishlist(WishlistID int) (bool, []models.Wish, error) {

	var wishes []models.Wish

	wishrecords := Instance.Where("`wishes`.enabled = ?", 1).Where("`wishes`.wishlist_id = ?", WishlistID).Joins("JOIN `users` on `users`.id = `wishes`.owner").Where("`users`.enabled = ?", 1).Find(&wishes)
	if wishrecords.Error != nil {
		return false, []models.Wish{}, wishrecords.Error
	} else if wishrecords.RowsAffected < 1 {
		return false, []models.Wish{}, nil
	}

	return true, wishes, nil
}

// get wish claims from wish, returns empty array without error if none are found.
func GetWishClaimFromWish(WishID int) ([]models.WishClaimObject, error) {
	var wish_claim models.WishClaim
	var wish_with_user models.WishClaimObject
	var wisharray_with_user []models.WishClaimObject

	wishclaimrecords := Instance.Where("`wish_claims`.enabled = ?", 1).Where("`wish_claims`.wish = ?", WishID).Joins("JOIN `users` on `users`.id = `wish_claims`.user").Where("`users`.enabled = ?", 1).Find(&wish_claim)
	if wishclaimrecords.Error != nil {
		return []models.WishClaimObject{}, wishclaimrecords.Error
	} else if wishclaimrecords.RowsAffected < 1 {
		return []models.WishClaimObject{}, nil
	}

	user_object, err := GetUserInformation(wish_claim.User)
	if err != nil {
		return []models.WishClaimObject{}, err
	}

	wish_with_user.User = user_object
	wish_with_user.CreatedAt = wish_claim.CreatedAt
	wish_with_user.DeletedAt = wish_claim.DeletedAt
	wish_with_user.Enabled = wish_claim.Enabled
	wish_with_user.ID = wish_claim.ID
	wish_with_user.Model = wish_claim.Model
	wish_with_user.UpdatedAt = wish_claim.UpdatedAt
	wish_with_user.Wish = wish_claim.Wish

	wisharray_with_user = append(wisharray_with_user, wish_with_user)

	return wisharray_with_user, err
}
