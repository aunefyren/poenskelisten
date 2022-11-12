package controllers

import (
	"net/http"
	"poenskelisten/database"
	"poenskelisten/middlewares"
	"poenskelisten/models"
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

	// Verify membership exists
	MembershipStatus, err := database.VerifyUserMembershipToGroup(UserID, wishlist.Group)
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

	if len(wishlist.Name) < 5 || wishlist.Name == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "The name of the wishlist must be five or more letters."})
		context.Abort()
		return
	}

	if now.After(wishlist.Date) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "The date of the wishlist must be in the future."})
		context.Abort()
		return
	}

	unique_wish_name, err := database.VerifyUniqueWishlistNameinGroup(wishlist.Name, wishlist.Group)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if !unique_wish_name {
		context.JSON(http.StatusBadRequest, gin.H{"error": "There is already a wishlist with that name in this group."})
		context.Abort()
		return
	}

	// Finalize wishlist object
	wishlistdb.Owner = UserID
	wishlistdb.Group = wishlist.Group
	wishlistdb.Date = wishlist.Date
	wishlistdb.Description = wishlist.Description
	wishlistdb.Name = wishlist.Name

	// Verify wishlist doesnt exist
	wishlistrecords := database.Instance.Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.name = ?", wishlistdb.Name).Where("`wishlists`.Owner = ?", wishlistdb.Owner).Find(&wishlistdb)
	if wishlistrecords.RowsAffected > 0 {
		//context.JSON(http.StatusInternalServerError, gin.H{"error": grouprecords.Error.Error()})
		context.JSON(http.StatusInternalServerError, gin.H{"error": "A wishlist with that name already exists."})
		context.Abort()
		return
	}

	// Create wishlist in DB
	record := database.Instance.Create(&wishlistdb)
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

	// Verify membership exists
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

		// Get wishes
		wishlist_id_int := int(wishlist.ID)
		wishes, err := database.GetWishesFromWishlist(wishlist_id_int)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			context.Abort()
			return
		}
		wishlist_with_user.Wishes = wishes

		wishlists_with_users = append(wishlists_with_users, wishlist_with_user)

	}

	context.JSON(http.StatusOK, gin.H{"wishlists": wishlists_with_users, "message": "Wishlists retrieved."})
}

func GetWishlist(context *gin.Context) {

	// Create wishlist request
	var wishlist models.Wishlist
	var wishlist_with_user models.WishlistUser
	var wishlist_id = context.Param("wishlist_id")

	// Get user ID
	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Verify group doesnt exist
	database.Instance.Where("`wishlists`.enabled = ?", 1).Where("`wishlists`.id = ?", wishlist_id).Joins("JOIN group_memberships on group_memberships.id = wishlists.group").Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.member = ?", UserID).Joins("JOIN groups on group_memberships.group = groups.id").Where("`groups`.enabled = ?", 1).Find(&wishlist)

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

	// Get wishes
	wishlist_id_int := int(wishlist.ID)
	wishes, err := database.GetWishesFromWishlist(wishlist_id_int)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}
	wishlist_with_user.Wishes = wishes

	context.JSON(http.StatusOK, gin.H{"wishlist": wishlist_with_user, "message": "Wishlist retrieved."})
}

func GetWishlists(context *gin.Context) {

	// Create wishlist request
	var wishlists []models.Wishlist
	var wishlists_with_users []models.WishlistUser

	// Get user ID
	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Verify group doesnt exist
	database.Instance.Where("`wishlists`.enabled = ?", 1).Joins("JOIN group_memberships on group_memberships.id = wishlists.group").Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.member = ?", UserID).Joins("JOIN groups on group_memberships.group = groups.id").Where("`groups`.enabled = ?", 1).Find(&wishlists)

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

		// Get wishes
		wishlist_id_int := int(wishlist.ID)
		wishes, err := database.GetWishesFromWishlist(wishlist_id_int)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			context.Abort()
			return
		}
		wishlist_with_user.Wishes = wishes

		wishlists_with_users = append(wishlists_with_users, wishlist_with_user)

	}

	context.JSON(http.StatusOK, gin.H{"wishlists": wishlists_with_users, "message": "Wishlists retrieved."})
}
