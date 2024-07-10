package controllers

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RegisterWishlist(context *gin.Context) {
	// Create wishlist request
	var wishlist models.WishlistCreationRequest
	var wishlistdb models.Wishlist
	var now = time.Now()

	if err := context.ShouldBindJSON(&wishlist); err != nil {
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Get queries
	groupContextIDString, groupContextIDOkay := context.GetQuery("groupContextID")

	// Trim request input
	wishlist.Name = strings.TrimSpace(wishlist.Name)
	wishlist.Description = strings.TrimSpace(wishlist.Description)

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	if wishlist.Groups != nil {
		for _, groupID := range *wishlist.Groups {
			MembershipStatus, err := database.VerifyUserMembershipToGroup(userID, groupID)
			if err != nil {
				logger.Log.Error("Failed to verify membership to group. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify membership to group."})
				context.Abort()
				return
			} else if !MembershipStatus {
				context.JSON(http.StatusBadRequest, gin.H{"error": "You are not a member of one/more groups."})
				context.Abort()
				return
			}
		}
	}

	if len(wishlist.Name) < 5 || wishlist.Name == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "The name of the wishlist must be five or more letters."})
		context.Abort()
		return
	}

	// Validate wishlist name format
	stringMatch, requirements, err := utilities.ValidateTextCharacters(wishlist.Name)
	if err != nil {
		logger.Log.Error("Failed to validate wishlist name text string. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
		context.Abort()
		return
	} else if !stringMatch {
		logger.Log.Error("Wishlist name text string failed validation.")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	// Validate wishlist description format
	stringMatch, requirements, err = utilities.ValidateTextCharacters(wishlist.Description)
	if err != nil {
		logger.Log.Error("Failed to validate description name text string. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
		context.Abort()
		return
	} else if !stringMatch {
		logger.Log.Error("description name text string failed validation.")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	wishlistdb.Expires = &wishlist.Expires

	if wishlist.Date != nil {
		newDate, err := time.Parse("2006-01-02T15:04:05.000Z", *wishlist.Date)
		if err != nil && *wishlistdb.Expires {
			logger.Log.Error("Failed to parse date time. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse date time."})
			context.Abort()
			return
		}
		wishlistdb.Date = &newDate

		if now.After(*wishlistdb.Date) && *wishlistdb.Expires {
			context.JSON(http.StatusBadRequest, gin.H{"error": "The date of the wishlist must be in the future."})
			context.Abort()
			return
		}
	}

	unique_wish_name, err := database.VerifyUniqueWishlistNameForUser(wishlist.Name, userID)
	if err != nil {
		logger.Log.Error("Failed to verify unique wishlist name. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify unique wishlist name."})
		context.Abort()
		return
	} else if !unique_wish_name {
		context.JSON(http.StatusBadRequest, gin.H{"error": "There is already a wishlist with that name on your profile."})
		context.Abort()
		return
	}

	// Finalize wishlist object
	wishlistdb.OwnerID = userID
	wishlistdb.Description = wishlist.Description
	wishlistdb.Name = wishlist.Name
	wishlistdb.Claimable = &wishlist.Claimable
	wishlistdb.HideClaimers = &wishlist.HideClaimers
	wishlistdb.ID = uuid.New()
	wishlistdb.Public = &wishlist.Public
	wishlistdb.PublicHash = uuid.New()

	if *wishlistdb.Public && *wishlistdb.Claimable {
		context.JSON(http.StatusBadRequest, gin.H{"error": "A wishlist cannot have claimable wishes and be public to users without accounts."})
		context.Abort()
		return
	}

	// Create wishlist in DB
	wishlistdb, err = database.CreateWishlistInDB(wishlistdb)
	if err != nil {
		logger.Log.Error("Failed to create wishlist in database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create wishlist in database."})
		context.Abort()
		return
	}

	var wishlists_with_users []models.WishlistUser

	// If a group was referenced, create the wishlist membership
	if wishlist.Groups != nil {
		for _, groupID := range *wishlist.Groups {
			var wishlistMembershipDB models.WishlistMembership
			wishlistMembershipDB.GroupID = groupID
			wishlistMembershipDB.WishlistID = wishlistdb.ID
			wishlistMembershipDB.ID = uuid.New()

			// Add group membership to database
			_, err := database.CreateWishlistMembershipInDB(wishlistMembershipDB)
			if err != nil {
				logger.Log.Error("Failed to create membership to wishlist. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create membership to wishlist."})
				context.Abort()
				return
			}
		}
	}

	if groupContextIDOkay {
		groupContextID, err := uuid.Parse(groupContextIDString)
		if err != nil {
			logger.Log.Error("Failed to parse group context ID. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse group context ID."})
			context.Abort()
			return
		}

		wishlists_with_users, err = GetWishlistObjectsFromGroup(groupContextID, userID)
		if err != nil {
			logger.Log.Error("Failed to get wishlist objects from group. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist objects from group."})
			context.Abort()
			return
		}
	} else {
		wishlists_with_users, err = GetWishlistObjects(userID)
		if err != nil {
			logger.Log.Error("Failed to get wishlist objects. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist objects."})
			context.Abort()
			return
		}
	}

	// Sort wishlists by creation date
	sort.Slice(wishlists_with_users, func(i, j int) bool {
		return wishlists_with_users[j].CreatedAt.Before(wishlists_with_users[i].CreatedAt)
	})

	context.JSON(http.StatusCreated, gin.H{"message": "Wishlist created.", "wishlists": wishlists_with_users})
}

