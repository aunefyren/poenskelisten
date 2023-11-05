package controllers

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"errors"
	"fmt"
	"log"
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

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		log.Println("Failed to get config file. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get config file."})
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

	_, wishes, err := database.GetWishesFromWishlist(wishlist_id_int)
	if err != nil {
		log.Println("Failed to get wishes from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishes from database."})
		context.Abort()
		return
	}

	wishObjects, err := ConvertWishesToWishObjects(wishes, &UserID)
	if err != nil {
		log.Println("Failed to convert wishes to wish objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert wishes to wish objects."})
		context.Abort()
		return
	}

	owner_id, err := database.GetWishlistOwner(wishlist_id_int)
	if err != nil {
		log.Println("Failed to get wishlist owner. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist owner."})
		context.Abort()
		return
	}

	wishlistCollabs, err := database.GetWishlistCollaboratorsFromWishlist(wishlist_id_int)
	if err != nil {
		log.Println("Failed to get wishlist collaborators. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist collaborators."})
		context.Abort()
		return
	}

	wishlistCollabsIntArray := []int{}
	for _, wishlistCollab := range wishlistCollabs {
		wishlistCollabsIntArray = append(wishlistCollabsIntArray, wishlistCollab.User)
	}

	context.JSON(http.StatusOK, gin.H{"owner_id": owner_id, "collaborators": wishlistCollabsIntArray, "wishes": wishObjects, "message": "Wishes retrieved.", "currency": config.PoenskelistenCurrency, "padding": config.PoenskelistenCurrencyPad})
}

func ConvertWishToWishObject(wish models.Wish, requestUserID *int) (models.WishObject, error) {

	wishObject := models.WishObject{}

	user_object, err := database.GetUserInformation(wish.Owner)
	if err != nil {
		log.Println("Failed to get information about wish owner for wish'" + strconv.Itoa(int(wish.ID)) + "' and user '" + strconv.Itoa(int(wish.Owner)) + "'. Returning. Error: " + err.Error())
		return models.WishObject{}, err
	}

	wishclaimobject, err := database.GetWishClaimFromWish(int(wish.ID))
	if err != nil {
		log.Println("Failed to get wish claims wish'" + strconv.Itoa(int(wish.ID)) + "'. Returning. Error: " + err.Error())
		return models.WishObject{}, err
	}

	_, wishlist, err := database.GetWishlistByWishlistID(wish.WishlistID)
	if err != nil {
		log.Println("Failed to get wishlist for wish'" + strconv.Itoa(int(wish.ID)) + "'. Returning. Error: " + err.Error())
		return models.WishObject{}, err
	}

	wishlistOwnerUser, err := database.GetUserInformation(wishlist.Owner)
	if err != nil {
		log.Println("Failed to get information about wishlist owner for wish'" + strconv.Itoa(int(wish.ID)) + "' and user '" + strconv.Itoa(int(wishlist.Owner)) + "'. Returning. Error: " + err.Error())
		return models.WishObject{}, err
	}

	imageExists, err := CheckIfWishImageExists(int(wish.ID))
	if err != nil {
		log.Println("Failed to check if wish'" + strconv.Itoa(int(wish.ID)) + "' had image. Setting to false. Error: " + err.Error())
		wishObject.Image = false
	} else if imageExists {
		wishObject.Image = true
	} else {
		wishObject.Image = false
	}

	wishlistCollabs, err := database.GetWishlistCollaboratorsFromWishlist(wish.WishlistID)
	if err != nil {
		log.Println("Failed to get wishlist collaborator from database. Error: " + err.Error())
		return models.WishObject{}, errors.New("Failed to get wishlist from database.")
	}
	wishlistCollabObjects, err := ConvertWishlistCollaberatorsToWishlistCollaberatorObjects(wishlistCollabs)
	if err != nil {
		log.Println("Failed to convert wishlist collaborators to objects. Error: " + err.Error())
		return models.WishObject{}, errors.New("Failed to convert wishlist collaborators to objects.")
	}

	// Purge the reply if the requester is the owner
	if requestUserID != nil {
		if wish.Owner == *requestUserID {
			wishclaimobject = []models.WishClaimObject{}
		}

		for _, wishCollaborator := range wishlistCollabObjects {
			if int(wishCollaborator.User.ID) == int(*requestUserID) {
				wishclaimobject = []models.WishClaimObject{}
			}
		}
	}

	wishObject.CreatedAt = wish.CreatedAt
	wishObject.DeletedAt = wish.DeletedAt
	wishObject.Enabled = wish.Enabled
	wishObject.ID = wish.ID
	wishObject.Model = wish.Model
	wishObject.Name = wish.Name
	wishObject.Note = wish.Note
	wishObject.Owner = user_object
	wishObject.WishlistOwner = wishlistOwnerUser
	wishObject.WishClaim = wishclaimobject
	wishObject.URL = wish.URL
	wishObject.Price = wish.Price
	wishObject.UpdatedAt = wish.UpdatedAt
	wishObject.WishlistID = wish.WishlistID
	wishObject.WishClaimable = *wishlist.Claimable
	wishObject.Collaborators = wishlistCollabObjects

	return wishObject, nil

}

