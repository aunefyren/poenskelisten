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
		Where("enabled = ?", true).
		Find(&inviteStruct)

	if inviteRecords.Error != nil {
		return []models.Invite{}, inviteRecords.Error
	}
	if inviteRecords.RowsAffected == 0 {
		return []models.Invite{}, nil
	}
	return inviteStruct, nil
}

// ErrInviteNotFound means the lookup ran but matched no enabled invite, as
// opposed to the query itself failing.
var ErrInviteNotFound = errors.New("Invite not found.")

// Get invite using ID
func GetInviteByID(inviteID uuid.UUID) (models.Invite, error) {
	var inviteStruct models.Invite

	inviteRecords := Instance.
		Where("id = ? AND enabled = ?", inviteID, true).
		Find(&inviteStruct)

	if inviteRecords.Error != nil {
		return models.Invite{}, inviteRecords.Error
	}
	if inviteRecords.RowsAffected != 1 {
		return models.Invite{}, ErrInviteNotFound
	}
	return inviteStruct, nil
}

// Set invite to disabled by ID
func DeleteInviteByID(inviteID uuid.UUID) error {
	var inviteStruct models.Invite

	inviteRecords := Instance.
		Model(inviteStruct).
		Where("id = ?", inviteID).
		Update("enabled", false)

	if inviteRecords.Error != nil {
		return inviteRecords.Error
	}
	if inviteRecords.RowsAffected != 1 {
		return errors.New("Code not changed in database.")
	}

	return nil
}