func DeleteWishlist(context *gin.Context) {
	var wishlistObjects = []models.WishlistUser{}
	var wishlist = context.Param("wishlist_id")

	// Get user ID
	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	// Parse wishlist id
	wishlist_id_int, err := uuid.Parse(wishlist)
	if err != nil {
		logger.Log.Error("Failed to parse wishlist ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wishlist ID."})
		context.Abort()
		return
	}

	// Verify wishlist owner
	MembershipStatus, err := database.VerifyUserOwnershipToWishlist(UserID, wishlist_id_int)
	if err != nil {
		logger.Log.Error("Failed to verify ownership of wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership of wishlist."})
		context.Abort()
		return
	} else if !MembershipStatus {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You are not the owner of this wishlist."})
		context.Abort()
		return
	}

	err = database.DeleteWishlist(wishlist_id_int)
	if err != nil {
		logger.Log.Error("Failed to delete wishlist. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to delete wishlist."})
		context.Abort()
		return
	}

	group_id, okay := context.GetQuery("group")
	if !okay {
		wishlistObjects, err = GetWishlistObjects(UserID)
		if err != nil {
			logger.Log.Error("Failed to get wishlist objects for user. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist objects for user."})
			context.Abort()
			return
		}
	} else {
		// Parse group id
		group_id_int, err := uuid.Parse(group_id)
		if err != nil {
			logger.Log.Error("Failed to parse group ID. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse group ID."})
			context.Abort()
			return
		}

		// Verify membership to group exists
		MembershipStatus, err := database.VerifyUserMembershipToGroup(UserID, group_id_int)
		if err != nil {
			logger.Log.Error("Failed to verify membership to group. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify membership to group."})
			context.Abort()
			return
		} else if !MembershipStatus {
			context.JSON(http.StatusBadRequest, gin.H{"error": "You are not a member of this group."})
			context.Abort()
			return
		}

		wishlistObjects, err = GetWishlistObjectsFromGroup(group_id_int, UserID)
		if err != nil {
			logger.Log.Error("Failed to get wishlists for group. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get wishlists for group."})
			context.Abort()
			return
		}
	}

	// Sort wishlists by creation date
	sort.Slice(wishlistObjects, func(i, j int) bool {
		return wishlistObjects[j].CreatedAt.Before(wishlistObjects[i].CreatedAt)
	})

	context.JSON(http.StatusOK, gin.H{"wishlists": wishlistObjects, "message": "Wishlist deleted."})

}

func GetWishlistObjectsFromGroup(group_id uuid.UUID, RequestUserID uuid.UUID) (wishlistObjects []models.WishlistUser, err error) {
	wishlists, err := database.GetWishlistsFromGroup(group_id)
	if err != nil {
		return []models.WishlistUser{}, err
	}

	wishlistObjects, err = ConvertWishlistsToWishlistObjects(wishlists, &RequestUserID)
	if err != nil {
		logger.Log.Error("Failed to convert wishlists to objects. Returning. Error: " + err.Error())
		return wishlistObjects, errors.New("Failed to convert wishlists to objects.")
	}

	return wishlistObjects, nil
}

func GetWishlist(context *gin.Context) {

	// Create wishlist request
	var wishlist_id = context.Param("wishlist_id")

	// Get configuration
	configFile, err := config.GetConfig()
	if err != nil {
		logger.Log.Error("Failed to get config file. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get config file."})
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

	// parse wishlist id
	wishlist_id_int, err := uuid.Parse(wishlist_id)
	if err != nil {
		logger.Log.Error("Failed to parse wishlist ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wishlist ID."})
		context.Abort()
		return
	}

	WishlistOwnership, err := database.VerifyUserOwnershipToWishlist(UserID, wishlist_id_int)
	if err != nil {
		logger.Log.Error("Failed to verify ownership of group. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership of group."})
		context.Abort()
		return
	}

	WishlistMembership, err := database.VerifyUserMembershipToGroupMembershipToWishlist(UserID, wishlist_id_int)
	if err != nil {
		logger.Log.Error("Failed to verify membership to group. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify membership to group."})
		context.Abort()
		return
	}

	if !WishlistOwnership && !WishlistMembership {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You are not a member of, or an owner of this group."})
		context.Abort()
		return
	}

	wishlist_with_user, err := GetWishlistObject(wishlist_id_int, UserID)
	if err != nil {
		logger.Log.Error("Failed to get wishlist object. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get wishlist object."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"wishlist": wishlist_with_user, "message": "Wishlist retrieved.", "public_url": configFile.PoenskelistenExternalURL})

}

func GetWishlistObject(WishlistID uuid.UUID, RequestUserID uuid.UUID) (wishlistObject models.WishlistUser, err error) {
	wishlist, err := database.GetWishlist(WishlistID)
	if err != nil {
		logger.Log.Error("Failed to get wishlist '" + WishlistID.String() + "' from DB. Returning. Error: " + err.Error())
		return models.WishlistUser{}, err
	}

	wishlistObject, err = ConvertWishlistToWishlistObject(wishlist, &RequestUserID)
	if err != nil {
		logger.Log.Error("Failed to convert wishlist '" + WishlistID.String() + "' to object. Returning. Error: " + err.Error())
		return models.WishlistUser{}, errors.New("Failed to convert wishlist '" + WishlistID.String() + "' to object.")
	}

	return
}

