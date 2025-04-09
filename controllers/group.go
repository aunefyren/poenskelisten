package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Trim request input
	group.Name = strings.TrimSpace(group.Name)
	group.Description = strings.TrimSpace(group.Description)

	// Copy the data from the GroupCreationRequest model to the Group model
	group.Description = groupCreationRequest.Description
	group.Name = groupCreationRequest.Name
	group.ID = uuid.New()

	// Get the user ID from the Authorization header of the request
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		// If there is an error getting the user ID, return a Bad Request response
		logger.Log.Error("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	// Set the group owner to the user ID we obtained
	group.OwnerID = userID

	// Verify that the group name is not empty and has at least 5 characters
	if len(group.Name) < 5 || group.Name == "" {
		// If the group name is not valid, return a Bad Request response
		context.JSON(http.StatusBadRequest, gin.H{"error": "The name of the group must be five or more letters."})
		context.Abort()
		return
	}

	stringMatch, requirements, err := utilities.ValidateTextCharacters(group.Name)
	if err != nil {
		logger.Log.Error("Failed to validate group name text string. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
		context.Abort()
		return
	} else if !stringMatch {
		logger.Log.Error("Group name text string failed validation.")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	stringMatch, requirements, err = utilities.ValidateTextCharacters(group.Description)
	if err != nil {
		logger.Log.Error("Failed to validate group description text string. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
		context.Abort()
		return
	} else if !stringMatch {
		logger.Log.Error("Group description text string failed validation.")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	// Check if owner already has a group with this name
	groupExists, err := database.VerifyIfGroupWithSameNameAndOwnerDoesNotExist(group.Name, group.OwnerID)
	if err != nil {
		logger.Log.Error("Failed to check for existing groups. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check for existing groups."})
		context.Abort()
		return
	} else if groupExists {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Owner already has a group with this name."})
		context.Abort()
		return
	}

	// Create the group in the database
	record := database.Instance.Create(&group)
	if record.Error != nil {
		logger.Log.Error("Failed to create group in database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group in database."})
		context.Abort()
		return
	}

	// Create a new instance of the GroupMembership model
	var groupMembership models.GroupMembership

	// Set the member and group ID for the new group membership
	groupMembership.MemberID = userID
	groupMembership.GroupID = group.ID
	groupMembership.ID = uuid.New()

	// Create the group membership in the database
	membershipRecord := database.Instance.Create(&groupMembership)
	if membershipRecord.Error != nil {
		// If there is an error creating the group membership, return an Internal Server Error response
		logger.Log.Error("Failed to create membership. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create membership."})
		context.Abort()
		return
	}

	// Create group memberships for all members in the group_creation_request.Members slice
	for _, memberID := range groupCreationRequest.Members {
		// Create a new instance of the GroupMembership model
		var groupMembership models.GroupMembership

		newMember, err := database.GetUserInformation(memberID)
		if err != nil {
			logger.Log.Error("Failed to get user. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user."})
			context.Abort()
			return
		}

		// Set the member and group ID for the new group membership
		groupMembership.MemberID = newMember.ID
		groupMembership.GroupID = group.ID
		groupMembership.ID = uuid.New()

		// Create the group membership in the database
		_, err = database.CreateGroupMembershipInDB(groupMembership)
		if err != nil {
			logger.Log.Error("Failed to create group memberships. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group memberships."})
			context.Abort()
			return
		}
	}

	for _, wishlistID := range groupCreationRequest.Wishlists {
		// Create a new instance of the GroupMembership model
		var wishlistMembership models.WishlistMembership

		wishlist, err := database.GetWishlist(wishlistID)
		if err != nil {
			logger.Log.Error("Failed to get wishlist. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get wishlist."})
			context.Abort()
			return
		}

		if wishlist.OwnerID != userID {
			context.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to add wishlist. You must own the wishlist."})
			context.Abort()
			return
		}

		// Set the member and group ID for the new group membership
		wishlistMembership.GroupID = group.ID
		wishlistMembership.WishlistID = wishlistID
		wishlistMembership.ID = uuid.New()

		// Create the group membership in the database
		_, err = database.CreateWishlistMembershipInDB(wishlistMembership)
		if err != nil {
			logger.Log.Error("Failed to create wishlist membership. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create wishlist membership."})
			context.Abort()
			return
		}
	}

	// Get a list of groups with the current user as the owner
	groupsWithOwner, err := GetGroupObjects(userID)
	if err != nil {
		// If there is an error getting the list of groups, return a Bad Request response
		logger.Log.Error("Failed to get group objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get group objects."})
		context.Abort()
		return
	}

	// Sort groups by creation date
	sort.Slice(groupsWithOwner, func(i, j int) bool {
		return groupsWithOwner[j].CreatedAt.Before(groupsWithOwner[i].CreatedAt)
	})

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
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
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
		logger.Log.Error("Failed to parse header. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse header."})
		context.Abort()
		return
	}

	// Parse the group ID from string to int
	groupIDInt, err := uuid.Parse(groupID)
	if err != nil {
		// If there is an error parsing the group ID, return a Bad Request response
		logger.Log.Error("Failed to parse group ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse group ID."})
		context.Abort()
		return
	}

	// Iterate over the members in the groupMembership.Members slice
	for _, memberID := range groupMembership.Members {
		// Create a new instance of the GroupMembership model
		var groupMembershipDB models.GroupMembership

		memberObject, err := database.GetUserInformation(memberID)
		if err != nil {
			logger.Log.Error("Failed to find user. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find user."})
			context.Abort()
			return
		}

		// Set the member ID for the new group membership
		groupMembershipDB.MemberID = memberObject.ID

		// Verify that the user is not already a member of the group
		membershipStatus, err := database.VerifyUserMembershipToGroup(memberObject.ID, groupIDInt)
		if err != nil {
			// If there is an error verifying the user's membership, return a Bad Request response
			logger.Log.Error("Failed to verify membership to group. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify membership to group."})
			context.Abort()
			return
		} else if membershipStatus {
			// If the user is already a member of the group, return a Bad Request response
			context.JSON(http.StatusBadRequest, gin.H{"error": "Group membership already exists."})
			context.Abort()
			return
		}

		// Verify that the group is owned by the current user
		_, err = database.GetGroupUsingGroupIDAndUserIDAsOwner(userID, groupIDInt)
		if err != nil {
			logger.Log.Error("Failed to verify ownership of group. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership of group."})
			context.Abort()
			return
		}

		// Set the group ID for the new group membership
		groupMembershipDB.GroupID = groupIDInt
		groupMembershipDB.ID = uuid.New()

		// Add the group membership to the database
		record := database.Instance.Create(&groupMembershipDB)
		if record.Error != nil {
			// If there is an error adding the group membership to the database, return an Internal Server Error response
			logger.Log.Error("Failed to create group membership in database.")
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group membership in database."})
			context.Abort()
			return
		}
	}

	// Get the updated list of groups with the current user as the owner
	groupsWithOwner, err := GetGroupObjects(userID)
	if err != nil {
		// If there is an error getting the updated list of groups, return a Bad Request response
		logger.Log.Error("Failed to get groups for user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get groups for user."})
		context.Abort()
		return
	}

	// Sort groups by creation date
	sort.Slice(groupsWithOwner, func(i, j int) bool {
		return groupsWithOwner[j].CreatedAt.Before(groupsWithOwner[i].CreatedAt)
	})

	// Return a Created response with a message indicating that the group member(s) joined successfully, and the updated list of groups
	context.JSON(http.StatusCreated, gin.H{"message": "Group member(s) joined.", "groups": groupsWithOwner})
}

