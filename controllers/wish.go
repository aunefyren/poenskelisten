package controllers

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetWishesFromWishlist(context *gin.Context) {

	// Create wish request
	var wishlist_id string

	wishlist_id, okay := context.GetQuery("wishlist")
	if !okay {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get wishlist from request."})
		context.Abort()
		return
	}

	// Get user ID
	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Parse wishlist id
	wishlist_id_int, err := uuid.Parse(wishlist_id)
	if err != nil {
		logger.Log.Error("Failed to parse wishlist ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wishlist ID."})
		context.Abort()
		return
	}

	WishlistOwnership, err := database.VerifyUserOwnershipToWishlist(UserID, wishlist_id_int)
	if err != nil {
		logger.Log.Error("Failed to verify ownership of wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership of wishlist."})
		context.Abort()
		return
	}

	WishlistMembership, err := database.VerifyUserMembershipToGroupMembershipToWishlist(UserID, wishlist_id_int)
	if err != nil {
		logger.Log.Error("Failed to verify membership of group. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify membership of group."})
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
		logger.Log.Error("Failed to get wishes from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishes from database."})
		context.Abort()
		return
	}

	wishObjects, err := ConvertWishesToWishObjects(wishes, &UserID)
	if err != nil {
		logger.Log.Error("Failed to convert wishes to wish objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert wishes to wish objects."})
		context.Abort()
		return
	}

	owner_id, err := database.GetWishlistOwner(wishlist_id_int)
	if err != nil {
		logger.Log.Error("Failed to get wishlist owner. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist owner."})
		context.Abort()
		return
	}

	wishlistCollabs, err := database.GetWishlistCollaboratorsFromWishlist(wishlist_id_int)
	if err != nil {
		logger.Log.Error("Failed to get wishlist collaborators. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist collaborators."})
		context.Abort()
		return
	}

	wishlistCollabsIntArray := []uuid.UUID{}
	for _, wishlistCollab := range wishlistCollabs {
		wishlistCollabsIntArray = append(wishlistCollabsIntArray, wishlistCollab.UserID)
	}

	// Sort wishes by creation date
	sort.Slice(wishObjects, func(i, j int) bool {
		return wishObjects[j].UpdatedAt.Before(wishObjects[i].UpdatedAt)
	})

	context.JSON(http.StatusOK, gin.H{
		"owner_id":      owner_id,
		"collaborators": wishlistCollabsIntArray,
		"wishes":        wishObjects, "message": "Wishes retrieved.",
		"currency":         config.ConfigFile.PoenskelistenCurrency,
		"currency_padding": config.ConfigFile.PoenskelistenCurrencyPad,
		"currency_left":    config.ConfigFile.PoenskelistenCurrencyLeft,
	})
}

