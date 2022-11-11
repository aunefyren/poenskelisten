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
	Group       int       `json:"group_id" gorm:"not null"`
	Date        time.Time `json:"date" gorm:"not null"`
}

type WishlistUser struct {
	gorm.Model
	Name        string    `json:"name" gorm:"not null"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled" gorm:"not null; default: true"`
	Owner       User      `json:"owner"`
	Group       int       `json:"group_id" gorm:"not null"`
	Date        time.Time `json:"date" gorm:"not null"`
	Members     []User    `json:"members"`
}
