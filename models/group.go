package models

import "github.com/google/uuid"

type Group struct {
	GormModel
	Name        string    `json:"name" gorm:"not null"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled" gorm:"not null;default: true"`
	OwnerID     uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Owner       User      `json:"owner" gorm:"not null;"`
}

type GroupCreationRequest struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Members     []string    `json:"members"`
	Wishlists   []uuid.UUID `json:"wishlists"`
}

type GroupUpdateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type GroupUser struct {
	GormModel
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled" `
	Owner       User   `json:"owner"`
	Members     []User `json:"members"`
}

type GroupMembership struct {
	GormModel
	GroupID  uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Group    Group     `json:"group" gorm:"not null;"`
	Enabled  bool      `json:"enabled" gorm:"not null;default: true"`
	MemberID uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Member   User      `json:"member" gorm:"not null;"`
}

type GroupMembershipUser struct {
	GormModel
	Group   uuid.UUID `json:"group_id"`
	Enabled bool      `json:"enabled"`
	Members User      `json:"members"`
}

type GroupMembershipCreationRequest struct {
	Members []string `json:"members"`
}

type GroupMembershipRemovalRequest struct {
	MemberID uuid.UUID `json:"member_id"`
}

type GroupAddWishlistsRequest struct {
	Wishlists []uuid.UUID `json:"wishlists"`
}
