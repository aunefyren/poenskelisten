package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// The RegisterGroup function creates a new group with the specified name and owner, and adds the specified members to the group.
// It returns a response indicating whether the group was created successfully and, if so, the updated list of groups with the current user as the owner.
func RegisterGroup(context *gin.Context) {
	// Create a new instance of the Group and GroupCreationRequest models
	var group models.Group
	var groupCreationRequest models.GroupCreationRequest

	// Bind the incoming request body to the GroupCreationRequest model
	if err := context.ShouldBindJSON(&groupCreationRequest); err != nil {
		// If there is an error binding the request, return a Bad Request response
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Copy the data from the GroupCreationRequest model to the Group model
	group.Description = groupCreationRequest.Description
	group.Name = groupCreationRequest.Name

	// Get the user ID from the Authorization header of the request
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		// If there is an error getting the user ID, return a Bad Request response
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Set the group owner to the user ID we obtained
	group.Owner = userID

	// Verify that the group name is not empty and has at least 5 characters
	if len(group.Name) < 5 || group.Name == "" {
		// If the group name is not valid, return a Bad Request response
		context.JSON(http.StatusBadRequest, gin.H{"error": "The name of the group must be five or more letters."})
		context.Abort()
		return
	}

	// Verify that a group with the same name and owner does not already exist
	groupRecords := database.Instance.Where("`groups`.enabled = ?", 1).Where("`groups`.name = ?", group.Name).Where("`groups`.Owner = ?", group.Owner).Find(&group)
	if groupRecords.RowsAffected > 0 {
		// If a group with the same name and owner already exists, return an Internal Server Error response
		context.JSON(http.StatusInternalServerError, gin.H{"error": "A group with that name already exists."})
		context.Abort()
		return
	}

	// Create the group in the database
	record := database.Instance.Create(&group)
	if record.Error != nil {
		// If there is an error creating the group, return an Internal Server Error response
		context.JSON(http.StatusInternalServerError, gin.H{"error": record.Error.Error()})
		context.Abort()
		return
	}

	// Create a new instance of the GroupMembership model
	var groupMembership models.GroupMembership

	// Set the member and group ID for the new group membership
	groupMembership.Member = userID
	groupMembership.Group = int(group.ID)

	// Create the group membership in the database
	membershipRecord := database.Instance.Create(&groupMembership)
	if membershipRecord.Error != nil {
		// If there is an error creating the group membership, return an Internal Server Error response
		context.JSON(http.StatusInternalServerError, gin.H{"error": membershipRecord.Error.Error()})
		context.Abort()
		return
	}

	// Create group memberships for all members in the group_creation_request.Members slice
	for _, member := range groupCreationRequest.Members {
		// Create a new instance of the GroupMembership model
		var groupMembership models.GroupMembership

		// Set the member and group ID for the new group membership
		groupMembership.Member = member
		groupMembership.Group = int(group.ID)

		// Create the group membership in the database
		_ = database.Instance.Create(&groupMembership)
	}

	// Get a list of groups with the current user as the owner
	groupsWithOwner, err := GetGroupObjects(userID)
	if err != nil {
		// If there is an error getting the list of groups, return a Bad Request response
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Return a response indicating that the group was created, along with the updated list of groups
	context.JSON(http.StatusCreated, gin.H{"message": "Group created.", "groups": groupsWithOwner})
}

// The JoinGroup function adds the specified members to the group with the given ID.
// It returns a response indicating whether the members were added successfully and, if so, the updated list of groups with the current user as the owner.
func JoinGroup(context *gin.Context) {
	// Get the group ID from the URL parameters
	var groupID = context.Param("group_id")

	// Create a new instance of the GroupMembershipCreationRequest model
	var groupMembership models.GroupMembershipCreationRequest

	// Bind the incoming request body to the GroupMembershipCreationRequest model
	if err := context.ShouldBindJSON(&groupMembership); err != nil {
		// If there is an error binding the request, return a Bad Request response
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Verify that the Members slice in the request contains at least one user
	if len(groupMembership.Members) < 1 {
		// If the Members slice is empty, return a Bad Request response
		context.JSON(http.StatusBadRequest, gin.H{"error": "You must provide one or more users."})
		context.Abort()
		return
	}

	// Get the user ID from the Authorization header of the request
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		// If there is an error getting the user ID, return a Bad Request response
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Parse the group ID from string to int
	groupIDInt, err := strconv.Atoi(groupID)
	if err != nil {
		// If there is an error parsing the group ID, return a Bad Request response
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Iterate over the members in the groupMembership.Members slice
	for _, member := range groupMembership.Members {
		// Create a new instance of the GroupMembership model
		var groupMembershipDB models.GroupMembership

		// Set the member ID for the new group membership
		groupMembershipDB.Member = member

		// Verify that the user exists
		_, err := database.GetUserInformation(member)
		if err != nil {
			// If the user does not exist, return a Bad Request response
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			context.Abort()
			return
		}

		// Verify that the user is not already a member of the group
		membershipStatus, err := database.VerifyUserMembershipToGroup(member, groupIDInt)
		if err != nil {
			// If there is an error verifying the user's membership, return a Bad Request response
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			context.Abort()
			return
		} else if membershipStatus {
			// If the user is already a member of the group, return a Bad Request response
			context.JSON(http.StatusBadRequest, gin.H{"error": "Group membership already exists."})
			context.Abort()
			return
		}

		// Verify that the group is owned by the current user
		var group models.Group
		groupRecord := database.Instance.Where("`groups`.enabled = ?", 1).Where("`groups`.id = ?", groupIDInt).Where("`groups`.owner = ?", userID).Find(&group)
		if groupRecord.Error != nil {
			// If the group is not owned by the current user, return a Bad Request response
			context.JSON(http.StatusBadRequest, gin.H{"error": "Only owners can edit their group memberships."})
			context.Abort()
			return
		}

		// Set the group ID for the new group membership
		groupMembershipDB.Group = groupIDInt

		// Add the group membership to the database
		record := database.Instance.Create(&groupMembershipDB)
		if record.Error != nil {
			// If there is an error adding the group membership to the database, return an Internal Server Error response
			context.JSON(http.StatusInternalServerError, gin.H{"error": record.Error.Error()})
			context.Abort()
			return
		}
	}

	// Get the updated list of groups with the current user as the owner
	groupsWithOwner, err := GetGroupObjects(userID)
	if err != nil {
		// If there is an error getting the updated list of groups, return a Bad Request response
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Return a Created response with a message indicating that the group member(s) joined successfully, and the updated list of groups
	context.JSON(http.StatusCreated, gin.H{"message": "Group member(s) joined.", "groups": groupsWithOwner})

}

func RemoveFromGroup(context *gin.Context) {

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
	} else if !MembershipStatus {
		//context.JSON(http.StatusInternalServerError, gin.H{"error": groupmembershiprecord.Error.Error()})
		context.JSON(http.StatusBadRequest, gin.H{"error": "Group membership doesn't exist."})
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

	if UserID == groupmembership.Member {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Owner cannot be removed as member."})
		context.Abort()
		return
	}

	grouprmembershipecord := database.Instance.Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.group = ?", group_id_int).Where("`group_memberships`.member = ?", groupmembership.Member).Find(&groupmembership)
	if grouprmembershipecord.Error != nil {
		//context.JSON(http.StatusInternalServerError, gin.H{"error": grouprecord.Error.Error()})
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify membership."})
		context.Abort()
		return
	}

	err = database.DeleteGroupMembership(int(groupmembership.ID))
	if grouprmembershipecord.Error != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// get new group list
	groups_with_owner, err := GetGroupObjects(UserID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Group member removed.", "groups": groups_with_owner})
}

func DeleteGroup(context *gin.Context) {

	// Create groupmembership request
	var group_id = context.Param("group_id")
	var groupmembership models.GroupMembership

	// Parse group id
	group_id_int, err := strconv.Atoi(group_id)
	if err != nil {
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

	// Verify group is owned by requester
	var group models.Group
	grouprecord := database.Instance.Where("`groups`.enabled = ?", 1).Where("`groups`.id = ?", groupmembership.Group).Where("`groups`.owner = ?", UserID).Find(&group)
	if grouprecord.Error != nil {
		//context.JSON(http.StatusInternalServerError, gin.H{"error": grouprecord.Error.Error()})
		context.JSON(http.StatusBadRequest, gin.H{"error": "Only owners can edit their group memberships."})
		context.Abort()
		return
	}

	err = database.DeleteGroup(group_id_int)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// get new group list
	groups_with_owner, err := GetGroupObjects(UserID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Group deleted.", "groups": groups_with_owner})
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
	grouprecords := database.Instance.Where("`groups`.enabled = ?", 1).Joins("JOIN group_memberships on group_memberships.group = groups.id").Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.member = ?", user_id).Find(&groups)

	if grouprecords.Error != nil {
		return []models.GroupUser{}, grouprecords.Error
	} else if grouprecords.RowsAffected == 0 {
		return []models.GroupUser{}, nil
	}

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
	grouprecords := database.Instance.Where("`groups`.enabled = ?", 1).Joins("JOIN group_memberships on group_memberships.group = groups.id").Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.member = ?", user_id).Where("`group_memberships`.group = ?", group_id).Find(&group)

	if grouprecords.Error != nil {
		return models.GroupUser{}, grouprecords.Error
	} else if grouprecords.RowsAffected == 0 {
		return models.GroupUser{}, nil
	}

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
			log.Println("Failed to get user information for group " + strconv.Itoa(group_id) + " members.")
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
