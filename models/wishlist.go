package models

import (
	"time"

	"github.com/google/uuid"
)

type Wishlist struct {
	GormModel
	Name        string    `json:"name" gorm:"not null"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled" gorm:"not null; default: true"`
	OwnerID     uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Owner       User      `json:"owner" gorm:"not null;"`
	Date        time.Time `json:"date" gorm:"not null"`
	Expires     *bool     `json:"expires" gorm:"not null; default: true"`
	Claimable   *bool     `json:"claimable" gorm:"not null; default: false"`
}

type WishlistCreationRequest struct {
	GormModel
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Date        string     `json:"date"`
	Expires     *bool      `json:"expires"`
	Group       *uuid.UUID `json:"group"`
	Claimable   *bool      `json:"claimable"`
}

type WishlistUpdateRequest struct {
	GormModel
	Name        string `json:"name"`
	Description string `json:"description"`
	Date        string `json:"date"`
	Expires     *bool  `json:"expires"`
	Claimable   *bool  `json:"claimable"`
}

type WishlistUser struct {
	GormModel
	Name          string                       `json:"name"`
	Description   string                       `json:"description"`
	Enabled       bool                         `json:"enabled"`
	Owner         User                         `json:"owner"`
	Date          time.Time                    `json:"date"`
	Expires       *bool                        `json:"expires"`
	Claimable     *bool                        `json:"claimable"`
	Members       []GroupUser                  `json:"members"`
	Wishes        []WishObject                 `json:"wishes"`
	Collaborators []WishlistCollaboratorObject `json:"collaborators"`
}

type WishlistMembership struct {
	GormModel
	GroupID    uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Group      Group     `json:"group" gorm:"not null;"`
	Enabled    bool      `json:"enabled" gorm:"not null;default: true"`
	WishlistID uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Wishlist   Wishlist  `json:"wishlist" gorm:"not null;"`
}

type WishlistMembershipObject struct {
	GormModel
	Group    GroupUser    `json:"group"`
	Enabled  bool         `json:"enabled"`
	Wishlist WishlistUser `json:"wishlist"`
}

type WishlistMembershipCreationRequest struct {
	Groups []uuid.UUID `json:"groups"`
}

type WishlistCollaborator struct {
	GormModel
	UserID     uuid.UUID `json:"" gorm:"type:varchar(100);"`
	User       User      `json:"user" gorm:"not null;"`
	Enabled    bool      `json:"enabled" gorm:"not null;default: true"`
	WishlistID uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Wishlist   Wishlist  `json:"wishlist" gorm:"not null;"`
}

type WishlistCollaboratorObject struct {
	GormModel
	User       User      `json:"user"`
	Enabled    bool      `json:"enabled"`
	WishlistID uuid.UUID `json:"wishlist"`
}

type WishlistCollaboratorCreationRequest struct {
	Users []uuid.UUID `json:"users"`
}

type WishlistMembershipDeletionRequest struct {
	Group uuid.UUID `json:"group_id"`
}

type WishlistCollaboratorDeletionRequest struct {
	User uuid.UUID `json:"user_id"`
}
