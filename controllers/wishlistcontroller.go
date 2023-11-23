package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"errors"
	"log"
	"net/http"
	"sort"
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

	var group_membership = false
	var group_id = uuid.UUID{}
	if wishlist.Group != nil {
		group_membership = true

		// Parse group id
		group_id = *wishlist.Group

		MembershipStatus, err := database.VerifyUserMembershipToGroup(UserID, group_id)
		if err != nil {
			log.Println("Failed to verify membership to group. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify membership to group."})
			context.Abort()
			return
		} else if !MembershipStatus {
			context.JSON(http.StatusBadRequest, gin.H{"error": "You are not a member of this group."})
			context.Abort()
			return
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
		log.Println("Failed to validate wishlist name text string. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
		context.Abort()
		return
	} else if !stringMatch {
		log.Println("Wishlist name text string failed validation.")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	// Validate wishlist description format
	stringMatch, requirements, err = utilities.ValidateTextCharacters(wishlist.Description)
	if err != nil {
		log.Println("Failed to validate description name text string. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
		context.Abort()
		return
	} else if !stringMatch {
		log.Println("description name text string failed validation.")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	wishlistdb.Expires = wishlist.Expires

	wishlistdb.Date, err = time.Parse("2006-01-02T15:04:05.000Z", wishlist.Date)
	if err != nil && *wishlistdb.Expires {
		log.Println("Failed to parse date time. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse date time."})
		context.Abort()
		return
	}

	if now.After(wishlistdb.Date) && *wishlistdb.Expires {
		context.JSON(http.StatusBadRequest, gin.H{"error": "The date of the wishlist must be in the future."})
		context.Abort()
		return
	}

	if !*wishlistdb.Expires {
		wishlistdb.Date = time.Now()
	}

	unique_wish_name, err := database.VerifyUniqueWishlistNameForUser(wishlist.Name, UserID)
	if err != nil {
		log.Println("Failed to verify unique wishlist name. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify unique wishlist name."})
		context.Abort()
		return
	} else if !unique_wish_name {
		context.JSON(http.StatusBadRequest, gin.H{"error": "There is already a wishlist with that name on your profile."})
		context.Abort()
		return
	}

	// Finalize wishlist object
	wishlistdb.OwnerID = UserID
	wishlistdb.Description = wishlist.Description
	wishlistdb.Name = wishlist.Name
	wishlistdb.Claimable = wishlist.Claimable
	wishlistdb.ID = uuid.New()

	// Create wishlist in DB
	wishlistdb, err = database.CreateWishlistInDB(wishlistdb)
	if err != nil {
		log.Println("Failed to create wishlist in database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create wishlist in database."})
		context.Abort()
		return
	}

	var wishlists_with_users []models.WishlistUser

	// If a group was referenced, create the wishlist membership
	if group_membership == true {
		var wishlistmembershipdb models.WishlistMembership
		wishlistmembershipdb.GroupID = group_id
		wishlistmembershipdb.WishlistID = wishlistdb.ID
		wishlistmembershipdb.ID = uuid.New()

		// Add group membership to database
		record := database.Instance.Create(&wishlistmembershipdb)
		if record.Error != nil {
			log.Println("Failed to create membership to wishlist. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create membership to wishlist."})
			context.Abort()
			return
		}

		wishlists_with_users, err = GetWishlistObjectsFromGroup(group_id, UserID)
		if err != nil {
			log.Println("Failed to get wishlist objects from group. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get wishlist objects from group."})
			context.Abort()
			return
		}

	} else {

		wishlists_with_users, err = GetWishlistObjects(UserID)
		if err != nil {
			log.Println("Failed to get wishlist objects. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get wishlist objects."})
			context.Abort()
			return
		}

	}

	context.JSON(http.StatusCreated, gin.H{"message": "Wishlist created.", "wishlists": wishlists_with_users})
}

