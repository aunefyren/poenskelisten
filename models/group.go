package models

import "gorm.io/gorm"

type Group struct {
	gorm.Model
	Name        string `json:"name" gorm:"not null"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled" gorm:"not null;default: true"`
	Owner       int    `json:"owner_id" gorm:"not null"`
}

type GroupMembership struct {
	gorm.Model
	Group   int  `json:"group_id" gorm:"not null"`
	Enabled bool `json:"enabled" gorm:"not null;default: true"`
	Member  int  `json:"member_id" gorm:"not null"`
}
