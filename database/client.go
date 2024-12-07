package database

import (
	"aunefyren/poenskelisten/models"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/thanhpk/randstr"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var Instance *gorm.DB
var dbError error

func Connect(dbType string, timezone string, dbUsername string, dbPassword string, dbIP string, dbPort int, dbName string, dbSSL bool, dbLocation string) error {

	if strings.ToLower(dbType) == "postgres" {
		log.Println("Attempting to connect to postgres database.")

		var sslString = "disable"
		if dbSSL {
			sslString = "enabled"
		}

		connStrDb := "host=" + dbIP + " user=" + dbUsername + " password=" + dbPassword + " dbname=" + dbName + " port=" + strconv.Itoa(dbPort) + " sslmode=" + sslString + " TimeZone=" + timezone
		Instance, dbError = gorm.Open(postgres.New(postgres.Config{
			DSN:                  connStrDb,
			PreferSimpleProtocol: true,
		}), &gorm.Config{
			PrepareStmt: true,
		})
		if dbError != nil {
			log.Println("Failed to connect to database. Error: " + dbError.Error())
			return errors.New("Failed to connect to database.")
		}
	} else if strings.ToLower(dbType) == "sqlite" {
		log.Println("Attempting to connect to sqlite database.")

		Instance, dbError = gorm.Open(sqlite.Open(dbLocation), &gorm.Config{})
		if dbError != nil {
			log.Println("Failed to connect to database. Error: " + dbError.Error())
			return errors.New("Failed to connect to database.")
		}
	} else if strings.ToLower(dbType) == "mysql" {
		log.Println("Attempting to connect to mysql database.")

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
				log.Println("Failed to connect to database. Error: " + dbError.Error())
				return errors.New("Failed to connect to database.")
			}
		}
	} else {
		return errors.New("Database type not recognized.")
	}

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
	invite.Code = strings.ToUpper(randomString)
	invite.ID = uuid.New()

	record := Instance.Create(&invite)
	if record.Error != nil {
		return "", record.Error
	}

	return invite.Code, nil
}

// Genrate a random verification code an return ut
func GenrateRandomVerificationCodeForuser(userID uuid.UUID) (string, error) {

	randomString := randstr.String(8)
	verificationCode := strings.ToUpper(randomString)

	var user models.User
	userrecord := Instance.Model(user).Where("enabled = ?", true).Where("ID = ?", userID).Update("verification_code", verificationCode)
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
	userrecords := Instance.Where("enabled = ?", true).Where("email = ?", providedEmail).Find(&user)
	if userrecords.Error != nil {
		return false, userrecords.Error
	}
	if userrecords.RowsAffected != 0 {
		return false, nil
	}
	return true, nil
}

// Verify if user has a verification code set
func VerifyUserHasVerfificationCode(userID uuid.UUID) (bool, error) {
	var user models.User
	userrecords := Instance.Where("enabled = ?", true).Where("ID = ?", userID).Find(&user)
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
func VerifyUserVerfificationCodeMatches(userID uuid.UUID, verificationCode string) (bool, error) {

	var user models.User

	userrecords := Instance.Where("enabled = ?", true).Where("ID = ?", userID).Where("verification_code = ?", verificationCode).Find(&user)

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
func VerifyUserIsVerified(userID uuid.UUID) (bool, error) {

	var user models.User
	userrecords := Instance.Where("id = ?", userID).Find(&user)
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
	inviterecords := Instance.Where("enabled = ?", true).Where("used = ?", false).Where("code = ?", providedCode).Find(&invitestruct)
	if inviterecords.Error != nil {
		return false, inviterecords.Error
	}
	if inviterecords.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Set invite code to used
func SetUsedUserInviteCode(providedCode string, userIDClaimer uuid.UUID) error {
	var invitestruct models.Invite

	inviterecords := Instance.Model(invitestruct).Where("code = ?", providedCode).Update("used", true)
	if inviterecords.Error != nil {
		return inviterecords.Error
	}
	if inviterecords.RowsAffected != 1 {
		return errors.New("Code not changed in database.")
	}

	inviterecords = Instance.Model(invitestruct).Where("code = ?", providedCode).Update("recipient_id", userIDClaimer)
	if inviterecords.Error != nil {
		return inviterecords.Error
	}
	if inviterecords.RowsAffected != 1 {
		return errors.New("Recipient not changed in database.")
	}

	return nil
}

// Set user to verified
func SetUserVerification(userID uuid.UUID, verified bool) error {

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

// Set group to disabled
func DeleteGroup(GroupID uuid.UUID) error {
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
func DeleteGroupMembership(GroupMembershipID uuid.UUID) error {
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
func DeleteWishlist(WishlistID uuid.UUID) error {
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
func DeleteWishlistMembership(WishlistMembershipID uuid.UUID) error {
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

// Get user information from group
func GetUserMembersFromGroup(GroupID uuid.UUID) ([]models.User, error) {
	var users []models.User
	var group_memberships []models.GroupMembership

	membershiprecords := Instance.Where("`group_memberships`.enabled = ?", 1).Joins("JOIN `groups` on `group_memberships`.group = `groups`.id").Where("`groups`.enabled = ?", 1).Where("`groups`.id = ?", GroupID).Joins("JOIN `users` on `group_memberships`.group_id = `users`.id").Where("`users`.enabled = ?", 1).Find(&group_memberships)
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