func DeleteWishlist(context *gin.Context) {
	var wishlistObjects = []models.WishlistUser{}
	var wishlist = context.Param("wishlist_id")

	// Get user ID
	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Parse wishlist id
	wishlist_id_int, err := uuid.Parse(wishlist)
	if err != nil {
		log.Println("Failed to parse wishlist ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wishlist ID."})
		context.Abort()
		return
	}

	// Verify wishlist owner
	MembershipStatus, err := database.VerifyUserOwnershipToWishlist(UserID, wishlist_id_int)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if !MembershipStatus {
		//context.JSON(http.StatusInternalServerError, gin.H{"error": groupmembershiprecord.Error.Error()})
		context.JSON(http.StatusInternalServerError, gin.H{"error": "You are not the owner of this wishlist."})
		context.Abort()
		return
	}

	err = database.DeleteWishlist(wishlist_id_int)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	group_id, okay := context.GetQuery("group")
	if !okay {
		wishlistObjects, err = GetWishlistObjects(UserID)
		if err != nil {
			log.Println("Failed to get wishlist objects for user. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist objects for user."})
			context.Abort()
			return
		}
	} else {
		// Parse group id
		group_id_int, err := uuid.Parse(group_id)
		if err != nil {
			log.Println("Failed to parse group ID. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse group ID."})
			context.Abort()
			return
		}

		// Verify membership to group exists
		MembershipStatus, err := database.VerifyUserMembershipToGroup(UserID, group_id_int)
		if err != nil {
			log.Println("Failed to verify membership to group. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify membership to group."})
			context.Abort()
			return
		} else if !MembershipStatus {
			//context.JSON(http.StatusInternalServerError, gin.H{"error": groupmembershiprecord.Error.Error()})
			context.JSON(http.StatusInternalServerError, gin.H{"error": "You are not a member of this group."})
			context.Abort()
			return
		}

		wishlistObjects, err = GetWishlistObjectsFromGroup(group_id_int, UserID)
		if err != nil {
			log.Println("Failed to get wishlists for group. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get wishlists for group."})
			context.Abort()
			return
		}
	}

	context.JSON(http.StatusOK, gin.H{"wishlists": wishlistObjects, "message": "Wishlist deleted."})

}

func GetWishlistObjectsFromGroup(group_id uuid.UUID, RequestUserID uuid.UUID) (wishlistObjects []models.WishlistUser, err error) {
	err = nil
	wishlistObjects = []models.WishlistUser{}

	wishlists, err := database.GetWishlistsFromGroup(group_id)
	if err != nil {
		return []models.WishlistUser{}, err
	}

	wishlistObjects, err = ConvertWishlistsToWishlistObjects(wishlists, &RequestUserID)
	if err != nil {
		log.Println("Failed to convert wishlists to objects. Returning. Error: " + err.Error())
		return wishlistObjects, errors.New("Failed to convert wishlists to objects.")
	}

	return wishlistObjects, nil
}

func GetWishlist(context *gin.Context) {

	// Create wishlist request
	var wishlist_id = context.Param("wishlist_id")

	// Get user ID
	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// parse wishlist id
	wishlist_id_int, err := uuid.Parse(wishlist_id)
	if err != nil {
		log.Println("Failed to parse wishlist ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wishlist ID."})
		context.Abort()
		return
	}

	WishlistOwnership, err := database.VerifyUserOwnershipToWishlist(UserID, wishlist_id_int)
	if err != nil {
		log.Println("Failed to verify ownership of group. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership of group."})
		context.Abort()
		return
	}

	WishlistMembership, err := database.VerifyUserMembershipToGroupmembershipToWishlist(UserID, wishlist_id_int)
	if err != nil {
		log.Println("Failed to verify membership to group. Error: " + err.Error())
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
		log.Println("Failed to get wishlist object. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get wishlist object."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"wishlist": wishlist_with_user, "message": "Wishlist retrieved."})

}