func ConvertWishToWishObject(wish models.Wish, requestUserID *uuid.UUID) (models.WishObject, error) {

	wishObject := models.WishObject{}

	user_object, err := database.GetUserInformation(wish.OwnerID)
	if err != nil {
		logger.Log.Error("Failed to get information about wish owner for wish'" + wish.ID.String() + "' and user '" + wish.OwnerID.String() + "'. Returning. Error: " + err.Error())
		return models.WishObject{}, err
	}

	wishClaimObject, err := database.GetWishClaimFromWish(wish.ID)
	if err != nil {
		logger.Log.Error("Failed to get wish claims wish'" + wish.ID.String() + "'. Returning. Error: " + err.Error())
		return models.WishObject{}, err
	}

	_, wishlist, err := database.GetWishlistByWishlistID(wish.WishlistID)
	if err != nil {
		logger.Log.Error("Failed to get wishlist for wish'" + wish.ID.String() + "'. Returning. Error: " + err.Error())
		return models.WishObject{}, err
	}

	wishlistOwnerUser, err := database.GetUserInformation(wishlist.OwnerID)
	if err != nil {
		logger.Log.Error("Failed to get information about wishlist owner for wish'" + wish.ID.String() + "' and user '" + wishlist.OwnerID.String() + "'. Returning. Error: " + err.Error())
		return models.WishObject{}, err
	}

	imageExists, err := CheckIfWishImageExists(wish.ID)
	if err != nil {
		logger.Log.Warn("Failed to check if wish'" + wish.ID.String() + "' had image. Setting to false. Error: " + err.Error())
		wishObject.Image = false
	} else if imageExists {
		wishObject.Image = true
	} else {
		wishObject.Image = false
	}

	wishlistCollabs, err := database.GetWishlistCollaboratorsFromWishlist(wish.WishlistID)
	if err != nil {
		logger.Log.Error("Failed to get wishlist collaborator from database. Error: " + err.Error())
		return models.WishObject{}, errors.New("Failed to get wishlist from database.")
	}

	wishlistCollabObjects, err := ConvertWishlistCollaboratorsToWishlistCollaboratorsObjects(wishlistCollabs)
	if err != nil {
		logger.Log.Error("Failed to convert wishlist collaborators to objects. Error: " + err.Error())
		return models.WishObject{}, errors.New("Failed to convert wishlist collaborators to objects.")
	}

	// Purge the reply if the requester is the owner
	if requestUserID != nil {
		if wish.OwnerID == *requestUserID {
			wishClaimObject = []models.WishClaimObject{}
		}

		if wishlistOwnerUser.ID == *requestUserID {
			wishClaimObject = []models.WishClaimObject{}
		}

		for _, wishCollaborator := range wishlistCollabObjects {
			if wishCollaborator.User.ID == *requestUserID {
				wishClaimObject = []models.WishClaimObject{}
			}
		}
	}

	// Purge claim details if claimers are hidden
	if wishlist.HideClaimers != nil && *wishlist.HideClaimers == true {
		newClaimers := []models.WishClaimObject{}
		for _, wishClaim := range wishClaimObject {
			wishClaim.User.CreatedAt = time.Now()
			wishClaim.User.UpdatedAt = time.Now()
			wishClaim.User.Email = nil
			wishClaim.User.FirstName = "Hidden"
			wishClaim.User.LastName = "User"
			newClaimers = append(newClaimers, wishClaim)
		}
		wishClaimObject = newClaimers
	}

	wishObject.CreatedAt = wish.CreatedAt
	wishObject.DeletedAt = wish.DeletedAt
	wishObject.Enabled = wish.Enabled
	wishObject.ID = wish.ID
	wishObject.Name = wish.Name
	wishObject.Note = wish.Note
	wishObject.Owner = user_object
	wishObject.WishlistOwner = wishlistOwnerUser
	wishObject.WishClaim = wishClaimObject
	wishObject.URL = wish.URL
	wishObject.Price = wish.Price
	wishObject.UpdatedAt = wish.UpdatedAt
	wishObject.WishlistID = wish.WishlistID
	wishObject.WishClaimable = *wishlist.Claimable
	wishObject.Collaborators = wishlistCollabObjects
	wishObject.Currency = config.ConfigFile.PoenskelistenCurrency
	wishObject.CurrencyPadding = config.ConfigFile.PoenskelistenCurrencyPad
	wishObject.CurrencyLeft = config.ConfigFile.PoenskelistenCurrencyLeft

	return wishObject, nil

}

func ConvertWishesToWishObjects(wishes []models.Wish, requestUserID *uuid.UUID) ([]models.WishObject, error) {

	wishObjects := []models.WishObject{}

	for _, wish := range wishes {

		wishObject, err := ConvertWishToWishObject(wish, requestUserID)
		if err != nil {
			logger.Log.Warn("Failed to convert wish '" + wish.ID.String() + "' to wish object. Skipping. Error: " + err.Error())
			continue
		}

		wishObjects = append(wishObjects, wishObject)

	}

	return wishObjects, nil

}