func GetWishlists(context *gin.Context) {
	var group_id string
	var wishlistObjects = []models.WishlistUser{}

	// Get user ID
	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	group_id, okay := context.GetQuery("group")
	if !okay {
		wishlistObjects, err = GetAllWishlistObjects(UserID)
		if err != nil {
			logger.Log.Error("Failed to get wishlist objects for user. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist objects for user."})
			context.Abort()
			return
		}

	} else {
		// Parse group id
		group_id_int, err := uuid.Parse(group_id)
		if err != nil {
			logger.Log.Error("Failed to parse group ID. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse group ID."})
			context.Abort()
			return
		}

		// Verify membership to group exists
		MembershipStatus, err := database.VerifyUserMembershipToGroup(UserID, group_id_int)
		if err != nil {
			logger.Log.Error("Failed to verify membership to group. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify membership to group."})
			context.Abort()
			return
		} else if !MembershipStatus {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "You are not a member of this group."})
			context.Abort()
			return
		}

		wishlistObjects, err = GetWishlistObjectsFromGroup(group_id_int, UserID)
		if err != nil {
			logger.Log.Error("Failed to get wishlists for group. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get wishlists for group."})
			context.Abort()
			return
		}
	}

	ownedString, ownedOkay := context.GetQuery("owned")
	if ownedOkay && ownedString == "true" {
		newWishlistList := []models.WishlistUser{}
		for _, wishlistObject := range wishlistObjects {
			if wishlistObject.Owner.ID == UserID {
				newWishlistList = append(newWishlistList, wishlistObject)
			} else {
				for _, collaborator := range wishlistObject.Collaborators {
					if collaborator.User.ID == UserID {
						newWishlistList = append(newWishlistList, wishlistObject)
						break
					}
				}
			}
		}
		wishlistObjects = newWishlistList
	}

	notAMemberOfGroupIDString, notAMemberOfGroupIDOkay := context.GetQuery("notAMemberOfGroupID")
	if notAMemberOfGroupIDOkay {
		notAMemberOfGroupID, err := uuid.Parse(notAMemberOfGroupIDString)
		if err != nil {
			logger.Log.Error("Failed to parse group ID. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse group ID."})
			context.Abort()
			return
		}

		groupWishlists, err := database.GetWishlistsFromGroup(notAMemberOfGroupID)
		if err != nil {
			logger.Log.Error("Failed to get wishlists from group. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlists from group."})
			context.Abort()
			return
		}

		newWishlistList := []models.WishlistUser{}
		for _, wishlistObject := range wishlistObjects {
			found := false
			for _, groupWishlist := range groupWishlists {
				if wishlistObject.ID == groupWishlist.ID {
					found = true
					break
				}
			}
			if !found {
				newWishlistList = append(newWishlistList, wishlistObject)
			}
		}
		wishlistObjects = newWishlistList
	}

	// Sort wishlists by creation date
	sort.Slice(wishlistObjects, func(i, j int) bool {
		return wishlistObjects[j].WishUpdatedAt.Before(wishlistObjects[i].WishUpdatedAt)
	})

	// Respect expired parameter
	expiredString, okay := context.GetQuery("expired")
	if okay && strings.ToLower(expiredString) == "true" {
		newWishlists := []models.WishlistUser{}
		for _, wishlistObject := range wishlistObjects {
			if wishlistObject.Expires != nil && wishlistObject.Date != nil && *wishlistObject.Expires && wishlistObject.Date.Before(time.Now()) {
				newWishlists = append(newWishlists, wishlistObject)
			}
		}
		wishlistObjects = newWishlists
	} else if okay && strings.ToLower(expiredString) == "false" {
		newWishlists := []models.WishlistUser{}
		for _, wishlistObject := range wishlistObjects {
			if wishlistObject.Expires == nil || wishlistObject.Date == nil || !*wishlistObject.Expires || (*wishlistObject.Expires && wishlistObject.Date.After(time.Now())) {
				newWishlists = append(newWishlists, wishlistObject)
			}
		}
		wishlistObjects = newWishlists
	}

	topString, okay := context.GetQuery("top")
	if okay {
		topInt, err := strconv.Atoi(topString)
		if err == nil && topInt > 0 && (len(wishlistObjects)-1) > topInt {
			wishlistObjects = wishlistObjects[0:topInt]
		}
	}

	context.JSON(http.StatusOK, gin.H{"wishlists": wishlistObjects, "message": "Wishlists retrieved."})
}

// Return wishlists you either own or are a collaborator of
func GetWishlistObjects(UserID uuid.UUID) (wishlistObjects []models.WishlistUser, err error) {
	wishlistObjects = []models.WishlistUser{}

	wishlists, err := database.GetOwnedWishlists(UserID)
	if err != nil {
		logger.Log.Error("Failed to get owned wishlists for user '" + UserID.String() + "'. Returning. Error: " + err.Error())
		return wishlistObjects, errors.New("Failed to get owned wishlists for user '" + UserID.String() + "'.")
	}

	wishlistsThroughCollab, err := database.GetWishlistsByUserIDThroughWishlistCollaborations(UserID)
	if err != nil {
		logger.Log.Error("Failed to get collaboration wishlists for user '" + UserID.String() + "'. Returning. Error: " + err.Error())
		return wishlistObjects, errors.New("Failed to get collaboration wishlists for user '" + UserID.String() + "'.")
	}

	wishlists = append(wishlists, wishlistsThroughCollab...)

	wishlistObjects, err = ConvertWishlistsToWishlistObjects(wishlists, &UserID)
	if err != nil {
		logger.Log.Error("Failed to convert wishlists to objects. Returning. Error: " + err.Error())
		return wishlistObjects, errors.New("Failed to convert wishlists to objects.")
	}

	return wishlistObjects, nil
}

