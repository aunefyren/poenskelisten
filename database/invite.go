package database

import (
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"errors"

	"github.com/google/uuid"
)

// Get invites in database that have not been disabled
func GetAllEnabledInvites() ([]models.Invite, error) {
	var inviteStruct []models.Invite

	inviteRecords := Instance.
		Where(&models.Invite{Enabled: &utilities.DBTrue}).
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

	inviteRecords := Instance.
		Where(&models.Invite{Enabled: &utilities.DBTrue}).
		Where(&models.GormModel{ID: inviteID}).
		Find(&inviteStruct)

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

	inviteRecords := Instance.
		Model(inviteStruct).
		Where(&models.GormModel{ID: inviteID}).
		Update("enabled", false)

	if inviteRecords.Error != nil {
		return inviteRecords.Error
	}
	if inviteRecords.RowsAffected != 1 {
		return errors.New("Code not changed in database.")
	}

	return nil
}