func RegisterWish(context *gin.Context) {
	// Create wish request
	var wishlist_id string

	wishlist_id, okay := context.GetQuery("wishlist")
	if !okay {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get wishlist from request."})
		context.Abort()
		return
	}

	var wish models.WishCreationRequest
	var db_wish models.Wish

	if err := context.ShouldBindJSON(&wish); err != nil {
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Trim request input
	wish.Name = strings.TrimSpace(wish.Name)
	wish.Note = strings.TrimSpace(wish.Note)
	wish.URL = strings.TrimSpace(wish.URL)

	// Get user ID
	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	// Parse wishlist id
	wishlist_id_int, err := uuid.Parse(wishlist_id)
	if err != nil {
		logger.Log.Error("Failed to parse wishlist ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wishlist ID."})
		context.Abort()
		return
	}

	// Verify if collaboration exists
	collaborationStatus, err := database.VerifyWishlistCollaboratorToWishlist(wishlist_id_int, UserID)
	if err != nil {
		logger.Log.Error("Failed to verify wishlist collaborator status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist collaborator status."})
		context.Abort()
		return
	}

	// Verify ownership exists
	MembershipStatus, err := database.VerifyUserOwnershipToWishlist(UserID, wishlist_id_int)
	if err != nil {
		logger.Log.Error("Failed to verify ownership of wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership of wishlist."})
		context.Abort()
		return
	} else if !MembershipStatus && !collaborationStatus {
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
		logger.Log.Error("Failed to validate wish name text string. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
		context.Abort()
		return
	} else if !stringMatch {
		logger.Log.Error("Wish name text string failed validation.")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	// Validate wish note format
	stringMatch, requirements, err = utilities.ValidateTextCharacters(wish.Note)
	if err != nil {
		logger.Log.Error("Failed to validate wish note text string. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
		context.Abort()
		return
	} else if !stringMatch {
		logger.Log.Error("Wish note text string failed validation.")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	// Validate wish url format
	stringMatch, requirements, err = utilities.ValidateTextCharacters(wish.URL)
	if err != nil {
		logger.Log.Error("Failed to validate wish URL text string. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
		context.Abort()
		return
	} else if !stringMatch {
		logger.Log.Error("Wish URL text string failed validation.")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	// Verify unique wish name in wishlist
	unique_wish_name, err := database.VerifyUniqueWishNameInWishlist(wish.Name, wishlist_id_int)
	if err != nil {
		logger.Log.Error("Failed to verify unique wishlist name. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify unique wishlist name."})
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

	db_wish.OwnerID = UserID
	db_wish.WishlistID = wishlist_id_int
	db_wish.Name = wish.Name
	db_wish.Note = wish.Note
	db_wish.URL = wish.URL
	db_wish.Price = wish.Price
	db_wish.ID = uuid.New()

	// Create wish in DB
	_, err = database.CreateWishInDB(db_wish)
	if err != nil {
		logger.Log.Error("Failed to create wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create wishlist."})
		context.Abort()
		return
	}

	// Save image
	if wish.Image != "" {
		err = SaveWishImage(db_wish.ID, wish.Image)
		if err != nil {
			logger.Log.Error("Failed to save wish image. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save wish image."})
			context.Abort()
			return
		}
	}

	_, wishes, err := database.GetWishesFromWishlist(wishlist_id_int)
	if err != nil {
		logger.Log.Error("Failed to get wishes from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishes from database."})
		context.Abort()
		return
	}

	wishObjects, err := ConvertWishesToWishObjects(wishes, &UserID)
	if err != nil {
		logger.Log.Error("Failed to convert wishes to wish objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert wishes to wish objects."})
		context.Abort()
		return
	}

	// Sort wishes by creation date
	sort.Slice(wishObjects, func(i, j int) bool {
		return wishObjects[j].CreatedAt.Before(wishObjects[i].CreatedAt)
	})

	// Return response
	context.JSON(http.StatusCreated, gin.H{"message": "Wish saved.", "wishes": wishObjects})
}

func DeleteWish(context *gin.Context) {

	// Create wish request
	var wish_id = context.Param("wish_id")

	// Get user ID
	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	// Parse wish id
	wish_id_int, err := uuid.Parse(wish_id)
	if err != nil {
		logger.Log.Error("Failed to parse wish ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wish ID."})
		context.Abort()
		return
	}

	wish, err := database.GetWishByWishID(wish_id_int)
	if err != nil {
		logger.Log.Error("Failed to get wish. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get wish."})
		context.Abort()
		return
	} else if wish == nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find wish."})
		context.Abort()
		return
	}

	wishObject, err := ConvertWishToWishObject(*wish, nil)
	if err != nil {
		logger.Log.Error("Failed to convert wish to wish object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert wish to wish object."})
		context.Abort()
		return
	}

	// Verify if collaboration exists
	collaborationStatus, err := database.VerifyWishlistCollaboratorToWishlist(wish.WishlistID, UserID)
	if err != nil {
		logger.Log.Error("Failed to verify wishlist collaborator status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist collaborator status."})
		context.Abort()
		return
	}

	// Verify ownership exists
	MembershipStatus, err := database.VerifyUserOwnershipToWishlist(UserID, wish.WishlistID)
	if err != nil {
		logger.Log.Error("Failed to verify ownership of wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership of wishlist."})
		context.Abort()
		return
	} else if !MembershipStatus && !collaborationStatus {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You are not an owner or collaborator of this wishlist."})
		context.Abort()
		return
	}

	wish.Enabled = false

	// delete wish
	*wish, err = database.UpdateWishInDB(*wish)
	if err != nil {
		logger.Log.Error("Failed to delete wish. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete wish."})
		context.Abort()
		return
	}

	_, wishes, err := database.GetWishesFromWishlist(wish.WishlistID)
	if err != nil {
		logger.Log.Error("Failed to get wishes from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishes from database."})
		context.Abort()
		return
	}

	wishObjects, err := ConvertWishesToWishObjects(wishes, &UserID)
	if err != nil {
		logger.Log.Error("Failed to convert wishes to wish objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert wishes to wish objects."})
		context.Abort()
		return
	}

	// Sort wishes by creation date
	sort.Slice(wishObjects, func(i, j int) bool {
		return wishObjects[j].CreatedAt.Before(wishObjects[i].CreatedAt)
	})

	// Return response
	context.JSON(http.StatusCreated, gin.H{"message": "Wish deleted.", "wishes": wishObjects})

	if wishObject.WishClaimable {
		for _, wishClaim := range wishObject.WishClaim {
			if wishClaim.Enabled {
				wishlist, err := database.GetWishlist(wish.WishlistID)
				if err != nil {
					logger.Log.Error("Failed to get wishlist. Error: " + err.Error())
					return
				}

				wishlistObject, err := ConvertWishlistToWishlistObject(wishlist, nil)
				if err != nil {
					logger.Log.Error("Failed to convert wishlist to wishlist object. Error: " + err.Error())
					return
				}

				wishClaimUser, err := database.GetAllUserInformation(wishClaim.User.ID)
				if err != nil {
					logger.Log.Error("Failed to get user object for user. Error: " + err.Error())
					return
				}

				utilities.SendSMTPDeletedClaimedWish(wishClaimUser, wishObject, wishlistObject)
			}
		}
	}
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
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Get user ID
	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	// Parse wishlist id
	wish_id_int, err := uuid.Parse(wish_id)
	if err != nil {
		logger.Log.Error("Failed to parse wish ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wish ID."})
		context.Abort()
		return
	}

	db_wishlist_id, err := database.GetWishlistIDFromWish(wish_id_int)
	if err != nil {
		logger.Log.Error("Failed to get wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist."})
		context.Abort()
		return
	} else if db_wishlist_id == nil {
		context.JSON(http.StatusNotFound, gin.H{"error": "Failed to get wishlist ID."})
		context.Abort()
		return
	}

	wishlistFound, wishlistObject, err := database.GetWishlistByWishlistID(*db_wishlist_id)
	if err != nil {
		logger.Log.Error("Failed to get wishlist object. Error: " + err.Error())
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

	WishlistOwnership, err := database.VerifyUserOwnershipToWishlist(UserID, *db_wishlist_id)
	if err != nil {
		logger.Log.Error("Failed to verify ownership of wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership of wishlist."})
		context.Abort()
		return
	}

	WishlistMembership, err := database.VerifyUserMembershipToGroupMembershipToWishlist(UserID, *db_wishlist_id)
	if err != nil {
		logger.Log.Error("Failed to verify membership to wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify membership to wishlist."})
		context.Abort()
		return
	}

	if !WishlistOwnership && !WishlistMembership {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You are not a member of, or an owner of this wishlist group."})
		context.Abort()
		return
	}

	// Verify if collaboration exists
	collaborationStatus, err := database.VerifyWishlistCollaboratorToWishlist(*db_wishlist_id, UserID)
	if err != nil {
		logger.Log.Error("Failed to verify wishlist collaborator status. Error: " + err.Error())
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
		logger.Log.Error("Failed to verify ownership of wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership of wishlist."})
		context.Abort()
		return
	} else if MembershipStatus {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You cannot claim your own wish."})
		context.Abort()
		return
	}

	// Verify if wish is claimed or not
	ClaimStatus, err := database.VerifyWishIsClaimed(wish_id_int)
	if err != nil {
		logger.Log.Error("Failed to verify claim status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify claim status."})
		context.Abort()
		return
	} else if ClaimStatus {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Wish is already claimed."})
		context.Abort()
		return
	}

	db_wishclaim.UserID = UserID
	db_wishclaim.WishID = wish_id_int
	db_wishclaim.ID = uuid.New()

	// Create wish claim
	_, err = database.CreateWishClaimInDB(db_wishclaim)
	if err != nil {
		logger.Log.Error("Failed to create claim. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create claim."})
		context.Abort()
		return
	}

	if wishclaim.WishlistID != nil {

		_, wishes, err := database.GetWishesFromWishlist(*db_wishlist_id)
		if err != nil {
			logger.Log.Error("Failed to get wishes from database. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishes from database."})
			context.Abort()
			return
		}

		wishObjects, err := ConvertWishesToWishObjects(wishes, &UserID)
		if err != nil {
			logger.Log.Error("Failed to convert wishes to wish objects. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert wishes to wish objects."})
			context.Abort()
			return
		}

		// Sort wishes by creation date
		sort.Slice(wishObjects, func(i, j int) bool {
			return wishObjects[j].CreatedAt.Before(wishObjects[i].CreatedAt)
		})

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
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Get user ID
	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	// Parse wishlist id
	wish_id_int, err := uuid.Parse(wish_id)
	if err != nil {
		logger.Log.Error("Failed to parse wish ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wish ID."})
		context.Abort()
		return
	}

	db_wishlist_id, err := database.GetWishlistIDFromWish(wish_id_int)
	if err != nil {
		logger.Log.Error("Failed to get wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist."})
		context.Abort()
		return
	} else if db_wishlist_id == nil {
		context.JSON(http.StatusNotFound, gin.H{"error": "Failed to get wishlist."})
		context.Abort()
		return
	}

	wishlistFound, wishlistObject, err := database.GetWishlistByWishlistID(*db_wishlist_id)
	if err != nil {
		logger.Log.Error("Failed to get wishlist object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist object."})
		context.Abort()
		return
	} else if !wishlistFound {
		context.JSON(http.StatusNotFound, gin.H{"error": "Failed to find wishlist."})
		context.Abort()
		return
	} else if !*wishlistObject.Claimable {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Wishes in the wishlist are not marked as claimable."})
		context.Abort()
		return
	}

	WishlistOwnership, err := database.VerifyUserOwnershipToWishlist(UserID, *db_wishlist_id)
	if err != nil {
		logger.Log.Error("Failed to verify ownership of wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership of wishlist."})
		context.Abort()
		return
	}

	WishlistMembership, err := database.VerifyUserMembershipToGroupMembershipToWishlist(UserID, *db_wishlist_id)
	if err != nil {
		logger.Log.Error("Failed to verify membership to wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify membership to wishlist."})
		context.Abort()
		return
	}

	if !WishlistOwnership && !WishlistMembership {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You are not a member of, or an owner of this wishlist group."})
		context.Abort()
		return
	}

	// Verify if collaboration exists
	collaborationStatus, err := database.VerifyWishlistCollaboratorToWishlist(*db_wishlist_id, UserID)
	if err != nil {
		logger.Log.Error("Failed to verify wishlist collaborator status. Error: " + err.Error())
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
		logger.Log.Error("Failed to verify ownership of wish. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership of wish."})
		context.Abort()
		return
	} else if !OwnershipStatus {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You cannot unclaim a wish you haven't claimed."})
		context.Abort()
		return
	}

	// Delete the membership
	err = database.DeleteWishClaimByUserAndWish(wish_id_int, UserID)
	if err != nil {
		logger.Log.Error("Failed to delete claim. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to delete claim."})
		context.Abort()
		return
	}

	if wishclaim.WishlistID != nil {

		_, wishes, err := database.GetWishesFromWishlist(*db_wishlist_id)
		if err != nil {
			logger.Log.Error("Failed to get wishes from database. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishes from database."})
			context.Abort()
			return
		}

		wishObjects, err := ConvertWishesToWishObjects(wishes, &UserID)
		if err != nil {
			logger.Log.Error("Failed to convert wishes to wish objects. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert wishes to wish objects."})
			context.Abort()
			return
		}

		// Sort wishes by creation date
		sort.Slice(wishObjects, func(i, j int) bool {
			return wishObjects[j].CreatedAt.Before(wishObjects[i].CreatedAt)
		})

		// Return response
		context.JSON(http.StatusOK, gin.H{"message": "Wish unclaimed.", "wishes": wishObjects})
		return

	} else {
		context.JSON(http.StatusOK, gin.H{"message": "Wish unclaimed."})
	}
}