// Return any wishlist you have access to
func GetAllWishlistObjects(UserID uuid.UUID) (wishlistObjects []models.WishlistUser, err error) {
	err = nil
	wishlistObjects = []models.WishlistUser{}
	wishlists := []models.Wishlist{}
	/*
		wishlists, err := database.GetOwnedWishlists(UserID)
		if err != nil {
			logger.Log.Error("Failed to get owned wishlists for user '" + UserID.String() + "'. Returning. Error: " + err.Error())
			return wishlistObjects, errors.New("Failed to get owned wishlists for user '" + UserID.String() + "'.")
		}

		wishlistsThroughCollaborations, err := database.GetWishlistsByUserIDThroughWishlistCollaborations(UserID)
		if err != nil {
			logger.Log.Error("Failed to get collaboration wishlists for user '" + UserID.String() + "'. Returning. Error: " + err.Error())
			return wishlistObjects, errors.New("Failed to get collaboration wishlists for user '" + UserID.String() + "'.")
		}

		for _, wishlistThroughCollaboration := range wishlistsThroughCollaborations {
			wishlists = append(wishlists, wishlistThroughCollaboration)
		}*/

	wishlistsThroughMemberships, err := database.GetWishlistsByUserIDThroughWishlistMemberships(UserID)
	if err != nil {
		logger.Log.Error("Failed to get membership wishlists for user '" + UserID.String() + "'. Returning. Error: " + err.Error())
		return wishlistObjects, errors.New("Failed to get membership wishlists for user '" + UserID.String() + "'.")
	}

	for _, wishlistThroughMembership := range wishlistsThroughMemberships {
		wishlists = append(wishlists, wishlistThroughMembership)
	}

	wishlistObjects, err = ConvertWishlistsToWishlistObjects(wishlists, &UserID)
	if err != nil {
		logger.Log.Error("Failed to convert wishlists to objects. Returning. Error: " + err.Error())
		return wishlistObjects, errors.New("Failed to convert wishlists to objects.")
	}

	return wishlistObjects, nil
}

func JoinWishlist(context *gin.Context) {
	// Create groupmembership request
	var wishlist_id = context.Param("wishlist_id")
	var wishlistmembership models.WishlistMembershipCreationRequest

	if err := context.ShouldBindJSON(&wishlistmembership); err != nil {
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	if len(wishlistmembership.Groups) < 1 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You must provide one or more groups."})
		context.Abort()
		return
	}

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

	for _, Group := range wishlistmembership.Groups {

		var wishlistmembershipdb models.WishlistMembership
		wishlistmembershipdb.GroupID = Group

		// Verify user exists
		_, err := database.GetGroupInformation(Group)
		if err != nil {
			logger.Log.Error("Failed to get group. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get group."})
			context.Abort()
			return
		}

		// Verify membership doesnt exist
		MembershipStatus, err := database.VerifyGroupMembershipToWishlist(wishlist_id_int, Group)
		if err != nil {
			logger.Log.Error("Failed to verify membership to group. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify membership to group."})
			context.Abort()
			return
		} else if MembershipStatus {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Wishlist membership already exists."})
			context.Abort()
			return
		}

		// Verify wishlist is owned by requester
		wishlistOwned, err := database.VerifyUserOwnershipToWishlist(UserID, wishlist_id_int)
		if err != nil {
			logger.Log.Error("Failed to verify ownership to wishlist. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership to wishlist."})
			context.Abort()
			return
		} else if !wishlistOwned {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Only owners can edit their wishlist."})
			context.Abort()
			return
		}

		wishlistmembershipdb.WishlistID = wishlist_id_int
		wishlistmembershipdb.ID = uuid.New()

		// Add group membership to database
		record := database.Instance.Create(&wishlistmembershipdb)
		if record.Error != nil {
			logger.Log.Error("Failed to create membership. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create membership."})
			context.Abort()
			return
		}

	}

	// get new group list
	wishlistObjects, err := GetWishlistObjects(UserID)
	if err != nil {
		logger.Log.Error("Failed to get wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist."})
		context.Abort()
		return
	}

	// Sort wishlists by creation date
	sort.Slice(wishlistObjects, func(i, j int) bool {
		return wishlistObjects[j].CreatedAt.Before(wishlistObjects[i].CreatedAt)
	})

	context.JSON(http.StatusCreated, gin.H{"message": "Wishlist member joined.", "wishlists": wishlistObjects})
}

