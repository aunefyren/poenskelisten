package database

import (
	"aunefyren/poenskelisten/models"
	"errors"
)

// Get wishlist id from wish id
func GetWishlistFromWish(WishID int) (bool, int, error) {
	var wish models.Wish
	wishrecord := Instance.Where("`wishes`.enabled = ?", 1).Where("`wishes`.id = ?", WishID).Find(&wish)
	if wishrecord.Error != nil {
		return false, 0, wishrecord.Error
	} else if wishrecord.RowsAffected != 1 {
		return false, 0, nil
	}

	return true, wish.WishlistID, nil
}

// Get wish by wish ID
func GetWishByWishID(wishID int) (bool, models.Wish, error) {

	var wish models.Wish

	wishRecord := Instance.Where("`wishes`.enabled = ?", 1).Where("`wishes`.id = ?", wishID).Find(&wish)

	if wishRecord.Error != nil {
		return false, models.Wish{}, wishRecord.Error
	} else if wishRecord.RowsAffected != 1 {
		return false, models.Wish{}, nil
	}

	return true, wish, nil

}

func UpdateWishValuesInDatabase(wishID int, wishName string, wishNote string, wishURL string) error {

	var wish models.Wish

	wishRecord := Instance.Model(wish).Where("`wishes`.enabled = ?", 1).Where("`wishes`.ID = ?", wishID).Update("name", wishName)
	if wishRecord.Error != nil {
		return wishRecord.Error
	} else if wishRecord.RowsAffected != 1 {
		return errors.New("Name not changed in database.")
	}

	wishRecord = Instance.Model(wish).Where("`wishes`.enabled = ?", 1).Where("`wishes`.ID = ?", wishID).Update("note", wishNote)
	if wishRecord.Error != nil {
		return wishRecord.Error
	} else if wishRecord.RowsAffected != 1 {
		return errors.New("Note not changed in database.")
	}

	wishRecord = Instance.Model(wish).Where("`wishes`.enabled = ?", 1).Where("`wishes`.ID = ?", wishID).Update("url", wishURL)
	if wishRecord.Error != nil {
		return wishRecord.Error
	} else if wishRecord.RowsAffected != 1 {
		return errors.New("URL not changed in database.")
	}

	return nil

}
