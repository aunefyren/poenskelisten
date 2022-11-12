package models

import (
	"gorm.io/gorm"
)

type Wish struct {
	gorm.Model
	Name       string `json:"name" gorm:"not null"`
	Note       string `json:"note"`
	Enabled    bool   `json:"enabled" gorm:"not null; default: true"`
	Owner      int    `json:"owner_id" gorm:"not null"`
	URL        string `json:"url"`
	WishlistID int    `json:"wishlist_id" gorm:"not null"`
}

type WishCreationRequest struct {
	Name string `json:"name"`
	Note string `json:"note"`
	URL  string `json:"url"`
}

type WishUser struct {
	gorm.Model
	Name       string `json:"name"`
	Note       string `json:"note"`
	Enabled    bool   `json:"enabled"`
	Owner      User   `json:"owner_id"`
	URL        string `json:"url"`
	WishlistID int    `json:"wishlist_id"`
}