func RemoveFromWishlist(context *gin.Context) {
	// Create groupmembership request
	var wishlist_id = context.Param("wishlist_id")
	var wishlistObjects = []models.WishlistUser{}

	var wishlistMembershipRequest models.WishlistMembershipDeletionRequest
	if err := context.ShouldBindJSON(&wishlistMembershipRequest); err != nil {
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	// Parse group id
	wishlist_id_int, err := uuid.Parse(wishlist_id)
	if err != nil {
		logger.Log.Error("Failed to parse wishlist ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wishlist ID."})
		context.Abort()
		return
	}

	// Verify membership exists
	MembershipStatus, err := database.VerifyGroupMembershipToWishlist(wishlist_id_int, wishlistMembershipRequest.Group)
	if err != nil {
		logger.Log.Error("Failed to verify membership to wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify membership to wishlist."})
		context.Abort()
		return
	} else if !MembershipStatus {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Wishlist membership doesn't exist."})
		context.Abort()
		return
	}

	ownWishlist, err := database.VerifyUserOwnershipToWishlist(UserID, wishlist_id_int)
	if err != nil {
		logger.Log.Error("Failed to verify ownership of wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership of wishlist."})
		context.Abort()
		return
	} else if !ownWishlist {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Only owners can edit their wishlist memberships."})
		context.Abort()
		return
	}

	// Get the membership id
	membershipFound, wishlistmembership, err := database.GetMembershipIDForGroupToWishlist(wishlist_id_int, wishlistMembershipRequest.Group)
	if err != nil {
		logger.Log.Error("Failed to get group membership ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get group membership ID."})
		context.Abort()
		return
	} else if !membershipFound {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find group membership ID."})
		context.Abort()
		return
	}

	// Delete the membership
	err = database.DeleteWishlistMembership(wishlistmembership.ID)
	if err != nil {
		logger.Log.Error("Failed to delete wishlist membership. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete wishlist membership."})
		context.Abort()
		return
	}

	group_id, okay := context.GetQuery("group")
	if !okay {
		wishlistObjects, err = GetWishlistObjects(UserID)
		if err != nil {
			logger.Log.Error("Failed to get wishlist objects for user. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist objects for user."})
			context.Abort()
			return
		}
	} else {
		// Parse group id
		group_id_int, err := uuid.Parse(group_id)
		if err != nil {
			logger.Log.Error("Failed to parse group ID. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse group ID."})
			context.Abort()
			return
		}

		// Verify membership to group exists
		MembershipStatus, err := database.VerifyUserMembershipToGroup(UserID, group_id_int)
		if err != nil {
			logger.Log.Error("Failed to verify membership to group. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify membership to group."})
			context.Abort()
			return
		} else if !MembershipStatus {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "You are not a member of this group."})
			context.Abort()
			return
		}

		wishlistObjects, err = GetWishlistObjectsFromGroup(group_id_int, UserID)
		if err != nil {
			logger.Log.Error("Failed to get wishlists for group. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get wishlists for group."})
			context.Abort()
			return
		}
	}

	// Sort wishlists by creation date
	sort.Slice(wishlistObjects, func(i, j int) bool {
		return wishlistObjects[j].CreatedAt.Before(wishlistObjects[i].CreatedAt)
	})

	context.JSON(http.StatusCreated, gin.H{"message": "Group member removed.", "wishlists": wishlistObjects})
}

