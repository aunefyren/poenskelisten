package database

import (
	"aunefyren/poenskelisten/models"
	"errors"

	"github.com/google/uuid"
)

// Set news post to disabled
func DeleteNewsPost(newsID uuid.UUID) error {
	var news models.News
	newsRecords := Instance.Model(news).Where("`news`.ID= ?", newsID).Update("enabled", 0)
	if newsRecords.Error != nil {
		return newsRecords.Error
	}
	if newsRecords.RowsAffected != 1 {
		return errors.New("Failed to delete news post in database.")
	}
	return nil
}

func GetNewsPosts() ([]models.News, error) {

	var newsPosts []models.News

	newsPostsRecords := Instance.Order("date desc").Where("`news`.enabled = ?", 1).Find(&newsPosts)

	if newsPostsRecords.Error != nil {
		return []models.News{}, newsPostsRecords.Error
	} else if newsPostsRecords.RowsAffected == 0 {
		return []models.News{}, nil
	}

	if len(newsPosts) == 0 {
		newsPosts = []models.News{}
	}

	return newsPosts, nil

}

func GetNewsPostByNewsID(newsID uuid.UUID) (models.News, error) {

	var newsPost models.News

	newsPostRecords := Instance.Where("`news`.enabled = ?", 1).Where("`news`.id = ?", newsID).Find(&newsPost)

	if newsPostRecords.Error != nil {
		return models.News{}, newsPostRecords.Error
	} else if newsPostRecords.RowsAffected != 1 {
		return models.News{}, errors.New("News post was not found.")
	}

	return newsPost, nil

}

func UpdateNewsPostInDB(newsPostOriginal models.News) (newsPost models.News, err error) {
	err = nil
	newsPost = newsPostOriginal

	record := Instance.Save(&newsPost)

	if record.Error != nil {
		return newsPost, record.Error
	}

	return
}
