package controllers

import (
	"net/http"
	"poenskelisten/database"
	"poenskelisten/middlewares"
	"poenskelisten/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

func RegisterWishlist(context *gin.Context) {

	// Create wishlist request
	var wishlist models.Wishlist
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

	// Finalize wishlist object
	wishlist.Owner = UserID

	// Verify wishlist doesnt exist
	wishlistrecords := database.Instance.Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.name = ?", wishlist.Name).Where("`wishlists`.Owner = ?", wishlist.Owner).Find(&wishlist)
	if wishlistrecords.RowsAffected > 0 {
		//context.JSON(http.StatusInternalServerError, gin.H{"error": grouprecords.Error.Error()})
		context.JSON(http.StatusInternalServerError, gin.H{"error": "A wishlist with that name already exists."})
		context.Abort()
		return
	}

	// Create wishlist in DB
	record := database.Instance.Create(&wishlist)
	if record.Error != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": record.Error.Error()})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Wishlist created."})
}

func GetWishlistsFromGroup(context *gin.Context) {

	// Create wishlist request
	var wishlists []models.Wishlist
	var wishlists_with_users []models.WishlistUser
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

	// Verify membership doesnt exist
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

	// Verify group doesnt exist
	database.Instance.Where("`wishlists`.enabled = ?", 1).Joins("JOIN group_memberships on group_memberships.id = wishlists.group").Where("`group_memberships`.group = ?", group).Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.member = ?", UserID).Joins("JOIN groups on group_memberships.group = groups.id").Where("`groups`.enabled = ?", 1).Find(&wishlists)

	// Add user information to each wishlist
	for _, wishlist := range wishlists {

		members, err := database.GetUserMembersFromGroup(wishlist.Group)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			context.Abort()
			return
		}

		owner, err := database.GetUserInformation(wishlist.Owner)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			context.Abort()
			return
		}

		var wishlist_with_user models.WishlistUser
		wishlist_with_user.CreatedAt = wishlist.CreatedAt
		wishlist_with_user.Date = wishlist.Date
		wishlist_with_user.DeletedAt = wishlist.DeletedAt
		wishlist_with_user.Description = wishlist.Description
		wishlist_with_user.Enabled = wishlist.Enabled
		wishlist_with_user.Group = wishlist.Group
		wishlist_with_user.ID = wishlist.ID
		wishlist_with_user.Members = members
		wishlist_with_user.Owner = owner
		wishlist_with_user.Model = wishlist.Model
		wishlist_with_user.Name = wishlist.Name
		wishlist_with_user.UpdatedAt = wishlist.UpdatedAt

		wishlists_with_users = append(wishlists_with_users, wishlist_with_user)

	}

	context.JSON(http.StatusCreated, gin.H{"wishlists": wishlists_with_users, "message": "Wishlists retrieved."})
}

func GetWishlistFromGroup(context *gin.Context) {

	// Create wishlist request
	var wishlist models.Wishlist
	var wishlist_with_user models.WishlistUser
	var group = context.Param("group_id")
	var wishlist_id = context.Param("wishlist_id")

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

	// Verify membership doesnt exist
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

	// Verify group doesnt exist
	database.Instance.Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.id = ?", wishlist_id).Joins("JOIN group_memberships on group_memberships.id = wishlists.group").Where("`group_memberships`.group = ?", group).Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.member = ?", UserID).Joins("JOIN groups on group_memberships.group = groups.id").Where("`groups`.enabled = ?", 1).Find(&wishlist)

	// Add user information to each wishlist
	members, err := database.GetUserMembersFromGroup(int(wishlist.Group))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	owner, err := database.GetUserInformation(wishlist.Owner)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	wishlist_with_user.CreatedAt = wishlist.CreatedAt
	wishlist_with_user.Date = wishlist.Date
	wishlist_with_user.DeletedAt = wishlist.DeletedAt
	wishlist_with_user.Description = wishlist.Description
	wishlist_with_user.Enabled = wishlist.Enabled
	wishlist_with_user.Group = wishlist.Group
	wishlist_with_user.ID = wishlist.ID
	wishlist_with_user.Members = members
	wishlist_with_user.Owner = owner
	wishlist_with_user.Model = wishlist.Model
	wishlist_with_user.Name = wishlist.Name
	wishlist_with_user.UpdatedAt = wishlist.UpdatedAt

	context.JSON(http.StatusCreated, gin.H{"wishlist": wishlist_with_user, "message": "Wishlist retrieved."})
}
