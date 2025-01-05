package database

import (
	"aunefyren/poenskelisten/models"
	"errors"

	"github.com/google/uuid"
)

// Get invites in database that have not been disabled
func GetAllEnabledInvites() ([]models.Invite, error) {
	var inviteStruct []models.Invite

	inviteRecords := Instance.
		Where("`invites`.enabled = ?", 1).
		Find(&inviteStruct)

	if inviteRecords.Error != nil {
		return []models.Invite{}, inviteRecords.Error
	}
	if inviteRecords.RowsAffected == 0 {
		return []models.Invite{}, nil
	}
	return inviteStruct, nil
}

// Get invite using ID
func GetInviteByID(inviteID uuid.UUID) (models.Invite, error) {
	var inviteStruct models.Invite
	inviteRecords := Instance.Where("`invites`.enabled = ?", 1).Where("`invites`.id = ?", inviteID).Find(&inviteStruct)

	if inviteRecords.Error != nil {
		return models.Invite{}, inviteRecords.Error
	}
	if inviteRecords.RowsAffected != 1 {
		return models.Invite{}, errors.New("Invite not found.")
	}
	return inviteStruct, nil
}

// Set invite to disabled by ID
func DeleteInviteByID(inviteID uuid.UUID) error {
	var inviteStruct models.Invite
	inviteRecords := Instance.Model(inviteStruct).Where("`invites`.id = ?", inviteID).Update("enabled", 0)
	if inviteRecords.Error != nil {
		return inviteRecords.Error
	}
	if inviteRecords.RowsAffected != 1 {
		return errors.New("Code not changed in database.")
	}

	return nil
}
