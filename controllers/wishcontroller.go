package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetWishesFromWishlist(context *gin.Context) {

	// Create wish request
	var wishlist_id = context.Param("wishlist_id")

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

	WishlistOwnership, err := database.VerifyUserOwnershipToWishlist(UserID, wishlist_id_int)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	WishlistMembership, err := database.VerifyUserMembershipToGroupmembershipToWishlist(UserID, wishlist_id_int)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	if !WishlistOwnership && !WishlistMembership {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You are not a member of, or an owner of this group."})
		context.Abort()
		return
	}

	wishes, err := database.GetWishesFromWishlist(wishlist_id_int, UserID)
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

	domain, scheme, err := parseRawURLFunction(wish.URL)
	if (err != nil || domain == "" || scheme == "") && wish.URL != "" {
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

	new_wishes, err := database.GetWishesFromWishlist(wishlist_id_int, UserID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err})
		context.Abort()
		return
	}

	// Return response
	context.JSON(http.StatusCreated, gin.H{"message": "Wish saved.", "wishes": new_wishes})
}

func DeleteWish(context *gin.Context) {

	// Create wish request
	var wish_id = context.Param("wish_id")

	// Get user ID
	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Parse wish id
	wish_id_int, err := strconv.Atoi(wish_id)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// get wishlist id
	wishlist_id, err := database.GetWishlistFromWish(wish_id_int)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Verify ownership exists
	MembershipStatus, err := database.VerifyUserOwnershipToWishlist(UserID, wishlist_id)
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

	// delete wish
	err = database.DeleteWish(wish_id_int)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	new_wishes, err := database.GetWishesFromWishlist(wishlist_id, UserID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err})
		context.Abort()
		return
	}

	// Return response
	context.JSON(http.StatusCreated, gin.H{"message": "Wish deleted.", "wishes": new_wishes})

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

func RegisterWishClaim(context *gin.Context) {
	// Create wish request
	var wish_id = context.Param("wish_id")
	var wishclaim models.WishClaimCreationRequest
	var db_wishclaim models.WishClaim

	if err := context.ShouldBindJSON(&wishclaim); err != nil {
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
	wish_id_int, err := strconv.Atoi(wish_id)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	db_wishlist_id, err := database.GetWishlistFromWish(wish_id_int)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	WishlistOwnership, err := database.VerifyUserOwnershipToWishlist(UserID, db_wishlist_id)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	WishlistMembership, err := database.VerifyUserMembershipToGroupmembershipToWishlist(UserID, db_wishlist_id)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	if !WishlistOwnership && !WishlistMembership {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You are not a member of, or an owner of this wishlist group."})
		context.Abort()
		return
	}

	// Verify if ownership of wish exists or not
	MembershipStatus, err := database.VerifyUserOwnershipToWish(UserID, wish_id_int)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if MembershipStatus {
		//context.JSON(http.StatusInternalServerError, gin.H{"error": groupmembershiprecord.Error.Error()})
		context.JSON(http.StatusBadRequest, gin.H{"error": "You cannot claim your own wish."})
		context.Abort()
		return
	}

	// Verify if wish is claimed or not
	ClaimStatus, err := database.VerifyWishIsClaimed(wish_id_int)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if ClaimStatus {
		//context.JSON(http.StatusInternalServerError, gin.H{"error": groupmembershiprecord.Error.Error()})
		context.JSON(http.StatusBadRequest, gin.H{"error": "Wish is already claimed."})
		context.Abort()
		return
	}

	db_wishclaim.User = UserID
	db_wishclaim.Wish = wish_id_int

	// Create wish is claimed
	record := database.Instance.Create(&db_wishclaim)
	if record.Error != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": record.Error.Error()})
		context.Abort()
		return
	}

	if wishclaim.WishlistID != 0 {
		new_wishes, err := database.GetWishesFromWishlist(wishclaim.WishlistID, UserID)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": err})
			context.Abort()
			return
		}

		// Return response
		context.JSON(http.StatusCreated, gin.H{"message": "Wish claimed.", "wishes": new_wishes})
	} else {
		context.JSON(http.StatusCreated, gin.H{"message": "Wish claimed."})
	}
}

func RemoveWishClaim(context *gin.Context) {
	// Create wish request
	var wish_id = context.Param("wish_id")
	var wishclaim models.WishClaimCreationRequest

	if err := context.ShouldBindJSON(&wishclaim); err != nil {
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
	wish_id_int, err := strconv.Atoi(wish_id)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	db_wishlist_id, err := database.GetWishlistFromWish(wish_id_int)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	WishlistOwnership, err := database.VerifyUserOwnershipToWishlist(UserID, db_wishlist_id)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	WishlistMembership, err := database.VerifyUserMembershipToGroupmembershipToWishlist(UserID, db_wishlist_id)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	if !WishlistOwnership && !WishlistMembership {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You are not a member of, or an owner of this wishlist group."})
		context.Abort()
		return
	}

	// Verify if ownership of wish exists or not
	OwnershipStatus, err := database.VerifyUserOwnershipToWishClaimByWish(UserID, wish_id_int)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if !OwnershipStatus {
		//context.JSON(http.StatusInternalServerError, gin.H{"error": groupmembershiprecord.Error.Error()})
		context.JSON(http.StatusBadRequest, gin.H{"error": "You cannot unclaim a wish you haven't claimed."})
		context.Abort()
		return
	}

	// Delete the membership
	err = database.DeleteWishClaimByUserAndWish(wish_id_int, UserID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	if wishclaim.WishlistID != 0 {
		new_wishes, err := database.GetWishesFromWishlist(wishclaim.WishlistID, UserID)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": err})
			context.Abort()
			return
		}

		// Return response
		context.JSON(http.StatusCreated, gin.H{"message": "Wish claimed.", "wishes": new_wishes})
	} else {
		context.JSON(http.StatusCreated, gin.H{"message": "Wish claimed."})
	}
}
