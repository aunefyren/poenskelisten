package database

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/models"
	"errors"
	"time"

	"github.com/google/uuid"
)

// SeedFirstPartyClient ensures the built-in web app OAuth client exists and keeps
// its redirect URI / scopes / name in sync with the current config. Idempotent;
// run on every migration.
func SeedFirstPartyClient() error {
	redirectURI := config.OAuthIssuer() + "/oauth/callback"
	scopes := []string{"openid", "profile", "email"}
	enabled := true

	var existing models.OAuthClient
	record := Instance.
		Where(&models.OAuthClient{ClientID: models.FirstPartyClientID}).
		Find(&existing)
	if record.Error != nil {
		return record.Error
	}

	if record.RowsAffected == 0 {
		client := models.OAuthClient{
			ClientID:                models.FirstPartyClientID,
			ClientName:              config.ConfigFile.PoenskelistenName,
			RedirectURIs:            []string{redirectURI},
			Scopes:                  scopes,
			GrantTypes:              []string{"authorization_code", "refresh_token"},
			TokenEndpointAuthMethod: models.TokenEndpointAuthNone,
			IsPublic:                true,
			IsFirstParty:            true,
			Registered:              false,
			Enabled:                 &enabled,
		}
		client.ID = uuid.New()
		return Instance.Create(&client).Error
	}

	existing.ClientName = config.ConfigFile.PoenskelistenName
	existing.RedirectURIs = []string{redirectURI}
	existing.Scopes = scopes
	existing.IsPublic = true
	existing.IsFirstParty = true
	existing.Enabled = &enabled
	return Instance.Save(&existing).Error
}

// CreateOAuthClient persists a new OAuth client (e.g. from dynamic registration).
func CreateOAuthClient(client models.OAuthClient) (models.OAuthClient, error) {
	client.ID = uuid.New()
	record := Instance.Create(&client)
	if record.Error != nil {
		return models.OAuthClient{}, record.Error
	}
	return client, nil
}

// GetAllOAuthClients returns every registered client, with the secret hash
// stripped, for admin listing.
func GetAllOAuthClients() ([]models.OAuthClient, error) {
	clients := []models.OAuthClient{}
	record := Instance.Find(&clients)
	if record.Error != nil {
		return nil, record.Error
	}
	for i := range clients {
		clients[i].ClientSecretHash = nil
	}
	return clients, nil
}

// DisableOAuthClient disables a registered client (soft revoke). It refuses to
// touch the built-in first-party client.
func DisableOAuthClient(clientID string) error {
	client, found, err := GetOAuthClient(clientID)
	if err != nil {
		return err
	}
	if !found {
		return errors.New("client not found")
	}
	if client.IsFirstParty {
		return errors.New("the built-in client cannot be revoked")
	}

	disabled := false
	record := Instance.
		Model(&models.OAuthClient{}).
		Where(&models.OAuthClient{ClientID: clientID}).
		Update("enabled", &disabled)
	if record.Error != nil {
		return record.Error
	}
	if record.RowsAffected != 1 {
		return errors.New("client not disabled in database")
	}
	return nil
}

// GetOAuthClient looks up a client by its client_id.
func GetOAuthClient(clientID string) (models.OAuthClient, bool, error) {
	var client models.OAuthClient
	record := Instance.
		Where(&models.OAuthClient{ClientID: clientID}).
		Find(&client)
	if record.Error != nil {
		return models.OAuthClient{}, false, record.Error
	}
	if record.RowsAffected == 0 {
		return models.OAuthClient{}, false, nil
	}
	if record.RowsAffected != 1 {
		return models.OAuthClient{}, false, errors.New("multiple clients share the same client_id")
	}
	return client, true, nil
}

// CreateAuthorizationCode persists a new authorization code (caller supplies the
// already-hashed code and all bindings).
func CreateAuthorizationCode(code models.AuthorizationCode) (models.AuthorizationCode, error) {
	code.ID = uuid.New()
	record := Instance.Create(&code)
	if record.Error != nil {
		return models.AuthorizationCode{}, record.Error
	}
	return code, nil
}

// ConsumeAuthorizationCode atomically marks an unused, unexpired code as used and
// returns it. It errors if the code is unknown, expired, or already used — which,
// for an already-used code, is how authorization-code replay is caught.
func ConsumeAuthorizationCode(codeHash string) (models.AuthorizationCode, error) {
	now := time.Now()

	record := Instance.
		Model(&models.AuthorizationCode{}).
		Where("code_hash = ? AND used_at IS NULL AND expires_at > ?", codeHash, now).
		Update("used_at", now)
	if record.Error != nil {
		return models.AuthorizationCode{}, record.Error
	}
	if record.RowsAffected != 1 {
		return models.AuthorizationCode{}, errors.New("authorization code invalid, expired, or already used")
	}

	var code models.AuthorizationCode
	lookup := Instance.Where("code_hash = ?", codeHash).Find(&code)
	if lookup.Error != nil {
		return models.AuthorizationCode{}, lookup.Error
	}
	if lookup.RowsAffected != 1 {
		return models.AuthorizationCode{}, errors.New("authorization code not found")
	}
	return code, nil
}

// GetConsent returns a user's stored consent for a client, if any.
func GetConsent(userID uuid.UUID, clientID string) (models.OAuthConsent, bool, error) {
	var consent models.OAuthConsent
	record := Instance.
		Where(&models.OAuthConsent{UserID: userID, ClientID: clientID}).
		Find(&consent)
	if record.Error != nil {
		return models.OAuthConsent{}, false, record.Error
	}
	if record.RowsAffected == 0 {
		return models.OAuthConsent{}, false, nil
	}
	return consent, true, nil
}

// GetUserConsents returns the clients a user has granted access to (their
// "connected apps").
func GetUserConsents(userID uuid.UUID) ([]models.OAuthConsent, error) {
	consents := []models.OAuthConsent{}
	record := Instance.
		Where(&models.OAuthConsent{UserID: userID}).
		Find(&consents)
	if record.Error != nil {
		return nil, record.Error
	}
	return consents, nil
}

// UpsertConsent stores (or replaces) the scopes a user has granted a client.
func UpsertConsent(userID uuid.UUID, clientID string, scopes []string) error {
	consent, found, err := GetConsent(userID, clientID)
	if err != nil {
		return err
	}
	if !found {
		consent = models.OAuthConsent{UserID: userID, ClientID: clientID, Scopes: scopes}
		consent.ID = uuid.New()
		return Instance.Create(&consent).Error
	}
	consent.Scopes = scopes
	return Instance.Save(&consent).Error
}

// RevokeConsent removes a user's consent for a client.
func RevokeConsent(userID uuid.UUID, clientID string) error {
	return Instance.
		Where(&models.OAuthConsent{UserID: userID, ClientID: clientID}).
		Delete(&models.OAuthConsent{}).Error
}

// SetUserSessionsInvalidatedAt stamps the global logout marker so SSO and access
// tokens issued before now are rejected.
func SetUserSessionsInvalidatedAt(userID uuid.UUID, at time.Time) error {
	record := Instance.
		Model(&models.User{}).
		Where(&models.GormModel{ID: userID}).
		Update("sessions_invalidated_at", at)
	return record.Error
}