// RemoveFromGroup creates a groupmembership request, gets the group ID from the URL parameter, and binds the request to a groupMembership variable.
// It then gets the user ID from the authorization header and parses the group ID as an integer.
// It then verifies the user's membership to the group, checks if the group is owned by the user, and verifies that the user is not trying to remove themselves as the group owner.
// It then deletes the group membership and gets an updated list of groups with the owner. It returns a success message and the updated list of groups.
func RemoveFromGroup(context *gin.Context) {
	// Bind groupmembership request and get group ID from URL parameter
	var groupMembershipRequest models.GroupMembershipRemovalRequest

	groupID := context.Param("group_id")
	if err := context.ShouldBindJSON(&groupMembershipRequest); err != nil {
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}
	// Get user ID from authorization header
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	// Parse group ID as integer
	groupIDInt, err := uuid.Parse(groupID)
	if err != nil {
		logger.Log.Error("Failed to parse group ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse group ID."})
		context.Abort()
		return
	}

	// Verify user membership to group
	membershipStatus, err := database.VerifyUserMembershipToGroup(groupMembershipRequest.MemberID, groupIDInt)
	if err != nil {
		logger.Log.Error("Failed to verify membership to group. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify membership to group."})
		context.Abort()
		return
	} else if !membershipStatus {
		// Return error if membership does not exist
		context.JSON(http.StatusBadRequest, gin.H{"error": "Group membership doesn't exist."})
		context.Abort()
		return
	}

	// Verify group is owned by requester
	group, err := database.GetGroupInformation(groupIDInt)
	if err != nil {
		// Return error if user is not owner of group
		logger.Log.Error("Failed to get group object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get group object."})
		context.Abort()
		return
	} else if group.OwnerID != userID {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Only owners can edit their group memberships."})
		context.Abort()
		return
	}

	if group.OwnerID == groupMembershipRequest.MemberID {
		// Return error if user is owner and trying to remove themselves
		context.JSON(http.StatusBadRequest, gin.H{"error": "Owner cannot be removed as member."})
		context.Abort()
		return
	}

	// Verify membership exists
	groupMembership, err := database.GetGroupMembershipByGroupIDAndMemberID(groupIDInt, groupMembershipRequest.MemberID)
	if err != nil {
		// Return error if membership does not exist
		logger.Log.Error("Failed to verify membership. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify membership."})
		context.Abort()
		return
	}

	// Delete group membership
	err = database.DeleteGroupMembership(groupMembership.ID)
	if err != nil {
		logger.Log.Error("Failed to delete group membership. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete group membership."})
		context.Abort()
		return
	}

	// Get updated list of groups with owner
	groupsWithOwner, err := GetGroupObjects(userID)
	if err != nil {
		logger.Log.Error("Failed to get group objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get group objects."})
		context.Abort()
		return
	}

	// Sort groups by creation date
	sort.Slice(groupsWithOwner, func(i, j int) bool {
		return groupsWithOwner[j].CreatedAt.Before(groupsWithOwner[i].CreatedAt)
	})

	// Return success message and updated list of groups
	context.JSON(http.StatusCreated, gin.H{"message": "Group member removed.", "groups": groupsWithOwner})
}

