package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetNews(context *gin.Context) {
	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	userObject, err := database.GetUserInformation(userID)
	if err != nil {
		log.Println("Failed to get user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user."})
		context.Abort()
		return
	}

	// Get all enabled news
	newsPosts, err := database.GetNewsPosts()
	if err != nil {
		// If there is an error getting the list of news, return an internal server error
		log.Println("Failed to get news. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get news."})
		context.Abort()
		return
	}

	now := time.Now()
	temporaryNewsPosts := []models.News{}
	for _, newsPost := range newsPosts {
		if newsPost.Date.After(now) && !userObject.Admin {
			continue
		}

		if newsPost.ExpiryDate != nil && newsPost.ExpiryDate.Before(now) {
			continue
		}

		temporaryNewsPosts = append(temporaryNewsPosts, newsPost)
	}
	newsPosts = temporaryNewsPosts

	sort.Slice(newsPosts, func(i, j int) bool {
		return newsPosts[j].Date.Before(newsPosts[i].Date)
	})

	// Return a response with all news posts
	context.JSON(http.StatusCreated, gin.H{"message": "News retrieved.", "news": newsPosts})
}

func GetNewsPost(context *gin.Context) {
	var newsIDString = context.Param("news_id")

	newsID, err := uuid.Parse(newsIDString)
	if err != nil {
		log.Println("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed parse request."})
		context.Abort()
		return
	}

	// Get the newspost by id
	newsPost, err := database.GetNewsPostByNewsID(newsID)
	if err != nil {
		// If there is an error getting the news, return an internal server error
		log.Println("Failed to get news post. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed get news post."})
		context.Abort()
		return
	}

	// Return a response with all news posts
	context.JSON(http.StatusCreated, gin.H{"message": "News retrieved.", "news": newsPost})
}

