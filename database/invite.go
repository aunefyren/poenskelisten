package database

import (
	"aunefyren/poenskelisten/models"
	"errors"

	"github.com/google/uuid"
)

// Get invites in database that have not been disabled
func GetAllEnabledInvites() ([]models.Invite, error) {
	var invitestruct []models.Invite
	inviterecords := Instance.Where("`invites`.enabled = ?", 1).Find(&invitestruct)
	if inviterecords.Error != nil {
		return []models.Invite{}, inviterecords.Error
	}
	if inviterecords.RowsAffected == 0 {
		return []models.Invite{}, nil
	}
	return invitestruct, nil
}

// Get invite using ID
func GetInviteByID(inviteID uuid.UUID) (models.Invite, error) {
	var invitestruct models.Invite
	inviterecords := Instance.Where("`invites`.enabled = ?", 1).Where("`invites`.id = ?", inviteID).Find(&invitestruct)
	if inviterecords.Error != nil {
		return models.Invite{}, inviterecords.Error
	}
	if inviterecords.RowsAffected != 1 {
		return models.Invite{}, errors.New("Invite not found.")
	}
	return invitestruct, nil
}

// Set invite to disabled by ID
func DeleteInviteByID(inviteID uuid.UUID) error {
	var invitestruct models.Invite
	inviterecords := Instance.Model(invitestruct).Where("`invites`.id = ?", inviteID).Update("enabled", 0)
	if inviterecords.Error != nil {
		return inviterecords.Error
	}
	if inviterecords.RowsAffected != 1 {
		return errors.New("Code not changed in database.")
	}

	return nil
}
