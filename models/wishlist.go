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
}

type WishlistCreationRequest struct {
	gorm.Model
	Name        string `json:"name"`
	Description string `json:"description"`
	Date        string `json:"date"`
}

type WishlistUser struct {
	gorm.Model
	Name        string      `json:"name" gorm:"not null"`
	Description string      `json:"description"`
	Enabled     bool        `json:"enabled" gorm:"not null; default: true"`
	Owner       User        `json:"owner"`
	Date        time.Time   `json:"date" gorm:"not null"`
	Members     []GroupUser `json:"members"`
	Wishes      []WishUser  `json:"wishes"`
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
