package models

import "gorm.io/gorm"

type Group struct {
	gorm.Model
	Name        string `json:"name" gorm:"not null"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled" gorm:"not null;default: true"`
	Owner       int    `json:"owner_id" gorm:"not null"`
}

type GroupCreationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Members     []int  `json:"members"`
}

type GroupUpdateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type GroupUser struct {
	gorm.Model
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled" `
	Owner       User   `json:"owner"`
	Members     []User `json:"members"`
}

type GroupMembership struct {
	gorm.Model
	Group   int  `json:"group_id" gorm:"not null"`
	Enabled bool `json:"enabled" gorm:"not null;default: true"`
	Member  int  `json:"member_id" gorm:"not null"`
}

type GroupMembershipUser struct {
	gorm.Model
	Group   int  `json:"group_id"`
	Enabled bool `json:"enabled"`
	Members User `json:"members"`
}

type GroupMembershipCreationRequest struct {
	Members []int `json:"members" gorm:"not null"`
}
