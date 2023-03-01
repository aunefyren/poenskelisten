package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
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

	group_membership := false
	group_id := 0
	if wishlist.Group != 0 {
		group_membership = true

		// Parse group id
		group_id = wishlist.Group

		MembershipStatus, err := database.VerifyUserMembershipToGroup(UserID, group_id)
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

	}

	if len(wishlist.Name) < 5 || wishlist.Name == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "The name of the wishlist must be five or more letters."})
		context.Abort()
		return
	}

	wishlistdb.Date, err = time.Parse("2006-01-02T15:04:05.000Z", wishlist.Date)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	if now.After(wishlistdb.Date) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "The date of the wishlist must be in the future."})
		context.Abort()
		return
	}

	unique_wish_name, err := database.VerifyUniqueWishlistNameForUser(wishlist.Name, UserID)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if !unique_wish_name {
		context.JSON(http.StatusBadRequest, gin.H{"error": "There is already a wishlist with that name on your profile."})
		context.Abort()
		return
	}

	// Finalize wishlist object
	wishlistdb.Owner = UserID
	wishlistdb.Description = wishlist.Description
	wishlistdb.Name = wishlist.Name

	// Create wishlist in DB
	record := database.Instance.Create(&wishlistdb)
	if record.Error != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": record.Error.Error()})
		context.Abort()
		return
	}

	var wishlists_with_users []models.WishlistUser

	// If a group was referenced, create the wishlist membership
	if group_membership == true {
		var wishlistmembershipdb models.WishlistMembership
		wishlistmembershipdb.Group = group_id
		wishlistmembershipdb.Wishlist = int(wishlistdb.ID)

		// Add group membership to database
		record := database.Instance.Create(&wishlistmembershipdb)
		if record.Error != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": record.Error.Error()})
			context.Abort()
			return
		}

		wishlists_with_users, err = GetWishlistObjectsFromGroup(group_id, UserID)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			context.Abort()
			return
		}

	} else {

		wishlists_with_users, err = GetWishlistObjects(UserID)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			context.Abort()
			return
		}

	}

	context.JSON(http.StatusCreated, gin.H{"message": "Wishlist created.", "wishlists": wishlists_with_users})
}

func GetWishlistsFromGroup(context *gin.Context) {

	// Create wishlist request
	var group = context.Param("group_id")

	// Get user ID
	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Parse group id
	group_id_int, err := strconv.Atoi(group)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	wishlists_with_users, err := GetWishlistObjectsFromGroup(group_id_int, UserID)
	if err != nil {
		log.Println("Failed to get wishlists for group. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get wishlists for group."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"wishlists": wishlists_with_users, "message": "Wishlists retrieved."})

}

func DeleteWishlist(context *gin.Context) {

	var wishlist = context.Param("wishlist_id")

	// Get user ID
	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Parse wishlist id
	wishlist_id_int, err := strconv.Atoi(wishlist)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	wishlists_with_users, err := GetWishlistObjects(UserID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"wishlists": wishlists_with_users, "message": "Wishlist deleted."})

}