func ConvertWishesToWishObjects(wishes []models.Wish, requestUserID *int) ([]models.WishObject, error) {

	wishObjects := []models.WishObject{}

	for _, wish := range wishes {

		wishObject, err := ConvertWishToWishObject(wish, requestUserID)
		if err != nil {
			log.Println("Failed to convert wish '" + strconv.Itoa(int(wish.ID)) + "' to wish object. Skipping. Error: " + err.Error())
			continue
		}

		wishObjects = append(wishObjects, wishObject)

	}

	return wishObjects, nil

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

	// Verify if collaboration exists
	collaborationStatus, err := database.VerifyWishlistCollaboratorToWishlist(wishlist_id_int, UserID)
	if err != nil {
		log.Println("Failed to verify wishlist collaborator status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist collaborator status."})
		context.Abort()
		return
	}

	// Verify ownership exists
	MembershipStatus, err := database.VerifyUserOwnershipToWishlist(UserID, wishlist_id_int)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if !MembershipStatus && !collaborationStatus {
		//context.JSON(http.StatusInternalServerError, gin.H{"error": groupmembershiprecord.Error.Error()})
		context.JSON(http.StatusBadRequest, gin.H{"error": "You are not an owner or collaborator of this wishlist."})
		context.Abort()
		return
	}

	if len(wish.Name) < 5 || wish.Name == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "The name of the wish must be five or more letters."})
		context.Abort()
		return
	}

	// Validate wish name format
	stringMatch, requirements, err := utilities.ValidateTextCharacters(wish.Name)
	if err != nil {
		log.Println("Failed to validate wish name text string. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
		context.Abort()
		return
	} else if !stringMatch {
		log.Println("Wish name text string failed validation.")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	// Validate wish note format
	stringMatch, requirements, err = utilities.ValidateTextCharacters(wish.Note)
	if err != nil {
		log.Println("Failed to validate wish note text string. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
		context.Abort()
		return
	} else if !stringMatch {
		log.Println("Wish note text string failed validation.")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	// Validate wish url format
	stringMatch, requirements, err = utilities.ValidateTextCharacters(wish.URL)
	if err != nil {
		log.Println("Failed to validate wish URL text string. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
		context.Abort()
		return
	} else if !stringMatch {
		log.Println("Wish URL text string failed validation.")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	// Verify unique wish name in wishlist
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

	// Validate valid URL
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
	db_wish.Price = wish.Price

	// Create user in DB
	record := database.Instance.Create(&db_wish)
	if record.Error != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": record.Error.Error()})
		context.Abort()
		return
	}

	// Save image
	if wish.Image != "" {
		err = SaveWishImage(int(db_wish.ID), wish.Image)
		if err != nil {
			log.Println("Failed to save wish image. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save wish image."})
			context.Abort()
			return
		}
	}

	_, wishes, err := database.GetWishesFromWishlist(wishlist_id_int)
	if err != nil {
		log.Println("Failed to get wishes from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishes from database."})
		context.Abort()
		return
	}

	wishObjects, err := ConvertWishesToWishObjects(wishes, &UserID)
	if err != nil {
		log.Println("Failed to convert wishes to wish objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert wishes to wish objects."})
		context.Abort()
		return
	}

	// Return response
	context.JSON(http.StatusCreated, gin.H{"message": "Wish saved.", "wishes": wishObjects})
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
	wishlistFound, wishlist_id, err := database.GetWishlistFromWish(wish_id_int)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if !wishlistFound {
		log.Println("Failed to get wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist."})
		context.Abort()
		return
	}

	// Verify if collaboration exists
	collaborationStatus, err := database.VerifyWishlistCollaboratorToWishlist(wishlist_id, UserID)
	if err != nil {
		log.Println("Failed to verify wishlist collaborator status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist collaborator status."})
		context.Abort()
		return
	}

	// Verify ownership exists
	MembershipStatus, err := database.VerifyUserOwnershipToWishlist(UserID, wishlist_id)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if !MembershipStatus && !collaborationStatus {
		//context.JSON(http.StatusInternalServerError, gin.H{"error": groupmembershiprecord.Error.Error()})
		context.JSON(http.StatusBadRequest, gin.H{"error": "You are not an owner or collaborator of this wishlist."})
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

	_, wishes, err := database.GetWishesFromWishlist(wishlist_id)
	if err != nil {
		log.Println("Failed to get wishes from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishes from database."})
		context.Abort()
		return
	}

	wishObjects, err := ConvertWishesToWishObjects(wishes, &UserID)
	if err != nil {
		log.Println("Failed to convert wishes to wish objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert wishes to wish objects."})
		context.Abort()
		return
	}

	// Return response
	context.JSON(http.StatusCreated, gin.H{"message": "Wish deleted.", "wishes": wishObjects})

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

	wishlistFound, db_wishlist_id, err := database.GetWishlistFromWish(wish_id_int)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if !wishlistFound {
		log.Println("Failed to get wishlist ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist ID."})
		context.Abort()
		return
	}

	wishlistFound, wishlistObject, err := database.GetWishlistByWishlistID(db_wishlist_id)
	if err != nil {
		log.Println("Failed to get wishlist object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist object."})
		context.Abort()
		return
	} else if !wishlistFound {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find wishlist."})
		context.Abort()
		return
	} else if !*wishlistObject.Claimable {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Wishes in the wishlist are not marked as claimable."})
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

	// Verify if collaboration exists
	collaborationStatus, err := database.VerifyWishlistCollaboratorToWishlist(db_wishlist_id, UserID)
	if err != nil {
		log.Println("Failed to verify wishlist collaborator status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist collaborator status."})
		context.Abort()
		return
	} else if collaborationStatus {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You cannot claim wishes on wishlists where you are a collaborator."})
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
		context.JSON(http.StatusBadRequest, gin.H{"error": "Wish is already claimed."})
		context.Abort()
		return
	}

	db_wishclaim.User = UserID
	db_wishclaim.Wish = wish_id_int

	// Create wish claim
	record := database.Instance.Create(&db_wishclaim)
	if record.Error != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": record.Error.Error()})
		context.Abort()
		return
	}

	if wishclaim.WishlistID != 0 {

		_, wishes, err := database.GetWishesFromWishlist(db_wishlist_id)
		if err != nil {
			log.Println("Failed to get wishes from database. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishes from database."})
			context.Abort()
			return
		}

		wishObjects, err := ConvertWishesToWishObjects(wishes, &UserID)
		if err != nil {
			log.Println("Failed to convert wishes to wish objects. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert wishes to wish objects."})
			context.Abort()
			return
		}

		// Return response
		context.JSON(http.StatusCreated, gin.H{"message": "Wish claimed.", "wishes": wishObjects})
		return

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

	wishlistFound, db_wishlist_id, err := database.GetWishlistFromWish(wish_id_int)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if !wishlistFound {
		log.Println("Failed to get wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist."})
		context.Abort()
		return
	}

	wishlistFound, wishlistObject, err := database.GetWishlistByWishlistID(db_wishlist_id)
	if err != nil {
		log.Println("Failed to get wishlist object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist object."})
		context.Abort()
		return
	} else if !wishlistFound {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find wishlist."})
		context.Abort()
		return
	} else if !*wishlistObject.Claimable {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Wishes in the wishlist are not marked as claimable."})
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

	// Verify if collaboration exists
	collaborationStatus, err := database.VerifyWishlistCollaboratorToWishlist(db_wishlist_id, UserID)
	if err != nil {
		log.Println("Failed to verify wishlist collaborator status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist collaborator status."})
		context.Abort()
		return
	} else if collaborationStatus {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You cannot unclaim wishes on wishlists where you are a collaborator."})
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

		_, wishes, err := database.GetWishesFromWishlist(db_wishlist_id)
		if err != nil {
			log.Println("Failed to get wishes from database. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishes from database."})
			context.Abort()
			return
		}

		wishObjects, err := ConvertWishesToWishObjects(wishes, &UserID)
		if err != nil {
			log.Println("Failed to convert wishes to wish objects. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert wishes to wish objects."})
			context.Abort()
			return
		}

		// Return response
		context.JSON(http.StatusCreated, gin.H{"message": "Wish claimed.", "wishes": wishObjects})
		return

	} else {
		context.JSON(http.StatusCreated, gin.H{"message": "Wish claimed."})
	}
}