func GetWishlistObject(WishlistID uuid.UUID, RequestUserID uuid.UUID) (wishlistObject models.WishlistUser, err error) {
	err = nil
	wishlistObject = models.WishlistUser{}

	wishlist, err := database.GetWishlist(WishlistID)
	if err != nil {
		log.Println("Failed to get wishlist '" + WishlistID.String() + "' from DB. Returning. Error: " + err.Error())
		return models.WishlistUser{}, err
	}

	wishlistObject, err = ConvertWishlistToWishlistObject(wishlist, &RequestUserID)
	if err != nil {
		log.Println("Failed to convert wishlist '" + WishlistID.String() + "' to object. Returning. Error: " + err.Error())
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
		wishlistObjects, err = GetWishlistObjects(UserID)
		if err != nil {
			log.Println("Failed to get wishlist objects for user. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist objects for user."})
			context.Abort()
			return
		}

	} else {
		// Parse group id
		group_id_int, err := uuid.Parse(group_id)
		if err != nil {
			log.Println("Failed to parse group ID. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse group ID."})
			context.Abort()
			return
		}

		// Verify membership to group exists
		MembershipStatus, err := database.VerifyUserMembershipToGroup(UserID, group_id_int)
		if err != nil {
			log.Println("Failed to verify membership to group. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify membership to group."})
			context.Abort()
			return
		} else if !MembershipStatus {
			//context.JSON(http.StatusInternalServerError, gin.H{"error": groupmembershiprecord.Error.Error()})
			context.JSON(http.StatusInternalServerError, gin.H{"error": "You are not a member of this group."})
			context.Abort()
			return
		}

		wishlistObjects, err = GetWishlistObjectsFromGroup(group_id_int, UserID)
		if err != nil {
			log.Println("Failed to get wishlists for group. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get wishlists for group."})
			context.Abort()
			return
		}
	}

	// Sort wishlists by creation date
	sort.Slice(wishlistObjects, func(i, j int) bool {
		return wishlistObjects[i].CreatedAt.Before(wishlistObjects[j].CreatedAt)
	})

	context.JSON(http.StatusOK, gin.H{"wishlists": wishlistObjects, "message": "Wishlists retrieved."})
}

func GetWishlistObjects(UserID uuid.UUID) (wishlistObjects []models.WishlistUser, err error) {
	err = nil
	wishlistObjects = []models.WishlistUser{}

	wishlists, err := database.GetOwnedWishlists(UserID)
	if err != nil {
		log.Println("Failed to get owned wishlists for user '" + UserID.String() + "'. Returning. Error: " + err.Error())
		return wishlistObjects, errors.New("Failed to get owned wishlists for user '" + UserID.String() + "'.")
	}

	wishlistsThroughCollab, err := database.GetWishlistsByUserIDThroughWishlistCollaborations(UserID)
	if err != nil {
		log.Println("Failed to get collaboration wishlists for user '" + UserID.String() + "'. Returning. Error: " + err.Error())
		return wishlistObjects, errors.New("Failed to get collaboration wishlists for user '" + UserID.String() + "'.")
	}

	for _, wishlistThroughCollab := range wishlistsThroughCollab {
		wishlists = append(wishlists, wishlistThroughCollab)
	}

	wishlistObjects, err = ConvertWishlistsToWishlistObjects(wishlists, &UserID)
	if err != nil {
		log.Println("Failed to convert wishlists to objects. Returning. Error: " + err.Error())
		return wishlistObjects, errors.New("Failed to convert wishlists to objects.")
	}

	return wishlistObjects, nil
}

