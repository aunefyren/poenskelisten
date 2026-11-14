package database

import (
	"aunefyren/poenskelisten/models"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	moderncsqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// Get wishlist id from wish id
func GetWishlistIDFromWish(WishID uuid.UUID) (*uuid.UUID, error) {
	var wish models.Wish

	wishRecord := Instance.
		Where("id = ? AND enabled = ?", WishID, true).
		Find(&wish)

	if wishRecord.Error != nil {
		return nil, wishRecord.Error
	} else if wishRecord.RowsAffected != 1 {
		return nil, nil
	}

	return &wish.WishlistID, nil
}

// Get wish by wish ID
func GetWishByWishID(wishID uuid.UUID) (*models.Wish, error) {
	var wish models.Wish

	wishRecord := Instance.
		Where("id = ? AND enabled = ?", wishID, true).
		Find(&wish)

	if wishRecord.Error != nil {
		return nil, wishRecord.Error
	} else if wishRecord.RowsAffected != 1 {
		return nil, nil
	}

	return &wish, nil
}

func UpdateWishInDB(wishOriginal models.Wish) (wish models.Wish, err error) {
	wish = wishOriginal
	err = nil

	wishRecord := Instance.Save(wish)

	if wishRecord.Error != nil {
		return wish, wishRecord.Error
	}

	return
}

// Get wishes from wishlist
// GetWishIDsFromWishlist returns the IDs of every wish on a wishlist, enabled
// or not, so cleanup of per-wish files can't miss any.
func GetWishIDsFromWishlist(wishlistID uuid.UUID) ([]uuid.UUID, error) {
	var wishIDs []uuid.UUID
	err := Instance.
		Model(&models.Wish{}).
		Where("wishlist_id = ?", wishlistID).
		Pluck("id", &wishIDs).Error
	return wishIDs, err
}

func GetWishesFromWishlist(WishlistID uuid.UUID) (bool, []models.Wish, error) {
	var wishes []models.Wish
	wishRecords := Instance.
		Order("wishes.created_at ASC").
		Where("wishes.enabled = ? AND wishes.wishlist_id = ?", true, WishlistID).
		Joins("JOIN users ON users.id = wishes.owner_id").
		Where("users.enabled = ?", true).
		Find(&wishes)

	if wishRecords.Error != nil {
		return false, []models.Wish{}, wishRecords.Error
	} else if wishRecords.RowsAffected < 1 {
		return false, []models.Wish{}, nil
	}

	return true, wishes, nil
}

// get wish claims from wish, returns empty array without error if none are found.
func GetWishClaimFromWish(WishID uuid.UUID) ([]models.WishClaimObject, error) {
	var wishClaim models.WishClaim
	var wishWithUser models.WishClaimObject
	var wishArrayWithUser []models.WishClaimObject

	wishClaimRecords := Instance.
		Where("wish_claims.enabled = ? AND wish_claims.wish_id = ?", true, WishID).
		Joins("JOIN users ON users.id = wish_claims.user_id").
		Where("users.enabled = ?", true).
		Find(&wishClaim)

	if wishClaimRecords.Error != nil {
		return []models.WishClaimObject{}, wishClaimRecords.Error
	} else if wishClaimRecords.RowsAffected < 1 {
		return []models.WishClaimObject{}, nil
	}

	userObject, err := GetUserInformation(wishClaim.UserID)
	if err != nil {
		return []models.WishClaimObject{}, err
	}

	profile := UserProfile(userObject)
	userObjectMinimal := models.UserMinimal{
		GormModel: profile.GormModel,
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
	}

	wishWithUser.User = userObjectMinimal
	wishWithUser.CreatedAt = wishClaim.CreatedAt
	wishWithUser.DeletedAt = wishClaim.DeletedAt
	wishWithUser.Enabled = wishClaim.Enabled
	wishWithUser.ID = wishClaim.ID
	wishWithUser.UpdatedAt = wishClaim.UpdatedAt
	wishWithUser.Wish = wishClaim.WishID

	wishArrayWithUser = append(wishArrayWithUser, wishWithUser)

	return wishArrayWithUser, err
}

// Verify if a user ID is an owner of a wish
func VerifyUserOwnershipToWish(UserID uuid.UUID, WishID uuid.UUID) (bool, error) {
	var wish models.Wish

	wishRecord := Instance.
		Where("id = ? AND enabled = ? AND owner_id = ?", WishID, true, UserID).
		Find(&wish)

	if wishRecord.Error != nil {
		return false, wishRecord.Error
	} else if wishRecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Verify if a user ID is an owner of a wish
func VerifyUserOwnershipToWishClaimByWish(UserID uuid.UUID, WishID uuid.UUID) (bool, error) {
	var wishClaim models.WishClaim

	wishClaimRecord := Instance.
		Where("enabled = ? AND wish_id = ? AND user_id = ?", true, WishID, UserID).
		Find(&wishClaim)

	if wishClaimRecord.Error != nil {
		return false, wishClaimRecord.Error
	} else if wishClaimRecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Verify if a user ID is an owner of a wish
func VerifyWishIsClaimed(WishID uuid.UUID) (bool, error) {
	var wishClaim models.WishClaim

	wishClaimRecord := Instance.
		Where("wish_claims.enabled = ? AND wish_claims.wish_id = ?", true, WishID).
		Joins("JOIN users ON users.id = wish_claims.user_id").
		Where("users.enabled = ?", true).
		Find(&wishClaim)

	if wishClaimRecord.Error != nil {
		return false, wishClaimRecord.Error
	} else if wishClaimRecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Set wish claim to disabled
func DeleteWishClaimByUserAndWish(WishID uuid.UUID, UserID uuid.UUID) error {
	var wishClaim models.WishClaim

	wishClaimRecords := Instance.
		Model(wishClaim).
		Where("wish_id = ? AND user_id = ? AND enabled = ?", WishID, UserID, true).
		Update("enabled", false)

	if wishClaimRecords.Error != nil {
		return wishClaimRecords.Error
	}
	if wishClaimRecords.RowsAffected != 1 {
		return errors.New("Failed to delete wish claim membership in database.")
	}
	return nil
}

// Get wish by wish ID
func GetWishlistByWishID(wishID uuid.UUID) (bool, models.Wishlist, error) {
	var wishlist models.Wishlist

	wishlistRecords := Instance.
		Where("wishlists.enabled = ?", true).
		Joins("JOIN wishes ON wishlists.id = wishes.wishlist_id").
		Where("wishes.enabled = ? AND wishes.id = ?", true, wishID).
		Find(&wishlist)

	if wishlistRecords.Error != nil {
		return false, models.Wishlist{}, wishlistRecords.Error
	} else if wishlistRecords.RowsAffected != 1 {
		return false, models.Wishlist{}, nil
	}

	return true, wishlist, nil
}

func CreateWishInDB(wishDB models.Wish) (wish models.Wish, err error) {
	record := Instance.Create(&wishDB)

	if record.Error != nil {
		return models.Wish{}, record.Error
	}

	if record.RowsAffected != 1 {
		return models.Wish{}, errors.New("Wish not added to database.")
	}

	return wishDB, nil
}

// ErrWishAlreadyClaimed means the wish got a live claim before this one was
// stored.
var ErrWishAlreadyClaimed = errors.New("Wish is already claimed.")

// CreateWishClaimInDB stores a claim unless the wish already has a live one.
// Callers check first for a friendly answer, but two claims checked at once
// both find the wish free; the unique index from EnsureOneClaimPerWish turns
// the second insert away, and on MySQL, which has no partial index, the wish
// is locked while its claims are counted.
func CreateWishClaimInDB(wishClaimDB models.WishClaim) (wishClaim models.WishClaim, err error) {
	if Instance.Dialector.Name() == "mysql" {
		err = Instance.Transaction(func(tx *gorm.DB) error {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", wishClaimDB.WishID).Find(&models.Wish{}).Error; err != nil {
				return err
			}
			var live int64
			if err := tx.Model(&models.WishClaim{}).
				Joins("JOIN users ON users.id = wish_claims.user_id").
				Where("wish_claims.wish_id = ? AND wish_claims.enabled = ? AND users.enabled = ?", wishClaimDB.WishID, true, true).
				Count(&live).Error; err != nil {
				return err
			} else if live > 0 {
				return ErrWishAlreadyClaimed
			}
			return insertWishClaim(tx, &wishClaimDB)
		})
	} else {
		err = insertWishClaim(Instance, &wishClaimDB)
	}
	if err != nil {
		return models.WishClaim{}, err
	}
	return wishClaimDB, nil
}

func insertWishClaim(db *gorm.DB, wishClaim *models.WishClaim) error {
	record := db.Create(wishClaim)
	if isUniqueViolation(record.Error) {
		return ErrWishAlreadyClaimed
	} else if record.Error != nil {
		return record.Error
	} else if record.RowsAffected != 1 {
		return errors.New("WishClaim not added to database.")
	}
	return nil
}

// EnsureOneClaimPerWish adds the unique index that refuses a second live
// claim on a wish. Claims that would break it are released first: those of
// disabled accounts, which already count as free, and all but the oldest
// live claim of a wish claimed twice by an earlier race.
func EnsureOneClaimPerWish() error {
	if Instance.Dialector.Name() == "mysql" {
		return nil
	}
	if err := Instance.Exec("UPDATE wish_claims SET enabled = ? WHERE enabled = ? AND user_id IN (SELECT id FROM users WHERE enabled = ?)", false, true, false).Error; err != nil {
		return err
	}
	if err := Instance.Exec(`UPDATE wish_claims SET enabled = ? WHERE id IN (
		SELECT id FROM (
			SELECT id, ROW_NUMBER() OVER (PARTITION BY wish_id ORDER BY created_at, id) AS position
			FROM wish_claims WHERE enabled = ? AND deleted_at IS NULL
		) ranked WHERE position > 1)`, false, true).Error; err != nil {
		return err
	}
	return Instance.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_wish_claims_one_live ON wish_claims (wish_id) WHERE enabled AND deleted_at IS NULL").Error
}

// DisableWishClaimsOfUser releases every claim of a user whose account is
// deleted, so their wishes can be claimed again.
func DisableWishClaimsOfUser(userID uuid.UUID) error {
	return Instance.Model(&models.WishClaim{}).
		Where("user_id = ? AND enabled = ?", userID, true).
		Update("enabled", false).Error
}

// isUniqueViolation recognises a broken unique index. GORM translates
// PostgreSQL's error, but not the modernc SQLite driver's.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	if translator, ok := Instance.Dialector.(gorm.ErrorTranslator); ok && errors.Is(translator.Translate(err), gorm.ErrDuplicatedKey) {
		return true
	}
	var sqliteErr *moderncsqlite.Error
	return errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE
}
