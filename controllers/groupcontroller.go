package controllers

import (
	"net/http"
	"poenskelisten/database"
	"poenskelisten/middlewares"
	"poenskelisten/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

func RegisterGroup(context *gin.Context) {

	// Create group request
	var group models.Group
	var group_creation_request models.GroupCreationRequest
	if err := context.ShouldBindJSON(&group_creation_request); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	group.Description = group_creation_request.Description
	group.Name = group_creation_request.Name

	// Get user ID
	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Finalize group object
	group.Owner = UserID

	// Verify group doesnt exist
	grouprecords := database.Instance.Where("`groups`.enabled = ?", 1).Where("`groups`.name = ?", group.Name).Where("`groups`.Owner = ?", group.Owner).Find(&group)
	if grouprecords.RowsAffected > 0 {
		//context.JSON(http.StatusInternalServerError, gin.H{"error": grouprecords.Error.Error()})
		context.JSON(http.StatusInternalServerError, gin.H{"error": "A group with that name already exists."})
		context.Abort()
		return
	}

	// Create group in DB
	record := database.Instance.Create(&group)
	if record.Error != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": record.Error.Error()})
		context.Abort()
		return
	}

	// Create group membership
	var groupmembership models.GroupMembership
	groupmembership.Member = UserID
	groupmembership.Group = int(group.ID)
	membershiprecord := database.Instance.Create(&groupmembership)
	if record.Error != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": membershiprecord.Error.Error()})
		context.Abort()
		return
	}

	for _, member := range group_creation_request.Members {
		// Create group membership
		var groupmembership models.GroupMembership
		groupmembership.Member = member
		groupmembership.Group = int(group.ID)
		_ = database.Instance.Create(&groupmembership)
	}

	// get new group list
	groups_with_owner, err := GetGroupObjects(UserID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Group created.", "groups": groups_with_owner})
}

func JoinGroup(context *gin.Context) {

	// Create groupmembership request
	var group_id = context.Param("group_id")
	var groupmembership models.GroupMembership
	if err := context.ShouldBindJSON(&groupmembership); err != nil {
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
	group_id_int, err := strconv.Atoi(group_id)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Verify membership doesnt exist
	MembershipStatus, err := database.VerifyUserMembershipToGroup(groupmembership.Member, group_id_int)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if MembershipStatus {
		//context.JSON(http.StatusInternalServerError, gin.H{"error": groupmembershiprecord.Error.Error()})
		context.JSON(http.StatusBadRequest, gin.H{"error": "Group membership already exists."})
		context.Abort()
		return
	}

	// Verify group is owned by requester
	var group models.Group
	grouprecord := database.Instance.Where("`groups`.enabled = ?", 1).Where("`groups`.id = ?", groupmembership.Group).Where("`groups`.owner = ?", UserID).Find(&group)
	if grouprecord.Error != nil {
		//context.JSON(http.StatusInternalServerError, gin.H{"error": grouprecord.Error.Error()})
		context.JSON(http.StatusBadRequest, gin.H{"error": "Only owners can edit their group memberships."})
		context.Abort()
		return
	}

	groupmembership.Group = group_id_int

	// Add group membership to database
	record := database.Instance.Create(&groupmembership)
	if record.Error != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": record.Error.Error()})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Group joined."})
}

func GetGroups(context *gin.Context) {

	// Get user ID
	UserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	groups_with_owner, err := GetGroupObjects(UserID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"groups": groups_with_owner, "message": "Groups retrieved."})
}

func GetGroupObjects(user_id int) ([]models.GroupUser, error) {

	// Create group request
	var groups []models.Group
	var groups_with_owner []models.GroupUser

	// Get groups
	database.Instance.Where("`groups`.enabled = ?", 1).Joins("JOIN group_memberships on group_memberships.group = groups.id").Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.member = ?", user_id).Find(&groups)

	// Add owner information to each group
	for _, group := range groups {

		group_with_owner, err := GetGroupObject(user_id, int(group.ID))
		if err != nil {
			return []models.GroupUser{}, err
		}

		groups_with_owner = append(groups_with_owner, group_with_owner)

	}

	return groups_with_owner, nil
}

func GetGroup(context *gin.Context) {
	// Create group request
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

	group_with_owner, err := GetGroupObject(UserID, group_id_int)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"group": group_with_owner, "message": "Group retrieved."})

}

func GetGroupObject(user_id int, group_id int) (models.GroupUser, error) {

	var group models.Group
	var group_memberships []models.GroupMembership

	// Get groups
	database.Instance.Where("`groups`.enabled = ?", 1).Joins("JOIN group_memberships on group_memberships.group = groups.id").Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.member = ?", user_id).Where("`group_memberships`.group = ?", group_id).Find(&group)

	// Add owner information to group
	user_object, err := database.GetUserInformation(group.Owner)
	if err != nil {
		return models.GroupUser{}, err
	}

	var group_with_owner models.GroupUser
	group_with_owner.CreatedAt = group.CreatedAt
	group_with_owner.DeletedAt = group.DeletedAt
	group_with_owner.Description = group.Description
	group_with_owner.Enabled = group.Enabled
	group_with_owner.ID = group.ID
	group_with_owner.Model = group.Model
	group_with_owner.Name = group.Name
	group_with_owner.Owner = user_object
	group_with_owner.UpdatedAt = group.UpdatedAt

	// Get group members
	database.Instance.Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.group = ?", group_id).Find(&group_memberships)

	// Add user information to each membership
	for _, membership := range group_memberships {

		user_object, err := database.GetUserInformation(membership.Member)
		if err != nil {
			return models.GroupUser{}, err
		}

		group_with_owner.Members = append(group_with_owner.Members, user_object)

	}

	return group_with_owner, nil
}

func GetGroupMembers(context *gin.Context) {

	// Create group request
	var group_memberships []models.GroupMembership
	var group_memberships_user []models.GroupMembershipUser
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

	// Get groups
	database.Instance.Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.group = ?", group).Find(&group_memberships)

	// Add user information to each membership
	for _, membership := range group_memberships {

		user_object, err := database.GetUserInformation(membership.Member)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			context.Abort()
			return
		}

		var groupmembership_with_user models.GroupMembershipUser
		groupmembership_with_user.Members = user_object
		groupmembership_with_user.CreatedAt = membership.CreatedAt
		groupmembership_with_user.DeletedAt = membership.DeletedAt
		groupmembership_with_user.Enabled = membership.Enabled
		groupmembership_with_user.Group = membership.Group
		groupmembership_with_user.ID = membership.ID
		groupmembership_with_user.Model = membership.Model
		groupmembership_with_user.UpdatedAt = membership.UpdatedAt

		group_memberships_user = append(group_memberships_user, groupmembership_with_user)

	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"group_members": group_memberships_user, "message": "Group members retrieved."})
}