// The function is an API endpoint that allows the authenticated user to remove themselves from a group.
// The user's membership to the group is verified, and the user's ownership of the group is also verified to ensure that the user is not the owner of the group.
// If everything checks out, the function deletes the user's membership record from the database and returns a success message along with an updated list of groups with the owner to the caller.
func RemoveSelfFromGroup(context *gin.Context) {
	// Bind groupmembership request and get group ID from URL parameter
	var groupMembershipRequest models.GroupMembership
	groupID := context.Param("group_id")

	// Get user ID from authorization header
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	// Parse group ID as integer
	groupIDInt, err := uuid.Parse(groupID)
	if err != nil {
		logger.Log.Error("Failed to parse group ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse group ID."})
		context.Abort()
		return
	}

	// Verify user membership to group
	membershipStatus, err := database.VerifyUserMembershipToGroup(userID, groupIDInt)
	if err != nil {
		logger.Log.Error("Failed to verify membership to group. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify membership to group."})
		context.Abort()
		return
	} else if !membershipStatus {
		// Return error if membership does not exist
		context.JSON(http.StatusBadRequest, gin.H{"error": "Group membership doesn't exist."})
		context.Abort()
		return
	}

	// Verify group is not owned by requester
	ownershipStatus, err := database.VerifyUserOwnershipToGroup(groupMembershipRequest.MemberID, groupIDInt)
	if err != nil {
		logger.Log.Error("Failed to verify ownership of group. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership of group."})
		context.Abort()
		return
	} else if ownershipStatus {
		// Return error if membership does not exist
		context.JSON(http.StatusBadRequest, gin.H{"error": "Owners cannot remove themselves as members."})
		context.Abort()
		return
	}

	// Verify membership exists
	groupMembership, err := database.GetGroupMembershipByGroupIDAndMemberID(groupIDInt, userID)
	if err != nil {
		// Return error if membership does not exist
		logger.Log.Error("Failed to verify membership to group. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify membership to group."})
		context.Abort()
		return
	}

	// Delete group membership
	err = database.DeleteGroupMembership(groupMembership.ID)
	if err != nil {
		logger.Log.Error("Failed to delete group membership. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete group membership."})
		context.Abort()
		return
	}

	// Get updated list of groups with owner
	groupsWithOwner, err := GetGroupObjects(userID)
	if err != nil {
		logger.Log.Error("Failed to get group objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get group objects."})
		context.Abort()
		return
	}

	// Sort groups by creation date
	sort.Slice(groupsWithOwner, func(i, j int) bool {
		return groupsWithOwner[j].CreatedAt.Before(groupsWithOwner[i].CreatedAt)
	})

	// Return success message and updated list of groups
	context.JSON(http.StatusCreated, gin.H{"message": "Group left.", "groups": groupsWithOwner})
}