func RegisterNewsPost(context *gin.Context) {
	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	userObject, err := database.GetUserInformation(userID)
	if err != nil {
		log.Println("Failed to get user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user."})
		context.Abort()
		return
	}

	// Create a new instance of the News and NewsCreationRequest models
	var news models.News
	var newsCreationRequest models.NewsCreationRequest

	// Bind the incoming request body to the NewsCreationRequest model
	if err := context.ShouldBindJSON(&newsCreationRequest); err != nil {
		// If there is an error binding the request, return a Bad Request response
		log.Println("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	newsCreationRequest.Title = strings.TrimSpace(newsCreationRequest.Title)
	newsCreationRequest.Body = strings.TrimSpace(newsCreationRequest.Body)

	// Copy the data from the NewsCreationRequest model to the News model
	news.Title = newsCreationRequest.Title
	news.Body = newsCreationRequest.Body

	// Verify that the News title is not empty and has at least 5 characters
	if len(news.Title) < 5 || news.Title == "" {
		// If the group name is not valid, return a Bad Request response
		context.JSON(http.StatusBadRequest, gin.H{"error": "The title of the news post must be five or more letters."})
		context.Abort()
		return
	}

	stringMatch, requirements, err := utilities.ValidateTextCharacters(news.Title)
	if err != nil {
		log.Println("Failed to validate news title text string. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
		context.Abort()
		return
	} else if !stringMatch {
		log.Println("News title text string failed validation.")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	if len(news.Body) < 5 || news.Body == "" {
		// If the News body is not valid, return a Bad Request response
		context.JSON(http.StatusBadRequest, gin.H{"error": "The body of the news post must be five or more letters."})
		context.Abort()
		return
	}

	stringMatch, requirements, err = utilities.ValidateTextCharacters(news.Body)
	if err != nil {
		log.Println("Failed to validate news body text string. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
		context.Abort()
		return
	} else if !stringMatch {
		log.Println("News body text string failed validation.")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	news.Date = newsCreationRequest.Date
	news.ExpiryDate = newsCreationRequest.ExpiryDate
	news.ID = uuid.New()

	// Create the news post in the database
	newsRecord := database.Instance.Create(&news)
	if newsRecord.Error != nil {
		// If there is an error creating the news, return an Internal Server Error response
		log.Println("Failed to create news post. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create news post."})
		context.Abort()
		return
	}

	newsPosts, err := database.GetNewsPosts()
	if err != nil {
		// If there is an error getting the list of news, return an internal server error
		log.Println("Failed to get news posts. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get news posts."})
		context.Abort()
		return
	}

	now := time.Now()
	temporaryNewsPosts := []models.News{}
	for _, newsPost := range newsPosts {
		if newsPost.Date.Before(now) && !userObject.Admin {
			continue
		}

		if newsPost.ExpiryDate != nil && newsPost.ExpiryDate.Before(now) {
			continue
		}

		temporaryNewsPosts = append(temporaryNewsPosts, newsPost)
	}
	newsPosts = temporaryNewsPosts

	// Return a response indicating that the group was created, along with the updated list of groups
	context.JSON(http.StatusCreated, gin.H{"message": "News post created.", "news": newsPosts})
}

func DeleteNewsPost(context *gin.Context) {
	// Bind news request and get news_id ID from URL parameter
	newsIDString := context.Param("news_id")

	// Parse news ID
	newsID, err := uuid.Parse(newsIDString)
	if err != nil {
		log.Println("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Verify that news post exists
	_, err = database.GetNewsPostByNewsID(newsID)
	if err != nil {
		// If there is an error getting the news, return an internal server error
		log.Println("Failed to get news post. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get news post."})
		context.Abort()
		return
	}

	// Set the news post to disabled in the database
	err = database.DeleteNewsPost(newsID)
	if err != nil {
		log.Println("Failed to delete news post. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete news post."})
		context.Abort()
		return
	}

	// Get updated list of news
	newsPosts, err := database.GetNewsPosts()
	if err != nil {
		// If there is an error getting the list of news, return an internal server error
		log.Println("Failed to get news posts. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get news posts."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "News post deleted.", "news": newsPosts})

}

func APIEditNewsPost(context *gin.Context) {
	// Bind news request and get news_id ID from URL parameter
	newsIDString := context.Param("news_id")

	// Parse news ID
	newsID, err := uuid.Parse(newsIDString)
	if err != nil {
		log.Println("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	var news models.News
	var newsUpdateRequest models.NewsUpdateRequest

	// Bind the incoming request body to the NewsCreationRequest model
	if err := context.ShouldBindJSON(&newsUpdateRequest); err != nil {
		// If there is an error binding the request, return a Bad Request response
		log.Println("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	news, err = database.GetNewsPostByNewsID(newsID)
	if err != nil {
		log.Println("Failed to get news post. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get news post."})
		context.Abort()
		return
	}

	newsUpdateRequest.Title = strings.TrimSpace(newsUpdateRequest.Title)
	newsUpdateRequest.Body = strings.TrimSpace(newsUpdateRequest.Body)

	// Copy the data from the NewsCreationRequest model to the News model
	news.Title = newsUpdateRequest.Title
	news.Body = newsUpdateRequest.Body

	// Verify that the News title is not empty and has at least 5 characters
	if len(news.Title) < 5 || news.Title == "" {
		// If the group name is not valid, return a Bad Request response
		context.JSON(http.StatusBadRequest, gin.H{"error": "The title of the news post must be five or more letters."})
		context.Abort()
		return
	}

	stringMatch, requirements, err := utilities.ValidateTextCharacters(news.Title)
	if err != nil {
		log.Println("Failed to validate news title text string. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
		context.Abort()
		return
	} else if !stringMatch {
		log.Println("News title text string failed validation.")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	if len(news.Body) < 5 || news.Body == "" {
		// If the News body is not valid, return a Bad Request response
		context.JSON(http.StatusBadRequest, gin.H{"error": "The body of the news post must be five or more letters."})
		context.Abort()
		return
	}

	stringMatch, requirements, err = utilities.ValidateTextCharacters(news.Body)
	if err != nil {
		log.Println("Failed to validate news body text string. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
		context.Abort()
		return
	} else if !stringMatch {
		log.Println("News body text string failed validation.")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	news.Date = newsUpdateRequest.Date
	news.ExpiryDate = newsUpdateRequest.ExpiryDate

	// Create the news post in the database
	news, err = database.UpdateNewsPostInDB(news)
	if err != nil {
		// If there is an error creating the news, return an Internal Server Error response
		log.Println("Failed to create news post. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create news post."})
		context.Abort()
		return
	}

	// Return a response indicating that the group was created, along with the updated list of groups
	context.JSON(http.StatusCreated, gin.H{"message": "News post created.", "news": news})
}