func JoinWishlist(context *gin.Context) {

	// Create groupmembership request
	var wishlist_id = context.Param("wishlist_id")
	var wishlistmembership models.WishlistMembershipCreationRequest

	if err := context.ShouldBindJSON(&wishlistmembership); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Parse wishlist id
	wishlist_id_int, err := uuid.Parse(wishlist_id)
	if err != nil {
		log.Println("Failed to parse wishlist ID. Error: " + err.Error())
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
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			context.Abort()
			return
		}

		// Verify membership doesnt exist
		MembershipStatus, err := database.VerifyGroupMembershipToWishlist(wishlist_id_int, Group)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			context.Abort()
			return
		} else if MembershipStatus {
			//context.JSON(http.StatusInternalServerError, gin.H{"error": groupmembershiprecord.Error.Error()})
			context.JSON(http.StatusBadRequest, gin.H{"error": "Wishlist membership already exists."})
			context.Abort()
			return
		}

		// Verify wishlist is owned by requester
		wishlistOwned, err := database.VerifyUserOwnershipToWishlist(UserID, wishlist_id_int)
		if err != nil {
			log.Println("Failed to verify ownership to wishlist. Error: " + err.Error())
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
			context.JSON(http.StatusInternalServerError, gin.H{"error": record.Error.Error()})
			context.Abort()
			return
		}

	}

	// get new group list
	wishlists_with_users, err := GetWishlistObjects(UserID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Wishlist member joined.", "wishlists": wishlists_with_users})
}

func RemoveFromWishlist(context *gin.Context) {
	// Create groupmembership request
	var wishlist_id = context.Param("wishlist_id")
	var wishlistObjects = []models.WishlistUser{}

	var wishlistMembershipRequest models.WishlistMembershipDeletionRequest
	if err := context.ShouldBindJSON(&wishlistMembershipRequest); err != nil {
		log.Println("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Parse group id
	wishlist_id_int, err := uuid.Parse(wishlist_id)
	if err != nil {
		log.Println("Failed to parse wishlist ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wishlist ID."})
		context.Abort()
		return
	}

	// Verify membership exists
	MembershipStatus, err := database.VerifyGroupMembershipToWishlist(wishlist_id_int, wishlistMembershipRequest.Group)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if !MembershipStatus {
		//context.JSON(http.StatusInternalServerError, gin.H{"error": groupmembershiprecord.Error.Error()})
		context.JSON(http.StatusBadRequest, gin.H{"error": "Wishlist membership doesn't exist."})
		context.Abort()
		return
	}

	ownWishlist, err := database.VerifyUserOwnershipToWishlist(UserID, wishlist_id_int)
	if err != nil {
		log.Println("Failed to verify ownership of wishlist. Error: " + err.Error())
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
		log.Println("Failed to get group membership ID. Error: " + err.Error())
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
		log.Println("Failed to delete wishlist membership. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete wishlist membership."})
		context.Abort()
		return
	}

	group_id, okay := context.GetQuery("group")
	if !okay {
		wishlistObjects, err = GetWishlistObjects(UserID)
		if err != nil {
			log.Println("Failed to get wishlist objects for user. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist objects for user."})
			context.Abort()
			return
		}
	} else {
		// Parse group id
		group_id_int, err := uuid.Parse(group_id)
		if err != nil {
			log.Println("Failed to parse group ID. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse group ID."})
			context.Abort()
			return
		}

		// Verify membership to group exists
		MembershipStatus, err := database.VerifyUserMembershipToGroup(UserID, group_id_int)
		if err != nil {
			log.Println("Failed to verify membership to group. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify membership to group."})
			context.Abort()
			return
		} else if !MembershipStatus {
			//context.JSON(http.StatusInternalServerError, gin.H{"error": groupmembershiprecord.Error.Error()})
			context.JSON(http.StatusInternalServerError, gin.H{"error": "You are not a member of this group."})
			context.Abort()
			return
		}

		wishlistObjects, err = GetWishlistObjectsFromGroup(group_id_int, UserID)
		if err != nil {
			log.Println("Failed to get wishlists for group. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get wishlists for group."})
			context.Abort()
			return
		}
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Group member removed.", "wishlists": wishlistObjects})
}