// The function is an API endpoint that allows the authenticated user to delete a group.
// The function first verifies that the group is owned by the user by checking the groups database table.
// If the user is the owner of the group, the function then proceeds to delete the group from the database.
// Finally, the function retrieves an updated list of groups with the owner using the GetGroupObjects function and returns a success message along with the updated list of groups to the caller.
func DeleteGroup(context *gin.Context) {

	// Bind groupmembership request and get group ID from URL parameter
	groupID := context.Param("group_id")

	// Parse group ID as integer
	groupIDInt, err := uuid.Parse(groupID)
	if err != nil {
		logger.Log.Error("Failed to parse group ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse group ID."})
		context.Abort()
		return
	}

	// Get user ID from authorization header
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	// Verify group is owned by requester
	ownershipStatus, err := database.VerifyUserOwnershipToGroup(userID, groupIDInt)
	if err != nil {
		logger.Log.Error("Failed to verify ownership of group. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership of group."})
		context.Abort()
		return
	} else if !ownershipStatus {
		// Return error if membership does not exist
		context.JSON(http.StatusBadRequest, gin.H{"error": "Only owners can edit their group memberships."})
		context.Abort()
		return
	}

	// Set the group to disabled in the database
	err = database.DeleteGroup(groupIDInt)
	if err != nil {
		logger.Log.Error("Failed to delete the group. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete the group."})
		context.Abort()
		return
	}

	// Get updated list of groups with owner
	groupsWithOwner, err := GetGroupObjects(userID)
	if err != nil {
		logger.Log.Error("Failed to get group objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get group objects."})
		context.Abort()
		return
	}

	// Sort groups by creation date
	sort.Slice(groupsWithOwner, func(i, j int) bool {
		return groupsWithOwner[j].CreatedAt.Before(groupsWithOwner[i].CreatedAt)
	})

	context.JSON(http.StatusCreated, gin.H{"message": "Group deleted.", "groups": groupsWithOwner})
}