func APIUpdateWishlist(context *gin.Context) {

	// Create wishlist request
	var wishlist_id = context.Param("wishlist_id")
	var wishlist models.WishlistUpdateRequest

	// Get configuration
	configFile, err := config.GetConfig()
	if err != nil {
		logger.Log.Error("Failed to get config file. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get config file."})
		context.Abort()
		return
	}

	if err := context.ShouldBindJSON(&wishlist); err != nil {
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Trim request input
	wishlist.Name = strings.TrimSpace(wishlist.Name)
	wishlist.Description = strings.TrimSpace(wishlist.Description)

	// Parse group id
	wishlist_id_int, err := uuid.Parse(wishlist_id)
	if err != nil {
		logger.Log.Error("Failed to parse wishlist ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wishlist ID."})
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

	// Verify if collaboration exists
	collaborationStatus, err := database.VerifyWishlistCollaboratorToWishlist(wishlist_id_int, UserID)
	if err != nil {
		logger.Log.Error("Failed to verify wishlist collaborator status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist collaborator status."})
		context.Abort()
		return
	}

	WishlistOwnership, err := database.VerifyUserOwnershipToWishlist(UserID, wishlist_id_int)
	if err != nil {
		logger.Log.Error("Failed to verify ownership of group. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership of group."})
		context.Abort()
		return
	} else if !WishlistOwnership && !collaborationStatus {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You can only edit wishlists you own or collaborate on."})
		context.Abort()
		return
	}

	// Get original wishlist from DB
	wishlistOriginal, err := database.GetWishlist(wishlist_id_int)
	if err != nil {
		logger.Log.Error("Failed to get wishlist object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist object."})
		context.Abort()
		return
	}

	// Validate if name has changed
	if wishlistOriginal.Name != wishlist.Name {

		if len(wishlist.Name) < 5 || wishlist.Name == "" {
			context.JSON(http.StatusBadRequest, gin.H{"error": "The name of the wishlist must be five or more letters."})
			context.Abort()
			return
		}

		// Validate wishlist name format
		stringMatch, requirements, err := utilities.ValidateTextCharacters(wishlist.Name)
		if err != nil {
			logger.Log.Error("Failed to validate wishlist name text string. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
			context.Abort()
			return
		} else if !stringMatch {
			logger.Log.Error("Wishlist name text string failed validation.")
			context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
			context.Abort()
			return
		}

		unique_wish_name, err := database.VerifyUniqueWishlistNameForUser(wishlist.Name, UserID)
		if err != nil {
			logger.Log.Error("Failed to verify unique wishlist name. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify unique wishlist name."})
			context.Abort()
			return
		} else if !unique_wish_name {
			context.JSON(http.StatusBadRequest, gin.H{"error": "There is already a wishlist with that name on your profile."})
			context.Abort()
			return
		}
	}

	if wishlistOriginal.Description != wishlist.Description && wishlist.Description != "" {

		// Validate wishlist description format
		stringMatch, requirements, err := utilities.ValidateTextCharacters(wishlist.Description)
		if err != nil {
			logger.Log.Error("Failed to validate wishlist description text string. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
			context.Abort()
			return
		} else if !stringMatch {
			logger.Log.Error("Wishlist description text string failed validation.")
			context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
			context.Abort()
			return
		}

	}

	// Parse expiration date
	if wishlist.Date != nil {
		newDate, err := time.Parse("2006-01-02T15:04:05.000Z", *wishlist.Date)
		if err != nil {
			logger.Log.Error("Failed to parse date request. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse date request."})
			context.Abort()
			return
		}
		wishlistOriginal.Date = &newDate
	}

	// Finalize wishlist object
	wishlistOriginal.OwnerID = UserID
	wishlistOriginal.Description = wishlist.Description
	wishlistOriginal.Name = wishlist.Name
	wishlistOriginal.Claimable = &wishlist.Claimable
	wishlistOriginal.HideClaimers = &wishlist.HideClaimers
	wishlistOriginal.Expires = &wishlist.Expires
	wishlistOriginal.Public = &wishlist.Public
	wishlistOriginal.PublicHash = uuid.New()
	wishlistOriginal.ID = wishlist_id_int

	if *wishlistOriginal.Public && *wishlistOriginal.Claimable {
		context.JSON(http.StatusBadRequest, gin.H{"error": "A wishlist cannot have claimable wishes and be public to users without accounts."})
		context.Abort()
		return
	}

	// Update wishlist in DB
	wishlistOriginal, err = database.UpdateWishlistInDB(wishlistOriginal)
	if err != nil {
		logger.Log.Error("Failed to update wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update wishlist."})
		context.Abort()
		return
	}

	wishlistObject, err := ConvertWishlistToWishlistObject(wishlistOriginal, &UserID)
	if err != nil {
		logger.Log.Error("Failed to convert wishlist to wishlist object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert wishlist to wishlist object."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Wishlist updated.", "wishlist": wishlistObject, "public_url": configFile.PoenskelistenExternalURL})
}

func ConvertWishlistCollaboratorToWishlistCollaboratorObject(wishlistCollab models.WishlistCollaborator) (wishlistCollabObject models.WishlistCollaboratorObject, err error) {
	wishlistCollabObject = models.WishlistCollaboratorObject{}

	userObject, err := database.GetUserInformation(wishlistCollab.UserID)
	if err != nil {
		logger.Log.Error("Failed to get user information for user ID '" + wishlistCollab.ID.String() + "'. Returning. Error: " + err.Error())
		return wishlistCollabObject, errors.New("Failed to get user information for user ID '" + wishlistCollab.ID.String() + "'.")
	}

	wishlistCollabObject.User = userObject

	// Prevent double nesting by just including Wihslist ID
	wishlistCollabObject.WishlistID = wishlistCollab.WishlistID

	wishlistCollabObject.CreatedAt = wishlistCollab.CreatedAt
	wishlistCollabObject.DeletedAt = wishlistCollab.DeletedAt
	wishlistCollabObject.Enabled = wishlistCollab.Enabled
	wishlistCollabObject.ID = wishlistCollab.ID
	wishlistCollabObject.UpdatedAt = wishlistCollab.UpdatedAt

	return
}

func ConvertWishlistCollaboratorsToWishlistCollaboratorsObjects(wishlistCollabs []models.WishlistCollaborator) (wishlistCollabObjects []models.WishlistCollaboratorObject, err error) {
	err = nil
	wishlistCollabObjects = []models.WishlistCollaboratorObject{}

	for _, wishlistCollab := range wishlistCollabs {
		wishlistCollabObject, err := ConvertWishlistCollaboratorToWishlistCollaboratorObject(wishlistCollab)
		if err != nil {
			logger.Log.Warn("Failed to get wishlist collaberator object for '" + wishlistCollab.ID.String() + "'. Skipping. Error: " + err.Error())
			continue
		}
		wishlistCollabObjects = append(wishlistCollabObjects, wishlistCollabObject)
	}

	return
}

func ConvertWishlistToWishlistObject(wishlist models.Wishlist, RequestUserID *uuid.UUID) (wishlistObject models.WishlistUser, err error) {
	wishlistObject = models.WishlistUser{}

	groups, err := database.GetGroupMembersFromWishlist(wishlist.ID, wishlist.OwnerID)
	if err != nil {
		return models.WishlistUser{}, err
	}

	groupObjects, err := ConvertGroupsToGroupObjects(groups)
	if err != nil {
		logger.Log.Error("Failed to convert groups to groups objects. Returning. Error: " + err.Error())
		return models.WishlistUser{}, err
	}

	wishlistsCollabs, err := database.GetWishlistCollaboratorsFromWishlist(wishlist.ID)
	if err != nil {
		logger.Log.Error("Failed to convert wishlist collaborators. Returning. Error: " + err.Error())
		return models.WishlistUser{}, err
	}

	wishlistsCollabObjects, err := ConvertWishlistCollaboratorsToWishlistCollaboratorsObjects(wishlistsCollabs)
	if err != nil {
		logger.Log.Error("Failed to convert wishlist collaborators to wishlist collaborator objects. Returning. Error: " + err.Error())
		return models.WishlistUser{}, err
	}

	owner, err := database.GetUserInformation(wishlist.OwnerID)
	if err != nil {
		logger.Log.Error("Failed to get information of wishlist owner '" + wishlist.OwnerID.String() + "'. Returning. Error: " + err.Error())
		return models.WishlistUser{}, err
	}

	configFile, err := config.GetConfig()
	if err != nil {
		logger.Log.Error("Failed to get config. Error: " + err.Error())
		return models.WishlistUser{}, errors.New("Failed to get config.")
	}

	wishlistObject.CreatedAt = wishlist.CreatedAt
	wishlistObject.Date = wishlist.Date
	wishlistObject.DeletedAt = wishlist.DeletedAt
	wishlistObject.Description = wishlist.Description
	wishlistObject.Enabled = wishlist.Enabled
	wishlistObject.ID = wishlist.ID
	wishlistObject.Members = groupObjects
	wishlistObject.Owner = owner
	wishlistObject.Name = wishlist.Name
	wishlistObject.UpdatedAt = wishlist.UpdatedAt
	wishlistObject.Claimable = wishlist.Claimable
	wishlistObject.HideClaimers = wishlist.HideClaimers
	wishlistObject.Collaborators = wishlistsCollabObjects
	wishlistObject.Expires = wishlist.Expires
	wishlistObject.Public = wishlist.Public
	wishlistObject.PublicHash = wishlist.PublicHash
	wishlistObject.Currency = configFile.PoenskelistenCurrency

	// Get wishes
	_, wishes, err := database.GetWishesFromWishlist(wishlist.ID)
	if err != nil {
		logger.Log.Error("Failed to get wishes from database. Returning. Error: " + err.Error())
		return models.WishlistUser{}, err
	}

	wishObjects, err := ConvertWishesToWishObjects(wishes, RequestUserID)
	if err != nil {
		logger.Log.Error("Failed to convert wishes to wish objects. Returning. Error: " + err.Error())
		return models.WishlistUser{}, err
	}

	// Sort wishes by creation date
	sort.Slice(wishObjects, func(i, j int) bool {
		return wishObjects[j].UpdatedAt.Before(wishObjects[i].UpdatedAt)
	})

	wishlistObject.Wishes = wishObjects

	if len(wishlistObject.Wishes) > 0 {
		var wishUpdatedAt = wishlistObject.Wishes[0].UpdatedAt
		if wishlistObject.UpdatedAt.After(wishUpdatedAt) {
			wishlistObject.WishUpdatedAt = wishlistObject.UpdatedAt
		} else {
			wishlistObject.WishUpdatedAt = wishUpdatedAt
		}
	} else {
		wishlistObject.WishUpdatedAt = wishlistObject.UpdatedAt
	}

	return
}

func ConvertWishlistsToWishlistObjects(wishlists []models.Wishlist, RequestUserID *uuid.UUID) (wishlistObjects []models.WishlistUser, err error) {
	err = nil
	wishlistObjects = []models.WishlistUser{}

	for _, wishlist := range wishlists {
		wishlistObject, err := ConvertWishlistToWishlistObject(wishlist, RequestUserID)
		if err != nil {
			logger.Log.Warn("Failed to get wishlist object for wishlist ID '" + wishlist.ID.String() + "'. Skipping. Error: " + err.Error())
			continue
		}
		wishlistObjects = append(wishlistObjects, wishlistObject)
	}

	return
}

func APICollaborateWishlist(context *gin.Context) {
	// Create groupmembership request
	var wishlist_id = context.Param("wishlist_id")
	var wishlistCollaboratorsRequest models.WishlistCollaboratorCreationRequest

	if err := context.ShouldBindJSON(&wishlistCollaboratorsRequest); err != nil {
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	if len(wishlistCollaboratorsRequest.Users) < 1 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You must provide one or more users."})
		context.Abort()
		return
	}

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

	for _, userID := range wishlistCollaboratorsRequest.Users {
		wishlistCollaborator := models.WishlistCollaborator{}

		// Verify user exists
		user, err := database.GetUserInformation(userID)
		if err != nil {
			logger.Log.Error("Failed to get user object. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user object."})
			context.Abort()
			return
		}

		wishlistCollaborator.UserID = user.ID

		// Verify collaboration doesn't exist
		collaborationStatus, err := database.VerifyWishlistCollaboratorToWishlist(wishlist_id_int, user.ID)
		if err != nil {
			logger.Log.Error("Failed to verify wishlist collaborator status. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist collaborator status."})
			context.Abort()
			return
		} else if collaborationStatus {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Wishlist collaboration already exists."})
			context.Abort()
			return
		}

		wishlistOwnerID, err := database.GetWishlistOwner(wishlist_id_int)
		if err != nil {
			logger.Log.Error("Failed to verify wishlist owner. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist owner."})
			context.Abort()
			return
		} else if wishlistOwnerID != UserID {
			context.JSON(http.StatusUnauthorized, gin.H{"error": "Only the wishlist owner can add collaborators."})
			context.Abort()
			return
		}

		if UserID == user.ID {
			context.JSON(http.StatusBadRequest, gin.H{"error": "The wishlist owner can't be a collaborator."})
			context.Abort()
			return
		}

		wishlistCollaborator.WishlistID = wishlist_id_int
		wishlistCollaborator.ID = uuid.New()

		// Add group membership to database
		err = database.CreateWishlistCollaboratorInDB(wishlistCollaborator)
		if err != nil {
			logger.Log.Error("Failed to save wishlist collaborator. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save wishlist collaborator."})
			context.Abort()
			return
		}

	}

	// get new wishlist list
	wishlistObjects, err := GetWishlistObjects(UserID)
	if err != nil {
		logger.Log.Error("Failed to get new wishlist objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get new wishlist objects."})
		context.Abort()
		return
	}

	// Sort wishlists by creation date
	sort.Slice(wishlistObjects, func(i, j int) bool {
		return wishlistObjects[j].CreatedAt.Before(wishlistObjects[i].CreatedAt)
	})

	context.JSON(http.StatusCreated, gin.H{"message": "Wishlist collaborators added.", "wishlists": wishlistObjects})
}