func GetWishlistObjectsFromGroup(group_id int, RequestUserID int) ([]models.WishlistUser, error) {

	var wishlists_with_users []models.WishlistUser

	wishlists, err := database.GetWishlistsFromGroup(group_id)
	if err != nil {
		return []models.WishlistUser{}, err
	}

	// Add user information to each wishlist
	for _, wishlist := range wishlists {

		owner, err := database.GetUserInformation(wishlist.Owner)
		if err != nil {
			log.Println("Failed to get user information for wishlist owner. Returning. Error: " + err.Error())
			return []models.WishlistUser{}, err
		}

		groups, err := database.GetGroupMembersFromWishlist(int(wishlist.ID))
		if err != nil {
			log.Println("Failed to get group memberships towards wishlist. Returning. Error: " + err.Error())
			return []models.WishlistUser{}, err
		}

		var groups_with_users []models.GroupUser

		for _, group := range groups {

			var group_with_user models.GroupUser

			members, err := database.GetUserMembersFromGroup(int(group.ID))
			if err != nil {
				log.Println("Failed to get group members for group. Returning. Error: " + err.Error())
				return []models.WishlistUser{}, err
			}

			owner, err := database.GetUserInformation(group.Owner)
			if err != nil {
				log.Println("Failed to get owner for group '" + strconv.Itoa(group.Owner) + "'. Skipping wishlist. Error: " + err.Error())
				continue
			}

			group_with_user.CreatedAt = group.CreatedAt
			group_with_user.UpdatedAt = group.UpdatedAt
			group_with_user.Description = group.Description
			group_with_user.Enabled = group.Enabled
			group_with_user.ID = group.ID
			group_with_user.Members = members
			group_with_user.Model = group.Model
			group_with_user.Name = group.Name
			group_with_user.Owner = owner
			group_with_user.UpdatedAt = group.UpdatedAt

			groups_with_users = append(groups_with_users, group_with_user)

		}

		var wishlist_with_user models.WishlistUser
		wishlist_with_user.CreatedAt = wishlist.CreatedAt
		wishlist_with_user.Date = wishlist.Date
		wishlist_with_user.DeletedAt = wishlist.DeletedAt
		wishlist_with_user.Description = wishlist.Description
		wishlist_with_user.Enabled = wishlist.Enabled
		wishlist_with_user.ID = wishlist.ID
		wishlist_with_user.Members = groups_with_users
		wishlist_with_user.Owner = owner
		wishlist_with_user.Model = wishlist.Model
		wishlist_with_user.Name = wishlist.Name
		wishlist_with_user.UpdatedAt = wishlist.UpdatedAt

		// Get wishes
		wishlist_id_int := int(wishlist.ID)
		wishes, err := database.GetWishesFromWishlist(wishlist_id_int, RequestUserID)
		if err != nil {
			log.Println("Failed to get wishes for wishlist '" + strconv.Itoa(wishlist_id_int) + "'. Returning. Error: " + err.Error())
			return []models.WishlistUser{}, err
		}
		wishlist_with_user.Wishes = wishes

		wishlists_with_users = append(wishlists_with_users, wishlist_with_user)

	}

	if len(wishlists_with_users) == 0 {
		wishlists_with_users = []models.WishlistUser{}
	}

	return wishlists_with_users, nil

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
	wishlist_id_int, err := strconv.Atoi(wishlist_id)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

func GetWishlistObject(WishlistID int, RequestUserID int) (models.WishlistUser, error) {

	var wishlist_with_user models.WishlistUser

	wishlist, err := database.GetWishlist(WishlistID)
	if err != nil {
		log.Println("Failed to get wishlist '" + strconv.Itoa(WishlistID) + "' from DB. Returning. Error: " + err.Error())
		return models.WishlistUser{}, err
	}

	groups, err := database.GetGroupMembersFromWishlist(int(wishlist.ID))
	if err != nil {
		return models.WishlistUser{}, err
	}

	var groups_with_users []models.GroupUser

	for _, group := range groups {

		var group_with_user models.GroupUser

		members, err := database.GetUserMembersFromGroup(int(group.ID))
		if err != nil {
			log.Println("Failed to get members to group '" + strconv.Itoa(int(group.ID)) + "'. Returning. Error: " + err.Error())
			return models.WishlistUser{}, err
		}

		owner, err := database.GetUserInformation(group.Owner)
		if err != nil {
			log.Println("Failed to get information of owner '" + strconv.Itoa(int(group.Owner)) + "' of group '" + strconv.Itoa(int(group.ID)) + "'. Skipping. Error: " + err.Error())
			continue
		}

		group_with_user.CreatedAt = group.CreatedAt
		group_with_user.UpdatedAt = group.UpdatedAt
		group_with_user.Description = group.Description
		group_with_user.Enabled = group.Enabled
		group_with_user.ID = group.ID
		group_with_user.Members = members
		group_with_user.Model = group.Model
		group_with_user.Name = group.Name
		group_with_user.Owner = owner
		group_with_user.UpdatedAt = group.UpdatedAt

		groups_with_users = append(groups_with_users, group_with_user)

	}

	if len(groups_with_users) < 1 {
		groups_with_users = []models.GroupUser{}
	}

	owner, err := database.GetUserInformation(wishlist.Owner)
	if err != nil {
		log.Println("Failed to get information of wishlist owner '" + strconv.Itoa(int(wishlist.Owner)) + "'. Returning. Error: " + err.Error())
		return models.WishlistUser{}, err
	}

	wishlist_with_user.CreatedAt = wishlist.CreatedAt
	wishlist_with_user.Date = wishlist.Date
	wishlist_with_user.DeletedAt = wishlist.DeletedAt
	wishlist_with_user.Description = wishlist.Description
	wishlist_with_user.Enabled = wishlist.Enabled
	wishlist_with_user.ID = wishlist.ID
	wishlist_with_user.Members = groups_with_users
	wishlist_with_user.Owner = owner
	wishlist_with_user.Model = wishlist.Model
	wishlist_with_user.Name = wishlist.Name
	wishlist_with_user.UpdatedAt = wishlist.UpdatedAt

	// Get wishes
	wishes, err := database.GetWishesFromWishlist(WishlistID, RequestUserID)
	if err != nil {
		return models.WishlistUser{}, err
	}

	wishlist_with_user.Wishes = wishes

	return wishlist_with_user, nil

}

func GetWishlists(context *gin.Context) {

	// Get user ID
	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	wishlists_with_users, err := GetWishlistObjects(UserID)

	context.JSON(http.StatusOK, gin.H{"wishlists": wishlists_with_users, "message": "Wishlists retrieved."})
}

func GetWishlistObjects(UserID int) ([]models.WishlistUser, error) {

	var wishlists_with_users []models.WishlistUser

	wishlists, err := database.GetOwnedWishlists(UserID)
	if err != nil {
		return []models.WishlistUser{}, err
	}

	// Add user information to each wishlist
	for _, wishlist := range wishlists {

		groups, err := database.GetGroupMembersFromWishlist(int(wishlist.ID))
		if err != nil {
			return []models.WishlistUser{}, err
		}

		var groups_with_users []models.GroupUser

		for _, group := range groups {

			var group_with_user models.GroupUser

			members, err := database.GetUserMembersFromGroup(int(group.ID))
			if err != nil {
				return []models.WishlistUser{}, err
			}

			owner, err := database.GetUserInformation(group.Owner)
			if err != nil {
				return []models.WishlistUser{}, err
			}

			group_with_user.CreatedAt = group.CreatedAt
			group_with_user.UpdatedAt = group.UpdatedAt
			group_with_user.Description = group.Description
			group_with_user.Enabled = group.Enabled
			group_with_user.ID = group.ID
			group_with_user.Members = members
			group_with_user.Model = group.Model
			group_with_user.Name = group.Name
			group_with_user.Owner = owner
			group_with_user.UpdatedAt = group.UpdatedAt

			groups_with_users = append(groups_with_users, group_with_user)

		}

		if len(groups_with_users) < 1 {
			groups_with_users = []models.GroupUser{}
		}

		owner, err := database.GetUserInformation(wishlist.Owner)
		if err != nil {
			return []models.WishlistUser{}, err
		}

		var wishlist_with_user models.WishlistUser
		wishlist_with_user.CreatedAt = wishlist.CreatedAt
		wishlist_with_user.Date = wishlist.Date
		wishlist_with_user.DeletedAt = wishlist.DeletedAt
		wishlist_with_user.Description = wishlist.Description
		wishlist_with_user.Enabled = wishlist.Enabled
		wishlist_with_user.ID = wishlist.ID
		wishlist_with_user.Members = groups_with_users
		wishlist_with_user.Owner = owner
		wishlist_with_user.Model = wishlist.Model
		wishlist_with_user.Name = wishlist.Name
		wishlist_with_user.UpdatedAt = wishlist.UpdatedAt

		// Get wishes
		wishlist_id_int := int(wishlist.ID)
		wishes, err := database.GetWishesFromWishlist(wishlist_id_int, UserID)
		if err != nil {
			return []models.WishlistUser{}, err
		}
		wishlist_with_user.Wishes = wishes

		wishlists_with_users = append(wishlists_with_users, wishlist_with_user)

	}

	if len(wishlists_with_users) == 0 {
		wishlists_with_users = []models.WishlistUser{}
	}

	return wishlists_with_users, nil

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
	wishlist_id_int, err := strconv.Atoi(wishlist_id)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	for _, Group := range wishlistmembership.Groups {

		var wishlistmembershipdb models.WishlistMembership
		wishlistmembershipdb.Group = Group

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
		var wishlist models.Wishlist
		wishlistrecord := database.Instance.Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.id = ?", wishlist_id_int).Where("`wishlists`.owner = ?", UserID).Find(&wishlist)
		if wishlistrecord.Error != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Only owners can edit their wishlist."})
			context.Abort()
			return
		}

		wishlistmembershipdb.Wishlist = wishlist_id_int

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
	var wishlistmembership models.WishlistMembership
	if err := context.ShouldBindJSON(&wishlistmembership); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	wishlist_id_int, err := strconv.Atoi(wishlist_id)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Verify membership exists
	MembershipStatus, err := database.VerifyGroupMembershipToWishlist(wishlist_id_int, wishlistmembership.Group)
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

	// Verify wishlist is owned by requester
	var wishlist models.Wishlist
	wishlistrecord := database.Instance.Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.id = ?", wishlist_id_int).Where("`wishlists`.owner = ?", UserID).Find(&wishlist)
	if wishlistrecord.Error != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Only owners can edit their wishlist memberships."})
		context.Abort()
		return
	}

	// Get the membership id
	wishlistmembershiprecord := database.Instance.Where("`wishlist_memberships`.enabled = ?", 1).Where("`wishlist_memberships`.wishlist = ?", wishlist_id_int).Where("`wishlist_memberships`.group = ?", wishlistmembership.Group).Find(&wishlistmembership)
	if wishlistmembershiprecord.Error != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify membership."})
		context.Abort()
		return
	}

	// Delete the membership
	err = database.DeleteWishlistMembership(int(wishlistmembership.ID))
	if wishlistmembershiprecord.Error != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// get new group list
	wishlists_with_users, err := GetWishlistObjects(UserID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Group member removed.", "wishlists": wishlists_with_users})
}
