package models

import (
	"time"

	"gorm.io/gorm"
)

type Wishlist struct {
	gorm.Model
	Name        string    `json:"name" gorm:"not null"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled" gorm:"not null; default: true"`
	Owner       int       `json:"owner_id" gorm:"not null"`
	Date        time.Time `json:"date" gorm:"not null"`
	Claimable   bool      `json:"claimable" gorm:"not null; default: false"`
}

type WishlistCreationRequest struct {
	gorm.Model
	Name        string `json:"name"`
	Description string `json:"description"`
	Date        string `json:"date"`
	Group       int    `json:"group"`
	Claimable   bool   `json:"claimable"`
}

type WishlistUpdateRequest struct {
	gorm.Model
	Name        string `json:"name"`
	Description string `json:"description"`
	Date        string `json:"date"`
	Claimable   bool   `json:"claimable"`
}

type WishlistUser struct {
	gorm.Model
	Name        string       `json:"name" gorm:"not null"`
	Description string       `json:"description"`
	Enabled     bool         `json:"enabled" gorm:"not null; default: true"`
	Owner       User         `json:"owner"`
	Date        time.Time    `json:"date" gorm:"not null"`
	Claimable   bool         `json:"claimable" gorm:"not null; default: false"`
	Members     []GroupUser  `json:"members"`
	Wishes      []WishObject `json:"wishes"`
}

type WishlistMembership struct {
	gorm.Model
	Group    int  `json:"group_id" gorm:"not null"`
	Enabled  bool `json:"enabled" gorm:"not null;default: true"`
	Wishlist int  `json:"wishlist_id" gorm:"not null"`
}

type WishlistMembershipObject struct {
	gorm.Model
	Group    GroupUser    `json:"group"`
	Enabled  bool         `json:"enabled"`
	Wishlist WishlistUser `json:"wishlist"`
}

type WishlistMembershipCreationRequest struct {
	Groups []int `json:"groups"`
}