// The function retrieves a list of groups that the authenticated user owns or is a member of.
func GetGroups(context *gin.Context) {

	// Get user ID from authorization header
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to verify login session. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session."})
		context.Abort()
		return
	}

	// Retrieve list of groups with owner
	groupsWithOwner, err := GetGroupObjects(userID)
	if err != nil {
		logger.Log.Error("Failed to get your groups. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get your groups."})
		context.Abort()
		return
	}

	ownedString, ownedOkay := context.GetQuery("owned")
	if ownedOkay {
		if ownedString == "true" {
			newGroupList := []models.GroupUser{}
			for _, group := range groupsWithOwner {
				if group.Owner.ID == userID {
					newGroupList = append(newGroupList, group)
				}
			}
			groupsWithOwner = newGroupList
		}
	}

	memberOfWishlistString, memberOfWishlistOkay := context.GetQuery("memberOfWishlistID")
	if memberOfWishlistOkay {
		memberOfWishlist, err := uuid.Parse(memberOfWishlistString)
		if err != nil {
			logger.Log.Error("Failed to parse wishlist ID. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wishlist ID."})
			context.Abort()
			return
		}

		newGroupList := []models.GroupUser{}
		for _, group := range groupsWithOwner {
			membership, err := database.VerifyGroupMembershipToWishlist(memberOfWishlist, group.ID)
			if err != nil {
				logger.Log.Error("Failed to validate group membership. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate group membership."})
				context.Abort()
				return
			} else if membership {
				newGroupList = append(newGroupList, group)
			}
		}
		groupsWithOwner = newGroupList
	}

	notAMemberOfWishlistString, notAMemberOfWishlistOkay := context.GetQuery("notAMemberOfWishlistID")
	if notAMemberOfWishlistOkay {
		notAMemberOfWishlistID, err := uuid.Parse(notAMemberOfWishlistString)
		if err != nil {
			logger.Log.Error("Failed to parse wishlist ID. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wishlist ID."})
			context.Abort()
			return
		}

		newGroupList := []models.GroupUser{}
		for _, group := range groupsWithOwner {
			membership, err := database.VerifyGroupMembershipToWishlist(notAMemberOfWishlistID, group.ID)
			if err != nil {
				logger.Log.Error("Failed to validate group membership. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate group membership."})
				context.Abort()
				return
			} else if !membership {
				newGroupList = append(newGroupList, group)
			}
		}
		groupsWithOwner = newGroupList
	}

	// Sort groups by creation date
	sort.Slice(groupsWithOwner, func(i, j int) bool {
		return groupsWithOwner[i].CreatedAt.Before(groupsWithOwner[j].CreatedAt)
	})

	// Return list of groups with owner and success message
	context.JSON(http.StatusOK, gin.H{"groups": groupsWithOwner, "message": "Groups retrieved."})
}

// The function retrieves a list of groups that the given user owns or is a member of.
func GetGroupObjects(userID uuid.UUID) ([]models.GroupUser, error) {

	// Create groups slice and groups with owner slice
	var groups []models.Group

	// Retrieve groups that the user is a member of
	groups, err := database.GetGroupsAUserIsAMemberOf(userID)
	if err != nil {
		logger.Log.Error("Failed to get groups from database. Error: " + err.Error())
		return []models.GroupUser{}, errors.New("Failed to get groups from database.")
	}

	groupObjects, err := ConvertGroupsToGroupObjects(groups)
	if err != nil {
		logger.Log.Error("Failed to convert groups to group objects. Error: " + err.Error())
		return []models.GroupUser{}, errors.New("Failed to convert groups to group objects.")
	}

	return groupObjects, nil

}

// GetGroup retrieves the specified group that the authenticated user is a member of.
func GetGroup(context *gin.Context) {

	// Bind group ID from URL parameter
	groupID := context.Param("group_id")

	// Get user ID from authorization header
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	// Parse group ID as integer
	groupIDInt, err := uuid.Parse(groupID)
	if err != nil {
		logger.Log.Error("Failed to parse group ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse group ID."})
		context.Abort()
		return
	}

	// Verify user membership to group
	membershipStatus, err := database.VerifyUserMembershipToGroup(userID, groupIDInt)
	if err != nil {
		logger.Log.Error("Failed to verify membership to group. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify membership to group."})
		context.Abort()
		return
	} else if !membershipStatus {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You are not a member of this group."})
		context.Abort()
		return
	}

	// Retrieve group object with owner
	groupWithOwner, err := GetGroupObject(userID, groupIDInt)
	if err != nil {
		logger.Log.Error("Failed process group object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed process group object."})
		context.Abort()
		return
	}

	// Return group with owner and success message
	context.JSON(http.StatusOK, gin.H{"group": groupWithOwner, "message": "Group retrieved."})

}