func APIUpdateWishlist(context *gin.Context) {

	// Create wishlist request
	var wishlist_id = context.Param("wishlist_id")
	var wishlist models.WishlistUpdateRequest
	var wishlistdb models.Wishlist

	if err := context.ShouldBindJSON(&wishlist); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Parse group id
	wishlist_id_int, err := uuid.Parse(wishlist_id)
	if err != nil {
		log.Println("Failed to parse wishlist ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wishlist ID."})
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

	// Verify if collaboration exists
	collaborationStatus, err := database.VerifyWishlistCollaboratorToWishlist(wishlist_id_int, UserID)
	if err != nil {
		log.Println("Failed to verify wishlist collaborator status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist collaborator status."})
		context.Abort()
		return
	}

	WishlistOwnership, err := database.VerifyUserOwnershipToWishlist(UserID, wishlist_id_int)
	if err != nil {
		log.Println("Failed to verify ownership of group. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership of group."})
		context.Abort()
		return
	} else if !WishlistOwnership && !collaborationStatus {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You can only edit wishlists you own or collaborate on."})
		context.Abort()
		return
	}

	// Get original wishlist from DB
	wishlistOriginal, err := GetWishlistObject(wishlist_id_int, UserID)
	if err != nil {
		log.Println("Failed to get wishlist object. Error: " + err.Error())
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
			log.Println("Failed to validate wishlist name text string. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
			context.Abort()
			return
		} else if !stringMatch {
			log.Println("Wishlist name text string failed validation.")
			context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
			context.Abort()
			return
		}

		unique_wish_name, err := database.VerifyUniqueWishlistNameForUser(wishlist.Name, UserID)
		if err != nil {
			log.Println("Failed to verify unique wishlist name. Error: " + err.Error())
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
			log.Println("Failed to validate wishlist description text string. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
			context.Abort()
			return
		} else if !stringMatch {
			log.Println("Wishlist description text string failed validation.")
			context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
			context.Abort()
			return
		}

	}

	// Parse expiration date
	wishlistdb.Date, err = time.Parse("2006-01-02T15:04:05.000Z", wishlist.Date)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Finalize wishlist object
	wishlistdb.OwnerID = UserID
	wishlistdb.Description = wishlist.Description
	wishlistdb.Name = wishlist.Name
	wishlistdb.Claimable = wishlist.Claimable
	wishlistdb.Expires = wishlist.Expires

	// Update wishlist in DB
	err = database.UpdateWishlistValuesByID(wishlist_id_int, wishlistdb.Name, wishlistdb.Description, wishlistdb.Date, *wishlistdb.Claimable, *wishlistdb.Expires)
	if err != nil {
		log.Println("Failed to update wishlist. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update wishlist."})
		context.Abort()
		return
	}

	// Get updated wishlist from DB
	wishlist_with_user, err := GetWishlistObject(wishlist_id_int, UserID)
	if err != nil {
		log.Println("Failed to get wishlist object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wishlist object."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Wishlist updated.", "wishlist": wishlist_with_user})
}

func ConvertWishlistCollaberatorToWishlistCollaberatorObject(wishlistCollab models.WishlistCollaborator) (wishlistCollabObject models.WishlistCollaboratorObject, err error) {
	err = nil
	wishlistCollabObject = models.WishlistCollaboratorObject{}

	userObject, err := database.GetUserInformation(wishlistCollab.UserID)
	if err != nil {
		log.Println("Failed to get user information for user ID '" + wishlistCollab.ID.String() + "'. Returning. Error: " + err.Error())
		return wishlistCollabObject, errors.New("Failed to get user information for user ID '" + wishlistCollab.ID.String() + "'.")
	}

	wishlistCollabObject.CreatedAt = wishlistCollab.CreatedAt
	wishlistCollabObject.DeletedAt = wishlistCollab.DeletedAt
	wishlistCollabObject.Enabled = wishlistCollab.Enabled
	wishlistCollabObject.ID = wishlistCollab.ID
	wishlistCollabObject.UpdatedAt = wishlistCollab.UpdatedAt
	wishlistCollabObject.User = userObject
	wishlistCollabObject.Wishlist = wishlistCollabObject.Wishlist

	return
}

func ConvertWishlistCollaberatorsToWishlistCollaberatorObjects(wishlistCollabs []models.WishlistCollaborator) (wishlistCollabObjects []models.WishlistCollaboratorObject, err error) {
	err = nil
	wishlistCollabObjects = []models.WishlistCollaboratorObject{}

	for _, wishlistCollab := range wishlistCollabs {
		wishlistCollabObject, err := ConvertWishlistCollaberatorToWishlistCollaberatorObject(wishlistCollab)
		if err != nil {
			log.Println("Failed to get wishlist collaberator object for '" + wishlistCollab.ID.String() + "'. Skipping. Error: " + err.Error())
			continue
		}
		wishlistCollabObjects = append(wishlistCollabObjects, wishlistCollabObject)
	}

	return
}

func ConvertWishlistToWishlistObject(wishlist models.Wishlist, RequestUserID *uuid.UUID) (wishlistObject models.WishlistUser, err error) {
	err = nil
	wishlistObject = models.WishlistUser{}

	groups, err := database.GetGroupMembersFromWishlist(wishlist.ID)
	if err != nil {
		return models.WishlistUser{}, err
	}

	groupObjects, err := ConvertGroupsToGroupObjects(groups)
	if err != nil {
		log.Println("Failed to convert groups to groups objects. Returning. Error: " + err.Error())
		return models.WishlistUser{}, err
	}

	wishlistsCollabs, err := database.GetWishlistCollaboratorsFromWishlist(wishlist.ID)
	if err != nil {
		log.Println("Failed to convert wishlist collaberators. Returning. Error: " + err.Error())
		return models.WishlistUser{}, err
	}

	wishlistsCollabObjects, err := ConvertWishlistCollaberatorsToWishlistCollaberatorObjects(wishlistsCollabs)
	if err != nil {
		log.Println("Failed to convert wishlist collaberators to wishlist collaberator objects. Returning. Error: " + err.Error())
		return models.WishlistUser{}, err
	}

	owner, err := database.GetUserInformation(wishlist.OwnerID)
	if err != nil {
		log.Println("Failed to get information of wishlist owner '" + wishlist.OwnerID.String() + "'. Returning. Error: " + err.Error())
		return models.WishlistUser{}, err
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
	wishlistObject.Collaborators = wishlistsCollabObjects
	wishlistObject.Expires = wishlist.Expires

	// Get wishes
	_, wishes, err := database.GetWishesFromWishlist(wishlist.ID)
	if err != nil {
		log.Println("Failed to get wishes from database. Returning. Error: " + err.Error())
		return models.WishlistUser{}, err
	}

	wishObjects, err := ConvertWishesToWishObjects(wishes, RequestUserID)
	if err != nil {
		log.Println("Failed to convert wishes to wish objects. Returning. Error: " + err.Error())
		return models.WishlistUser{}, err
	}

	wishlistObject.Wishes = wishObjects

	return
}

func ConvertWishlistsToWishlistObjects(wishlists []models.Wishlist, RequestUserID *uuid.UUID) (wishlistObjects []models.WishlistUser, err error) {
	err = nil
	wishlistObjects = []models.WishlistUser{}

	for _, wishlist := range wishlists {
		wishlistObject, err := ConvertWishlistToWishlistObject(wishlist, RequestUserID)
		if err != nil {
			log.Println("Failed to get wishlist object for wishlist ID '" + wishlist.ID.String() + "'. Skipping. Error: " + err.Error())
			continue
		}
		wishlistObjects = append(wishlistObjects, wishlistObject)
	}

	return
}

func APICollaborateWishlist(context *gin.Context) {
	// Create groupmembership request
	var wishlist_id = context.Param("wishlist_id")
	var wishlistCollaboratorsequest models.WishlistCollaboratorCreationRequest

	if err := context.ShouldBindJSON(&wishlistCollaboratorsequest); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	if len(wishlistCollaboratorsequest.Users) < 1 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You must provide one or more users."})
		context.Abort()
		return
	}

	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Parse wishlist id
	wishlist_id_int, err := uuid.Parse(wishlist_id)
	if err != nil {
		log.Println("Failed to parse wishlist ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wishlist ID."})
		context.Abort()
		return
	}

	for _, user := range wishlistCollaboratorsequest.Users {

		wishlistCollaborator := models.WishlistCollaborator{}
		wishlistCollaborator.UserID = user

		// Verify user exists
		_, err := database.GetUserInformation(user)
		if err != nil {
			log.Println("Failed to get user object. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user object."})
			context.Abort()
			return
		}

		// Verify collaboration doesnt exist
		collaborationStatus, err := database.VerifyWishlistCollaboratorToWishlist(wishlist_id_int, user)
		if err != nil {
			log.Println("Failed to verify wishlist collaborator status. Error: " + err.Error())
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
			log.Println("Failed to verify wishlist owner. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify wishlist owner."})
			context.Abort()
			return
		} else if wishlistOwnerID != UserID {
			context.JSON(http.StatusUnauthorized, gin.H{"error": "Only the wishlist owner can add collaborators."})
			context.Abort()
			return
		}

		if UserID == user {
			context.JSON(http.StatusBadRequest, gin.H{"error": "The wishlist owner can't be a collaborator."})
			context.Abort()
			return
		}

		wishlistCollaborator.WishlistID = wishlist_id_int
		wishlistCollaborator.ID = uuid.New()

		// Add group membership to database
		err = database.CreateWishlistCollaboratorInDB(wishlistCollaborator)
		if err != nil {
			log.Println("Failed to save wishlist collaborator. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save wishlist collaborator."})
			context.Abort()
			return
		}

	}

	// get new wishlist list
	wishlists_with_users, err := GetWishlistObjects(UserID)
	if err != nil {
		log.Println("Failed to get new wishlist objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get new wishlist objects."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Wishlist collaborators added.", "wishlists": wishlists_with_users})
}

func APIUnCollaborateWishlist(context *gin.Context) {

	// Create groupmembership request
	var wishlist_id = context.Param("wishlist_id")
	wishlistCollaboratorRequest := models.WishlistCollaboratorDeletionRequest{}
	if err := context.ShouldBindJSON(&wishlistCollaboratorRequest); err != nil {
		log.Println("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to get UserID from headers. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get UserID from headers."})
		context.Abort()
		return
	}

	// Parse group id
	wishlist_id_int, err := uuid.Parse(wishlist_id)
	if err != nil {
		log.Println("Failed to parse wishlist ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wishlist ID."})
		context.Abort()
		return
	}

	// Verify collaboration exists
	collaborationStatus, err := database.VerifyWishlistCollaboratorToWishlist(wishlist_id_int, wishlistCollaboratorRequest.User)
	if err != nil {
		log.Println("Failed to verify wishlist collaborator status. Error: " + err.Error())
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
		log.Println("Failed to verify wishlist owner. Error: " + err.Error())
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
		log.Println("Failed to get collaboration in database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get collaboration in database."})
		context.Abort()
		return
	}

	// Delete the collaboration
	err = database.DeleteWishlistCollaboratorByWishlistCollaboratorID(wishlistCollaborator.ID)
	if err != nil {
		log.Println("Failed to remove collaborator in database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove collaborator in database."})
		context.Abort()
		return
	}

	// get new wishlist list
	wishlists_with_users, err := GetWishlistObjects(UserID)
	if err != nil {
		log.Println("Failed to get new wishlist objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get new wishlist objects."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Wishlist collaborator removed.", "wishlists": wishlists_with_users})
}
