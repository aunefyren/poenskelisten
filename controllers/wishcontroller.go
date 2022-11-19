package controllers

import (
	"fmt"
	"net/http"
	"net/url"
	"poenskelisten/database"
	"poenskelisten/middlewares"
	"poenskelisten/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetWishesFromWishlist(context *gin.Context) {

	// Create wish request
	var wishlist_id = context.Param("wishlist_id")
	var group_id = context.Param("group_id")

	// Get user ID
	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Parse group id
	group_id_int, err := strconv.Atoi(group_id)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Parse wishlist id
	wishlist_id_int, err := strconv.Atoi(wishlist_id)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Verify ownership exists
	MembershipStatus, err := database.VerifyUserMembershipToGroup(UserID, group_id_int)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if !MembershipStatus {
		//context.JSON(http.StatusInternalServerError, gin.H{"error": groupmembershiprecord.Error.Error()})
		context.JSON(http.StatusInternalServerError, gin.H{"error": "You are not a member of this group."})
		context.Abort()
		return
	}

	wishes, err := database.GetWishesFromWishlist(wishlist_id_int)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err})
		context.Abort()
		return
	}

	owner_id, err := database.GetWishlistOwner(wishlist_id_int)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"owner_id": owner_id, "wishes": wishes, "message": "Wishes retrieved."})
}

func RegisterWish(context *gin.Context) {
	// Create wish request
	var wishlist_id = context.Param("wishlist_id")
	var wish models.WishCreationRequest
	var db_wish models.Wish

	if err := context.ShouldBindJSON(&wish); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Get user ID
	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Parse wishlist id
	wishlist_id_int, err := strconv.Atoi(wishlist_id)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Verify ownership exists
	MembershipStatus, err := database.VerifyUserOwnershipToWishlist(UserID, wishlist_id_int)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if !MembershipStatus {
		//context.JSON(http.StatusInternalServerError, gin.H{"error": groupmembershiprecord.Error.Error()})
		context.JSON(http.StatusBadRequest, gin.H{"error": "You are not an owner of this wishlist."})
		context.Abort()
		return
	}

	if len(wish.Name) < 5 || wish.Name == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "The name of the wish must be five or more letters."})
		context.Abort()
		return
	}

	unique_wish_name, err := database.VerifyUniqueWishNameinWishlist(wish.Name, wishlist_id_int)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if !unique_wish_name {
		context.JSON(http.StatusBadRequest, gin.H{"error": "There is already a wish with that name in this wishlist."})
		context.Abort()
		return
	}

	domain, _, err := parseRawURLFunction(wish.URL)
	if (err != nil || domain == "") && wish.URL != "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL given."})
		context.Abort()
		return
	}

	db_wish.Owner = UserID
	db_wish.WishlistID = wishlist_id_int
	db_wish.Name = wish.Name
	db_wish.Note = wish.Note
	db_wish.URL = wish.URL

	// Create user in DB
	record := database.Instance.Create(&db_wish)
	if record.Error != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": record.Error.Error()})
		context.Abort()
		return
	}

	new_wishes, err := database.GetWishesFromWishlist(wishlist_id_int)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err})
		context.Abort()
		return
	}

	// Return response
	context.JSON(http.StatusCreated, gin.H{"message": "Wish saved.", "wishes": new_wishes})
}

func parseRawURLFunction(rawurl string) (domain string, scheme string, err error) {
	u, err := url.ParseRequestURI(rawurl)
	if err != nil || u.Host == "" {
		u, repErr := url.ParseRequestURI("https://" + rawurl)
		if repErr != nil {
			fmt.Printf("Could not parse raw url: %s, error: %v", rawurl, err)
			return
		}
		domain = u.Host
		err = nil
		return
	}

	domain = u.Host
	scheme = u.Scheme
	return
}
