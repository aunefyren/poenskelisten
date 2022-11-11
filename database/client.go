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