func APIUnCollaborateWishlist(context *gin.Context) {

	// Create groupmembership request
	var wishlist_id = context.Param("wishlist_id")
	wishlistCollaboratorRequest := models.WishlistCollaboratorDeletionRequest{}
	if err := context.ShouldBindJSON(&wishlistCollaboratorRequest); err != nil {
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to get user ID from headers. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get UserID from headers."})
		context.Abort()
		return
	}

	// Parse group id
	wishlist_id_int, err := uuid.Parse(wishlist_id)
	if err != nil {
		logger.Log.Error("Failed to parse wishlist ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wishlist ID."})
		context.Abort()
		return
	}

	// Verify collaboration exists
	collaborationStatus, err := database.VerifyWishlistCollaboratorToWishlist(wishlist_id_int, wishlistCollaboratorRequest.User)
	if err != nil {
		logger.Log.Error("Failed to verify wishlist collaborator status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist collaborator status."})
		context.Abort()
		return
	} else if !collaborationStatus {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Wishlist collaboration doesn't exist."})
		context.Abort()
		return
	}

	// Verify wishlist is owned by requester
	wishlistOwnerID, err := database.GetWishlistOwner(wishlist_id_int)
	if err != nil {
		logger.Log.Error("Failed to verify wishlist owner. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist owner."})
		context.Abort()
		return
	} else if wishlistOwnerID != UserID {
		context.JSON(http.StatusUnauthorized, gin.H{"error": "Only the wishlist owner can add collaborators."})
		context.Abort()
		return
	}

	// Get the collaboration id
	wishlistCollaborator, err := database.GetWishlistCollaboratorByUserIDAndWishlistID(wishlist_id_int, wishlistCollaboratorRequest.User)
	if err != nil {
		logger.Log.Error("Failed to get collaboration in database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get collaboration in database."})
		context.Abort()
		return
	}

	// Delete the collaboration
	err = database.DeleteWishlistCollaboratorByWishlistCollaboratorID(wishlistCollaborator.ID)
	if err != nil {
		logger.Log.Error("Failed to remove collaborator in database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove collaborator in database."})
		context.Abort()
		return
	}

	// get new wishlist list
	wishlistObjects, err := GetWishlistObjects(UserID)
	if err != nil {
		logger.Log.Error("Failed to get new wishlist objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get new wishlist objects."})
		context.Abort()
		return
	}

	// Sort wishlists by creation date
	sort.Slice(wishlistObjects, func(i, j int) bool {
		return wishlistObjects[j].CreatedAt.Before(wishlistObjects[i].CreatedAt)
	})

	context.JSON(http.StatusCreated, gin.H{"message": "Wishlist collaborator removed.", "wishlists": wishlistObjects})
}

