package models

import (
	"github.com/google/uuid"
)

type Invite struct {
	GormModel
	Code        string     `json:"invite_code" gorm:"unique;not null"`
	Used        bool       `json:"invite_used" gorm:"not null; default: false"`
	RecipientID *uuid.UUID `json:"" gorm:"type: varchar(100); default: null; index;"`
	Recipient   *User      `json:"invite_recipient" gorm:"not null;"`
	Enabled     *bool      `json:"invite_enabled" gorm:"not null; default: true"`
}

type InviteObject struct {
	GormModel
	InviteCode    string `json:"invite_code"`
	InviteUsed    bool   `json:"invite_used"`
	User          User   `json:"user"`
	InviteEnabled *bool  `json:"invite_enabled"`
}