func APIUpdateWish(context *gin.Context) {

	// Create wish request
	var wishID = context.Param("wish_id")
	var wish models.WishCreationRequest

	// Bind the incoming request body to the model
	if err := context.ShouldBindJSON(&wish); err != nil {
		// If there is an error binding the request, return a Bad Request response
		log.Println(("Failed to parse request. Error: " + err.Error()))
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Parse wishlist id
	wishIDInt, err := strconv.Atoi(wishID)
	if err != nil {
		log.Println("Failed to parse wish ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wish ID."})
		context.Abort()
		return
	}

	// Get wishlist ID
	wishlistFound, wishlistID, err := database.GetWishlistFromWish(wishIDInt)
	if err != nil {
		log.Println("Failed to get wishlist ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist ID."})
		context.Abort()
		return
	} else if !wishlistFound {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find wishlist."})
		context.Abort()
		return
	}

	// Verify if collaboration exists
	collaborationStatus, err := database.VerifyWishlistCollaboratorToWishlist(wishlistID, userID)
	if err != nil {
		log.Println("Failed to verify wishlist collaborator status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist collaborator status."})
		context.Abort()
		return
	}

	// Verify ownership exists
	MembershipStatus, err := database.VerifyUserOwnershipToWishlist(userID, wishlistID)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if !MembershipStatus && !collaborationStatus {
		//context.JSON(http.StatusInternalServerError, gin.H{"error": groupmembershiprecord.Error.Error()})
		context.JSON(http.StatusBadRequest, gin.H{"error": "You are not an owner or collaborator of this wishlist."})
		context.Abort()
		return
	}

	// Get original wish
	wishFound, wishOriginal, err := database.GetWishByWishID(wishIDInt)
	if err != nil {
		log.Println("Failed to get wish. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wish."})
		context.Abort()
		return
	} else if !wishFound {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find wish."})
		context.Abort()
		return
	}

	// If new wish name, verify name
	if wish.Name != wishOriginal.Name {

		if len(wish.Name) < 5 || wish.Name == "" {
			context.JSON(http.StatusBadRequest, gin.H{"error": "The name of the wish must be five or more letters."})
			context.Abort()
			return
		}

		// Validate wish name format
		stringMatch, requirements, err := utilities.ValidateTextCharacters(wish.Name)
		if err != nil {
			log.Println("Failed to validate wish name text string. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
			context.Abort()
			return
		} else if !stringMatch {
			log.Println("Wish name text string failed validation.")
			context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
			context.Abort()
			return
		}

		unique_wish_name, err := database.VerifyUniqueWishNameinWishlist(wish.Name, wishlistID)
		if err != nil {
			log.Println("Failed to verify wish name. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wish name."})
			context.Abort()
			return
		} else if !unique_wish_name {
			context.JSON(http.StatusBadRequest, gin.H{"error": "There is already a wish with that name in this wishlist."})
			context.Abort()
			return
		}

	}

	if wish.URL != wishOriginal.URL && wishOriginal.URL != "" {

		// Validate wish URL format
		stringMatch, requirements, err := utilities.ValidateTextCharacters(wish.URL)
		if err != nil {
			log.Println("Failed to validate wish URL text string. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
			context.Abort()
			return
		} else if !stringMatch {
			log.Println("Wish URL text string failed validation.")
			context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
			context.Abort()
			return
		}

		domain, scheme, err := parseRawURLFunction(wish.URL)
		if (err != nil || domain == "" || scheme == "") && wish.URL != "" {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL given."})
			context.Abort()
			return
		}
	}

	if wish.Note != wishOriginal.Note && wishOriginal.Note != "" {

		// Validate wish note format
		stringMatch, requirements, err := utilities.ValidateTextCharacters(wish.Note)
		if err != nil {
			log.Println("Failed to validate wish note text string. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
			context.Abort()
			return
		} else if !stringMatch {
			log.Println("Wish note text string failed validation.")
			context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
			context.Abort()
			return
		}

	}

	// Save image
	if wish.Image != "" {
		err = SaveWishImage(int(wishIDInt), wish.Image)
		if err != nil {
			log.Println("Failed to save wish image. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save wish image."})
			context.Abort()
			return
		}
	}

	// Create user in DB
	err = database.UpdateWishValuesInDatabase(wishIDInt, wish.Name, wish.Note, wish.URL, wish.Price)
	if err != nil {
		log.Println("Failed to update wish in database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update wish in database."})
		context.Abort()
		return
	}

	wishFound, wishNew, err := database.GetWishByWishID(wishIDInt)
	if err != nil {
		log.Println("Failed to get wish from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wish from database."})
		context.Abort()
		return
	} else if !wishFound {
		log.Println("Failed to find wish in database. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find wish in database."})
		context.Abort()
		return
	}

	wishObject, err := ConvertWishToWishObject(wishNew, &userID)
	if err != nil {
		log.Println("Failed to convert wish to wish object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert wish to wish object."})
		context.Abort()
		return
	}

	// Return response
	context.JSON(http.StatusCreated, gin.H{"message": "Wish updated.", "wish": wishObject})
}

