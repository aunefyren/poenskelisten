package database

import (
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"database/sql"
	"errors"
	"fmt"
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
		logger.Log.Debug("attempting to connect to postgres database")

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
			logger.Log.Error("failed to connect to database. error: " + dbError.Error())
			return errors.New("failed to connect to database")
		}
	} else if strings.ToLower(dbType) == "sqlite" {
		logger.Log.Debug("attempting to connect to sqlite database")

		Instance, dbError = gorm.Open(sqlite.Open(dbLocation), &gorm.Config{})
		if dbError != nil {
			logger.Log.Error("failed to connect to database. error: " + dbError.Error())
			return errors.New("failed to connect to database")
		}
	} else if strings.ToLower(dbType) == "mysql" {
		logger.Log.Debug("attempting to connect to mysql database")

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
				logger.Log.Error("failed to connect to database. error: " + dbError.Error())
				return errors.New("failed to connect to database")
			}
		}
	} else {
		return errors.New("database type not recognized")
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
	errUser := Instance.AutoMigrate(&models.User{})
	errInvite := Instance.AutoMigrate(&models.Invite{})
	errGroup := Instance.AutoMigrate(&models.Group{})
	errGroupMembership := Instance.AutoMigrate(&models.GroupMembership{})
	errWishlist := Instance.AutoMigrate(&models.Wishlist{})
	errWishlistMembership := Instance.AutoMigrate(&models.WishlistMembership{})
	errWishlistCollaborator := Instance.AutoMigrate(&models.WishlistCollaborator{})
	errWish := Instance.AutoMigrate(&models.Wish{})
	errWishClaim := Instance.AutoMigrate(&models.WishClaim{})
	errNews := Instance.AutoMigrate(&models.News{})

	err := errors.Join(errUser, errInvite, errGroup, errGroupMembership, errWishlist, errWishlistMembership, errWishlistCollaborator, errWish, errWishClaim, errNews)
	if err != nil {
		panic(err)
	}
	logger.Log.Debug("database migration completed")
}

