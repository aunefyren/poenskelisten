package database

import (
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"errors"

	"github.com/google/uuid"
)

// Get invites in database that have not been disabled
func GetAllEnabledInvites() ([]models.Invite, error) {
	var invitestruct []models.Invite
	var LolTrue = true
	inviterecords := Instance.Where(&models.Invite{Enabled: &LolTrue}).Find(&invitestruct)
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
	inviterecords := Instance.Where(&models.Invite{Enabled: &utilities.DBTrue}).Where(&models.GormModel{ID: inviteID}).Find(&invitestruct)
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
	inviterecords := Instance.Model(invitestruct).Where(&models.GormModel{ID: inviteID}).Update("enabled", false)
	if inviterecords.Error != nil {
		return inviterecords.Error
	}
	if inviterecords.RowsAffected != 1 {
		return errors.New("Code not changed in database.")
	}

	return nil
}