func APIUpdateWish(context *gin.Context) {

	// Create wish request
	var wishIDString = context.Param("wish_id")
	var wish models.WishUpdateRequest

	// Bind the incoming request body to the model
	if err := context.ShouldBindJSON(&wish); err != nil {
		// If there is an error binding the request, return a Bad Request response
		logger.Log.Error(("Failed to parse request. Error: " + err.Error()))
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Trim request input
	wish.Name = strings.TrimSpace(wish.Name)
	wish.Note = strings.TrimSpace(wish.Note)
	wish.URL = strings.TrimSpace(wish.URL)

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	// Parse wishlist id
	wishID, err := uuid.Parse(wishIDString)
	if err != nil {
		logger.Log.Error("Failed to parse wish ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wish ID."})
		context.Abort()
		return
	}

	// Get original wish
	wishOriginal, err := database.GetWishByWishID(wishID)
	if err != nil {
		logger.Log.Error("Failed to get wish. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wish."})
		context.Abort()
		return
	} else if wishOriginal == nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find wish."})
		context.Abort()
		return
	}

	// Verify if collaboration exists
	collaborationStatus, err := database.VerifyWishlistCollaboratorToWishlist(wishOriginal.WishlistID, userID)
	if err != nil {
		logger.Log.Error("Failed to verify wishlist collaborator status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist collaborator status."})
		context.Abort()
		return
	}

	// Verify ownership exists
	MembershipStatus, err := database.VerifyUserOwnershipToWishlist(userID, wishOriginal.WishlistID)
	if err != nil {
		logger.Log.Error("Failed to verify ownership of wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership of wishlist."})
		context.Abort()
		return
	} else if !MembershipStatus && !collaborationStatus {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You are not an owner or collaborator of this wishlist."})
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
			logger.Log.Error("Failed to validate wish name text string. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
			context.Abort()
			return
		} else if !stringMatch {
			logger.Log.Error("Wish name text string failed validation.")
			context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
			context.Abort()
			return
		}

		unique_wish_name, err := database.VerifyUniqueWishNameInWishlist(wish.Name, wishOriginal.WishlistID)
		if err != nil {
			logger.Log.Error("Failed to verify wish name. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wish name."})
			context.Abort()
			return
		} else if !unique_wish_name {
			context.JSON(http.StatusBadRequest, gin.H{"error": "There is already a wish with that name in this wishlist."})
			context.Abort()
			return
		}

		wishOriginal.Name = strings.TrimSpace(wish.Name)
	}

	if wish.URL != wishOriginal.URL {
		// Validate wish URL format
		stringMatch, requirements, err := utilities.ValidateTextCharacters(wish.URL)
		if err != nil {
			logger.Log.Error("Failed to validate wish URL text string. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
			context.Abort()
			return
		} else if !stringMatch {
			logger.Log.Error("Wish URL text string failed validation.")
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

		wishOriginal.URL = strings.TrimSpace(wish.URL)
	}

	if wish.Note != wishOriginal.Note {
		// Validate wish note format
		stringMatch, requirements, err := utilities.ValidateTextCharacters(wish.Note)
		if err != nil {
			logger.Log.Error("Failed to validate wish note text string. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
			context.Abort()
			return
		} else if !stringMatch {
			logger.Log.Error("Wish note text string failed validation.")
			context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
			context.Abort()
			return
		}

		wishOriginal.Note = strings.TrimSpace(wish.Note)
	}

	if wish.Price != wishOriginal.Price {
		wishOriginal.Price = wish.Price
	}

	// Save image
	if wish.Image != "" && !wish.ImageDelete {
		err = SaveWishImage(wishID, wish.Image)
		if err != nil {
			logger.Log.Error("Failed to save wish image. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save wish image."})
			context.Abort()
			return
		}
	} else if wish.ImageDelete {
		err = DeleteWishImage(wishID)
		if err != nil {
			logger.Log.Error("Failed to delete wish image. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deletes wish image."})
			context.Abort()
			return
		}
	}

	*wishOriginal, err = database.UpdateWishInDB(*wishOriginal)
	if err != nil {
		logger.Log.Error("Failed to update wish in database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update wish in database."})
		context.Abort()
		return
	}

	wishObject, err := ConvertWishToWishObject(*wishOriginal, &userID)
	if err != nil {
		logger.Log.Error("Failed to convert wish to wish object. Error: " + err.Error())
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
		logger.Log.Error("Failed to parse header. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse header."})
		context.Abort()
		return
	}

	// Parse wishlist id
	wishIDInt, err := uuid.Parse(wishID)
	if err != nil {
		logger.Log.Error("Failed to parse wish ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wish ID."})
		context.Abort()
		return
	}

	wishlistID, err := database.GetWishlistIDFromWish(wishIDInt)
	if err != nil {
		logger.Log.Error("Failed to get wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist."})
		context.Abort()
		return
	} else if wishlistID == nil {
		logger.Log.Error("Failed to find wishlist. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find wishlist."})
		context.Abort()
		return
	}

	WishlistOwnership, err := database.VerifyUserOwnershipToWishlist(userID, *wishlistID)
	if err != nil {
		logger.Log.Error("Failed to verify wishlist ownership. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist ownership."})
		context.Abort()
		return
	}

	WishlistMembership, err := database.VerifyUserMembershipToGroupMembershipToWishlist(userID, *wishlistID)
	if err != nil {
		logger.Log.Error("Failed to verify wishlist membership. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist membership."})
		context.Abort()
		return
	}

	if !WishlistOwnership && !WishlistMembership {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You are not a member of, or an owner of this group."})
		context.Abort()
		return
	}

	wish, err := database.GetWishByWishID(wishIDInt)
	if err != nil {
		logger.Log.Error("Failed to get wish from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wish from database."})
		context.Abort()
		return
	} else if wish == nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find wish in the database."})
		context.Abort()
		return
	}

	wishObject, err := ConvertWishToWishObject(*wish, &userID)
	if err != nil {
		logger.Log.Error("Failed to convert wish to wish object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert wish to wish object."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"wish": wishObject, "message": "Wish retrieved."})
}