// GetGroupObject retrieves the specified group that the authenticated user is a member of,
// along with the group's owner and members.
func GetGroupObject(userID uuid.UUID, groupID uuid.UUID) (models.GroupUser, error) {
	var group models.Group

	// Get group
	group, err := database.GetGroupUsingGroupIDAndMembershipUsingUserID(userID, groupID)
	if err != nil {
		logger.Log.Error("Failed to get group for user. Error: " + err.Error())
		return models.GroupUser{}, errors.New("Failed to get group for user.")
	}

	groupObject, err := ConvertGroupToGroupObject(group)
	if err != nil {
		logger.Log.Error("Failed to convert group to group object. Error: " + err.Error())
		return models.GroupUser{}, errors.New("Failed to convert group to group object.")
	}

	return groupObject, nil
}

func GetGroupMembers(context *gin.Context) {

	// Create group request variables
	var groupMembershipsWithUser []models.GroupMembershipUser
	var group = context.Param("group_id")

	// Get user ID from header
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	// Parse group id for usage
	groupIDInt, err := uuid.Parse(group)
	if err != nil {
		logger.Log.Error("Failed to parse group ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse group ID."})
		context.Abort()
		return
	}

	// Verify membership does exist
	MembershipStatus, err := database.VerifyUserMembershipToGroup(userID, groupIDInt)
	if err != nil {
		logger.Log.Error("Failed to verify membership to group. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify membership to group."})
		context.Abort()
		return
	} else if !MembershipStatus {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "You are not a member of this group."})
		context.Abort()
		return
	}

	// Get group members from the group
	groupMemberships, err := database.GetGroupMembershipsFromGroup(groupIDInt)
	if err != nil {
		logger.Log.Error("Failed to get group memberships for group. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get group memberships for group."})
		context.Abort()
		return
	}

	// Add user information to each membership
	for _, membership := range groupMemberships {

		userObject, err := database.GetUserInformation(membership.MemberID)
		if err != nil {
			logger.Log.Error("Failed to get user object for group member. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user object for group member."})
			context.Abort()
			return
		}

		var groupmembershipWithUser models.GroupMembershipUser
		groupmembershipWithUser.Members = userObject
		groupmembershipWithUser.CreatedAt = membership.CreatedAt
		groupmembershipWithUser.DeletedAt = membership.DeletedAt
		groupmembershipWithUser.Enabled = membership.Enabled
		groupmembershipWithUser.Group = membership.GroupID
		groupmembershipWithUser.ID = membership.ID
		groupmembershipWithUser.UpdatedAt = membership.UpdatedAt

		groupMembershipsWithUser = append(groupMembershipsWithUser, groupmembershipWithUser)

	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"group_members": groupMembershipsWithUser, "message": "Group members retrieved."})
}