func APIGetWish(context *gin.Context) {

	// Create wish request
	var wishID = context.Param("wish_id")

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to parse header. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse header."})
		context.Abort()
		return
	}

	// Parse wishlist id
	wishIDInt, err := strconv.Atoi(wishID)
	if err != nil {
		log.Println("Failed to parse wish ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wish ID."})
		context.Abort()
		return
	}

	wishlistFound, wishlistID, err := database.GetWishlistFromWish(wishIDInt)
	if err != nil {
		log.Println("Failed to get wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist."})
		context.Abort()
		return
	} else if !wishlistFound {
		log.Println("Failed to find wishlist. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find wishlist."})
		context.Abort()
		return
	}

	WishlistOwnership, err := database.VerifyUserOwnershipToWishlist(userID, int(wishlistID))
	if err != nil {
		log.Println("Failed to verify wishlist ownership. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist ownership."})
		context.Abort()
		return
	}

	WishlistMembership, err := database.VerifyUserMembershipToGroupmembershipToWishlist(userID, wishlistID)
	if err != nil {
		log.Println("Failed to verify wishlist membership. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist membership."})
		context.Abort()
		return
	}

	if !WishlistOwnership && !WishlistMembership {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You are not a member of, or an owner of this group."})
		context.Abort()
		return
	}

	_, wish, err := database.GetWishByWishID(wishIDInt)
	if err != nil {
		log.Println("Failed to get wish from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wish from database."})
		context.Abort()
		return
	}

	wishObject, err := ConvertWishToWishObject(wish, &userID)
	if err != nil {
		log.Println("Failed to convert wish to wish object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert wish to wish object."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"wish": wishObject, "message": "Wish retrieved."})
}
