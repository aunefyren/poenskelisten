package models

import (
	"time"
)

type News struct {
	GormModel
	Title      string     `json:"title" gorm:"not null"`
	Body       string     `json:"body"`
	Enabled    bool       `json:"enabled" gorm:"not null; default: true"`
	Date       time.Time  `json:"date" gorm:"not null"`
	ExpiryDate *time.Time `json:"expiry_date" gorm:""`
}

type NewsCreationRequest struct {
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	Date       time.Time  `json:"date"`
	ExpiryDate *time.Time `json:"expiry_date"`
}

type NewsUpdateRequest struct {
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	Date       time.Time  `json:"date"`
	ExpiryDate *time.Time `json:"expiry_date"`
}