func APIUpdateGroup(context *gin.Context) {

	// Create a new instance of the Group and GroupCreationRequest models
	var group models.Group
	var groupUpdateRequest models.GroupUpdateRequest
	var groupID = context.Param("group_id")

	// Bind the incoming request body to the GroupCreationRequest model
	if err := context.ShouldBindJSON(&groupUpdateRequest); err != nil {
		// If there is an error binding the request, return a Bad Request response
		logger.Log.Error(("Failed to parse request. Error: " + err.Error()))
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Trim request input
	group.Name = strings.TrimSpace(group.Name)
	group.Description = strings.TrimSpace(group.Description)

	// Parse group id for usage
	groupIDInt, err := uuid.Parse(groupID)
	if err != nil {
		logger.Log.Error("Failed to parse group ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse group ID."})
		context.Abort()
		return
	}

	// Get the user ID from the Authorization header of the request
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		// If there is an error getting the user ID, return a Bad Request response
		logger.Log.Error(("Failed to get user ID. Error: " + err.Error()))
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	// Verify group is owned by requester
	ownershipStatus, err := database.VerifyUserOwnershipToGroup(userID, groupIDInt)
	if err != nil {
		logger.Log.Error(("Failed to verify group ownership. Error: " + err.Error()))
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify group ownership."})
		context.Abort()
		return
	} else if !ownershipStatus {
		// Return error if membership does not exist
		context.JSON(http.StatusBadRequest, gin.H{"error": "You don't own this group."})
		context.Abort()
		return
	}

	// Copy the data from the GroupUpdateRequest model to the Group model
	group.Description = groupUpdateRequest.Description
	group.Name = groupUpdateRequest.Name
	group.OwnerID = userID

	groupOriginal, err := database.GetGroupInformation(groupIDInt)
	if err != nil {
		logger.Log.Error(("Failed to find group. Error: " + err.Error()))
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find group."})
		context.Abort()
		return
	}

	if group.Name != groupOriginal.Name {

		// Verify that the group name is not empty and has at least 5 characters
		if len(group.Name) < 5 || group.Name == "" {
			// If the group name is not valid, return a Bad Request response
			context.JSON(http.StatusBadRequest, gin.H{"error": "The name of the group must be five or more letters."})
			context.Abort()
			return
		}

		stringMatch, requirements, err := utilities.ValidateTextCharacters(group.Name)
		if err != nil {
			logger.Log.Error("Failed to validate group name text string. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
			context.Abort()
			return
		} else if !stringMatch {
			logger.Log.Error("Group name text string failed validation.")
			context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
			context.Abort()
			return
		}

		// Verify that a group with the same name and owner does not already exist
		groupExists, _, err := database.VerifyGroupExistsByNameForUser(group.Name, group.OwnerID)
		if err != nil {
			logger.Log.Error(("Failed verify group name. Error: " + err.Error()))
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed verify group name."})
			context.Abort()
			return
		} else if groupExists {
			context.JSON(http.StatusBadRequest, gin.H{"error": "That group name is already in use."})
			context.Abort()
			return
		}

	}

	if group.Description != groupOriginal.Description {
		if len(group.Description) < 5 || group.Description == "" {
			// If the group desc is not valid, return a Bad Request response
			context.JSON(http.StatusBadRequest, gin.H{"error": "The description of the group must be five or more letters."})
			context.Abort()
			return
		}

		stringMatch, requirements, err := utilities.ValidateTextCharacters(group.Description)
		if err != nil {
			logger.Log.Error("Failed to validate group description text string. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
			context.Abort()
			return
		} else if !stringMatch {
			logger.Log.Error("Group description text string failed validation.")
			context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
			context.Abort()
			return
		}
	}

	err = database.UpdateGroupValuesByID(groupIDInt, group.Name, group.Description)
	if err != nil {
		logger.Log.Error(("Failed update group. Error: " + err.Error()))
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed update group."})
		context.Abort()
		return
	}

	groupObjectNew, err := GetGroupObject(userID, groupIDInt)
	if err != nil {
		logger.Log.Error(("Failed convert group to group object. Error: " + err.Error()))
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed convert group to group object."})
		context.Abort()
		return
	}

	// Return a response indicating that the group was update, along with the updated group
	context.JSON(http.StatusCreated, gin.H{"message": "Group updated.", "group": groupObjectNew})
}

func ConvertGroupToGroupObject(group models.Group) (groupObject models.GroupUser, err error) {
	err = nil
	groupObject = models.GroupUser{}

	// Add owner information to group
	userObject, err := database.GetUserInformation(group.OwnerID)
	if err != nil {
		logger.Log.Error("Failed to get user object for user ID " + group.ID.String() + ". Returning. Error: " + err.Error())
		return models.GroupUser{}, errors.New("Failed to get user object for user ID " + group.ID.String() + ".")
	}

	groupObject.CreatedAt = group.CreatedAt
	groupObject.DeletedAt = group.DeletedAt
	groupObject.Description = group.Description
	groupObject.Enabled = group.Enabled
	groupObject.ID = group.ID
	groupObject.Name = group.Name
	groupObject.Owner = userObject
	groupObject.UpdatedAt = group.UpdatedAt
	groupObject.Members = []models.User{}

	// Get group members
	groupMemberships, err := database.GetGroupMembershipsFromGroup(group.ID)
	if err != nil {
		logger.Log.Error("Failed to get group memberships for group " + group.ID.String() + ". Returning. Error: " + err.Error())
		return groupObject, errors.New("Failed to get group memberships for group " + group.ID.String() + ".")
	}

	// Add user information to each membership
	for _, membership := range groupMemberships {
		userObject, err := database.GetUserInformation(membership.MemberID)
		if err != nil {
			logger.Log.Error("Failed to get user information for group '" + group.ID.String() + "' member '" + membership.MemberID.String() + "'. Returning. Error: " + err.Error())
			return models.GroupUser{}, errors.New("Failed to get user information for group '" + group.ID.String() + "' member '" + membership.MemberID.String() + "'.")
		}

		groupObject.Members = append(groupObject.Members, userObject)
	}

	return
}

func ConvertGroupsToGroupObjects(groups []models.Group) (groupObjects []models.GroupUser, err error) {
	err = nil
	groupObjects = []models.GroupUser{}

	for _, group := range groups {
		groupObject, err := ConvertGroupToGroupObject(group)
		if err != nil {
			logger.Log.Warn("Failed to get group object for '" + group.ID.String() + "'. Skipping. Error: " + err.Error())
			continue
		}
		groupObjects = append(groupObjects, groupObject)
	}

	return
}

func APIAddWishlistsToGroup(context *gin.Context) {
	// Create a new instance of model
	var wishlistsRequest models.GroupAddWishlistsRequest
	var groupIDString = context.Param("group_id")

	if err := context.ShouldBindJSON(&wishlistsRequest); err != nil {
		// If there is an error binding the request, return a Bad Request response
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	if len(wishlistsRequest.Wishlists) < 1 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You must provide one or more wishlists."})
		context.Abort()
		return
	}

	// Get the user ID from the Authorization header of the request
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to parse header. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse header."})
		context.Abort()
		return
	}

	// Parse the group ID from string to int
	groupID, err := uuid.Parse(groupIDString)
	if err != nil {
		// If there is an error parsing the group ID, return a Bad Request response
		logger.Log.Error("Failed to parse group ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse group ID."})
		context.Abort()
		return
	}

	for _, wishlistID := range wishlistsRequest.Wishlists {
		var wishlistMembership models.WishlistMembership

		wishlist, err := database.GetWishlist(wishlistID)
		if err != nil {
			logger.Log.Error("Failed to find wishlist. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find wishlist."})
			context.Abort()
			return
		} else if wishlist.OwnerID != userID {
			context.JSON(http.StatusUnauthorized, gin.H{"error": "You can only add wishlists you own."})
			context.Abort()
			return
		}

		wishlistMembership.WishlistID = wishlistID

		membershipStatus, err := database.VerifyGroupMembershipToWishlist(wishlistID, groupID)
		if err != nil {
			logger.Log.Error("Failed to verify membership to group. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify membership to group."})
			context.Abort()
			return
		} else if membershipStatus {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Group membership already exists for wishlist."})
			context.Abort()
			return
		}

		membershipStatus, err = database.VerifyUserMembershipToGroup(userID, groupID)
		if err != nil {
			logger.Log.Error("Failed to verify membership to group. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify membership to group."})
			context.Abort()
			return
		} else if !membershipStatus {
			context.JSON(http.StatusBadRequest, gin.H{"error": "You must be a member of the group."})
			context.Abort()
			return
		}

		wishlistMembership.GroupID = groupID
		wishlistMembership.ID = uuid.New()

		_, err = database.CreateWishlistMembershipInDB(wishlistMembership)
		if err != nil {
			logger.Log.Error("Failed to create group membership for wishlist in database. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group membership for wishlist in database."})
			context.Abort()
			return
		}
	}

	groupsWithOwner, err := GetGroupObjects(userID)
	if err != nil {
		logger.Log.Error("Failed to get groups for user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get groups for user."})
		context.Abort()
		return
	}

	// Sort groups by creation date
	sort.Slice(groupsWithOwner, func(i, j int) bool {
		return groupsWithOwner[j].CreatedAt.Before(groupsWithOwner[i].CreatedAt)
	})

	// Return a Created response with a message indicating that the group member(s) joined successfully, and the updated list of groups
	context.JSON(http.StatusCreated, gin.H{"message": "Wishlists added to group.", "groups": groupsWithOwner})
}