func GetPublicWishlist(context *gin.Context) {
	// Create wishlist request
	var wishlistHash = context.Param("wishlist_hash")

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		logger.Log.Error("Failed to get config file. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get config file."})
		context.Abort()
		return
	}

	// parse wishlist id
	wishlistHashUUID, err := uuid.Parse(wishlistHash)
	if err != nil {
		logger.Log.Error("Failed to parse wishlist hash. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wishlist hash."})
		context.Abort()
		return
	}

	wishlistFound, wishlist, err := database.GetPublicWishListByWishlistHash(wishlistHashUUID)
	if err != nil {
		logger.Log.Error("Failed to get wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist."})
		context.Abort()
		return
	} else if !wishlistFound {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find wishlist."})
		context.Abort()
		return
	}

	wishlistObject, err := ConvertWishlistToWishlistObject(wishlist, nil)
	if err != nil {
		logger.Log.Error("Failed to convert to wishlist object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert to wishlist object."})
		context.Abort()
		return
	}

	// Sort wishlist wishes by creation date
	sort.Slice(wishlistObject.Wishes, func(i, j int) bool {
		return wishlistObject.Wishes[j].CreatedAt.Before(wishlistObject.Wishes[i].CreatedAt)
	})

	context.JSON(http.StatusOK, gin.H{"wishlist": wishlistObject, "message": "Wishlist retrieved.", "currency": config.PoenskelistenCurrency, "padding": config.PoenskelistenCurrencyPad})
}
