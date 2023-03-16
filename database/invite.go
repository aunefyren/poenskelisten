package database

import (
	"aunefyren/poenskelisten/models"
	"errors"
)

// Get invites in database that have not been disabled
func GetAllEnabledInvites() ([]models.Invite, error) {
	var invitestruct []models.Invite
	inviterecords := Instance.Where("`invites`.invite_enabled = ?", 1).Find(&invitestruct)
	if inviterecords.Error != nil {
		return []models.Invite{}, inviterecords.Error
	}
	if inviterecords.RowsAffected == 0 {
		return []models.Invite{}, nil
	}
	return invitestruct, nil
}

// Get invite using ID
func GetInviteByID(inviteID int) (models.Invite, error) {
	var invitestruct models.Invite
	inviterecords := Instance.Where("`invites`.invite_enabled = ?", 1).Where("`invites`.ID = ?", inviteID).Find(&invitestruct)
	if inviterecords.Error != nil {
		return models.Invite{}, inviterecords.Error
	}
	if inviterecords.RowsAffected != 1 {
		return models.Invite{}, errors.New("Invite not found.")
	}
	return invitestruct, nil
}

// Set invite to disabled by ID
func DeleteInviteByID(inviteID int) error {
	var invitestruct models.Invite
	inviterecords := Instance.Model(invitestruct).Where("`invites`.ID= ?", inviteID).Update("invite_enabled", 0)
	if inviterecords.Error != nil {
		return inviterecords.Error
	}
	if inviterecords.RowsAffected != 1 {
		return errors.New("Code not changed in database.")
	}

	return nil
}
