package models

import (
	"gorm.io/gorm"
)

type Wish struct {
	gorm.Model
	Name       string  `json:"name" gorm:"not null"`
	Note       string  `json:"note"`
	Price      float64 `json:"price"`
	Enabled    bool    `json:"enabled" gorm:"not null; default: true"`
	Owner      int     `json:"owner_id" gorm:"not null"`
	URL        string  `json:"url"`
	WishlistID int     `json:"wishlist_id" gorm:"not null"`
}

type WishCreationRequest struct {
	Name  string  `json:"name"`
	Note  string  `json:"note"`
	Price float64 `json:"price"`
	URL   string  `json:"url"`
}

type WishUpdateRequest struct {
	Name  string  `json:"name"`
	Note  string  `json:"note"`
	Price float64 `json:"price"`
	URL   string  `json:"url"`
}

type WishObject struct {
	gorm.Model
	Name       string            `json:"name"`
	Note       string            `json:"note"`
	Price      float64           `json:"price"`
	Enabled    bool              `json:"enabled"`
	Owner      User              `json:"owner_id"`
	URL        string            `json:"url"`
	WishlistID int               `json:"wishlist_id"`
	WishClaim  []WishClaimObject `json:"wishclaim"`
}

type WishClaim struct {
	gorm.Model
	Wish    int  `json:"wish_id" gorm:"not null"`
	User    int  `json:"user_id" gorm:"not null"`
	Enabled bool `json:"enabled" gorm:"not null;default: true"`
}

type WishClaimObject struct {
	gorm.Model
	Wish    int  `json:"wish_id"`
	User    User `json:"user"`
	Enabled bool `json:"enabled"`
}

type WishClaimCreationRequest struct {
	WishlistID int `json:"wishlist_id"`
}
