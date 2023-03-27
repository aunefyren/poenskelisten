package database

import (
	"aunefyren/poenskelisten/models"
	"errors"
)

func VerifyGroupExistsByNameForUser(groupName string, groupOwnerID int) (bool, models.Group, error) {

	var groupStruct models.Group

	groupRecords := Instance.Where("`groups`.enabled = ?", 1).Where("`groups`.name = ?", groupName).Where("`groups`.Owner = ?", groupOwnerID).Find(&groupStruct)

	if groupRecords.Error != nil {
		return false, models.Group{}, groupRecords.Error
	} else if groupRecords.RowsAffected != 1 {
		return false, groupStruct, nil
	}

	return true, groupStruct, nil
}

// Get group by Group ID
func GetGroupInformation(GroupID int) (models.Group, error) {
	var group models.Group
	grouprecord := Instance.Where("`groups`.enabled = ?", 1).Where("`groups`.id = ?", GroupID).Find(&group)
	if grouprecord.Error != nil {
		return models.Group{}, grouprecord.Error
	} else if grouprecord.RowsAffected != 1 {
		return models.Group{}, errors.New("Failed to find correct group in DB.")
	}

	return group, nil
}

// Update values on group object in DB
func UpdateGroupValuesByID(groupID int, groupName string, groupDesc string) error {

	var group models.Group

	groupRecord := Instance.Model(group).Where("`groups`.enabled = ?", 1).Where("`groups`.ID = ?", groupID).Update("name", groupName)
	if groupRecord.Error != nil {
		return groupRecord.Error
	}
	if groupRecord.RowsAffected != 1 {
		return errors.New("Name not changed in database.")
	}

	groupRecord = Instance.Model(group).Where("`groups`.enabled = ?", 1).Where("`groups`.ID = ?", groupID).Update("description", groupDesc)
	if groupRecord.Error != nil {
		return groupRecord.Error
	}
	if groupRecord.RowsAffected != 1 {
		return errors.New("Description not changed in database.")
	}

	return nil

}

// Verify if a user ID is an owner of a group
func VerifyUserOwnershipToGroup(UserID int, GroupID int) (bool, error) {

	var group models.Group

	grouprecord := Instance.Where("`groups`.enabled = ?", 1).Where("`groups`.id = ?", GroupID).Where("`groups`.owner = ?", UserID).Find(&group)
	if grouprecord.Error != nil {
		return false, grouprecord.Error
	} else if grouprecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}
