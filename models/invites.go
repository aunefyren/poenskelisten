package models

import (
	"gorm.io/gorm"
)

type Invite struct {
	gorm.Model
	InviteCode      string `json:"invite_code" gorm:"unique;not null"`
	InviteUsed      *bool  `json:"invite_used" gorm:"not null;default: false"`
	InviteRecipient int    `json:"invite_recipient" gorm:"default: null"`
	InviteEnabled   *bool  `json:"invite_enabled" gorm:"not null;default: true"`
}