// Generate a random invite code an return ut
func GenerateRandomInvite() (string, error) {
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

// Generate a random verification code an return ut
func GenerateRandomVerificationCodeForUser(userID uuid.UUID) (string, error) {

	randomString := randstr.String(8)
	verificationCode := strings.ToUpper(randomString)

	var user models.User
	userRecord := Instance.
		Model(user).
		Where(&models.User{Enabled: &utilities.DBTrue}).
		Where(&models.GormModel{ID: userID}).
		Update("verification_code", verificationCode)

	if userRecord.Error != nil {
		return "", userRecord.Error
	}
	if userRecord.RowsAffected != 1 {
		return "", errors.New("verification code not changed in database")
	}

	return verificationCode, nil

}

// Verify e-mail is not in use
func VerifyUniqueUserEmail(providedEmail string) (bool, error) {
	var user models.User
	userRecords := Instance.
		Where(&models.User{Enabled: &utilities.DBTrue, Email: &providedEmail}).
		Find(&user)

	if userRecords.Error != nil {
		return false, userRecords.Error
	}
	if userRecords.RowsAffected != 0 {
		return false, nil
	}
	return true, nil
}

// Verify if user has a verification code set
func VerifyUserHasVerificationCode(userID uuid.UUID) (bool, error) {
	var user models.User
	userRecords := Instance.
		Where(&models.User{Enabled: &utilities.DBTrue}).
		Where(&models.GormModel{ID: userID}).
		Find(&user)

	if userRecords.Error != nil {
		return false, userRecords.Error
	}
	if userRecords.RowsAffected != 1 {
		return false, errors.New("couldn't find the user")
	}

	if user.VerificationCode == nil || *user.VerificationCode == "" {
		return false, nil
	} else {
		return true, nil
	}
}

// Verify if user has a verification code set
func VerifyUserVerificationCodeMatches(userID uuid.UUID, verificationCode string) (bool, error) {
	var user models.User

	userRecords := Instance.
		Where(&models.User{Enabled: &utilities.DBTrue, VerificationCode: &verificationCode}).
		Where(&models.GormModel{ID: userID}).
		Find(&user)

	if userRecords.Error != nil {
		return false, userRecords.Error
	}

	if userRecords.RowsAffected != 1 {
		return false, nil
	} else {
		return true, nil
	}

}

// Verify if user is verified
func VerifyUserIsVerified(userID uuid.UUID) (bool, error) {
	var user models.User

	userRecords := Instance.
		Where(&models.GormModel{ID: userID}).
		Find(&user)

	if userRecords.Error != nil {
		return false, userRecords.Error
	}
	if userRecords.RowsAffected != 1 {
		return false, errors.New("no user found")
	}

	return *user.Verified, nil
}

// Verify unsued invite code exists
func VerifyUnusedUserInviteCode(providedCode string) (bool, error) {
	var inviteStruct models.Invite

	inviteRecords := Instance.
		Where(&models.Invite{Used: false, Code: providedCode, Enabled: &utilities.DBTrue}).
		Find(&inviteStruct)

	if inviteRecords.Error != nil {
		return false, inviteRecords.Error
	}
	if inviteRecords.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Set invite code to used
func SetUsedUserInviteCode(providedCode string, userIDClaimer uuid.UUID) error {
	var inviteStruct models.Invite

	inviteRecords := Instance.
		Model(inviteStruct).Where(&models.Invite{Code: providedCode}).
		Update("used", true)

	if inviteRecords.Error != nil {
		return inviteRecords.Error
	}
	if inviteRecords.RowsAffected != 1 {
		return errors.New("code not changed in database")
	}

	inviteRecords = Instance.
		Model(inviteStruct).
		Where(&models.Invite{Code: providedCode}).
		Update("recipient_id", userIDClaimer)

	if inviteRecords.Error != nil {
		return inviteRecords.Error
	}
	if inviteRecords.RowsAffected != 1 {
		return errors.New("recipient not changed in database")
	}

	return nil
}

// Set user to verified
func SetUserVerification(userID uuid.UUID, verified bool) error {
	var user models.User

	userRecords := Instance.
		Model(user).Where(models.GormModel{ID: userID}).
		Where(&models.User{Enabled: &utilities.DBTrue}).
		Update("verified", verified)

	if userRecords.Error != nil {
		return userRecords.Error
	}
	if userRecords.RowsAffected != 1 {
		return errors.New("verification not changed in database")
	}

	return nil
}

// Set group to disabled
func DeleteGroup(GroupID uuid.UUID) error {
	var group models.Group

	groupRecords := Instance.
		Model(group).Where(&models.GormModel{ID: GroupID}).
		Update("enabled", false)

	if groupRecords.Error != nil {
		return groupRecords.Error
	}
	if groupRecords.RowsAffected != 1 {
		return errors.New("failed to delete group in database")
	}
	return nil
}

// Set group membership to disabled
func DeleteGroupMembership(GroupMembershipID uuid.UUID) error {
	var groupMembership models.GroupMembership

	groupRecords := Instance.
		Model(groupMembership).
		Where(&models.GormModel{ID: GroupMembershipID}).
		Update("enabled", false)

	if groupRecords.Error != nil {
		return groupRecords.Error
	}
	if groupRecords.RowsAffected != 1 {
		return errors.New("failed to delete group membership in database")
	}
	return nil
}

// Set wishlist to disabled
func DeleteWishlist(WishlistID uuid.UUID) error {
	var wishlist models.Wishlist

	wishlistRecords := Instance.
		Model(wishlist).
		Where(&models.GormModel{ID: WishlistID}).
		Update("enabled", false)

	if wishlistRecords.Error != nil {
		return wishlistRecords.Error
	}
	if wishlistRecords.RowsAffected != 1 {
		return errors.New("failed to delete wishlist in database")
	}
	return nil
}

// Set wishlist membership to disabled
func DeleteWishlistMembership(WishlistMembershipID uuid.UUID) error {
	var wishlistMembership models.WishlistMembership

	wishlistMembershipRecords := Instance.
		Model(wishlistMembership).
		Where(&models.GormModel{ID: WishlistMembershipID}).
		Update("enabled", false)

	if wishlistMembershipRecords.Error != nil {
		return wishlistMembershipRecords.Error
	}
	if wishlistMembershipRecords.RowsAffected != 1 {
		return errors.New("failed to delete wishlist membership in database")
	}
	return nil
}

// Get user information from group
func GetUserMembersFromGroup(GroupID uuid.UUID) ([]models.User, error) {
	var users []models.User
	var groupMemberships []models.GroupMembership

	membershipRecords := Instance.
		Where(&models.GroupMembership{Enabled: true}).
		Joins("JOIN groups ON group_memberships.group = groups.id").
		Where("groups.enabled = ?", true).
		Where("groups.id = ?", GroupID).
		Joins("JOIN users ON group_memberships.group_id = users.id").
		Where("users.enabled = ?", true).
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
