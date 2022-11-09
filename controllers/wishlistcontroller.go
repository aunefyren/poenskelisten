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

func GetWishlists(context *gin.Context) {

	// Create wishlist request
	var wishlists []models.Wishlist
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

	context.JSON(http.StatusCreated, gin.H{"wishlists": wishlists, "message": "Wishlists retrieved."})
}
