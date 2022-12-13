package models

import (
	"time"

	"gorm.io/gorm"
)

type News struct {
	gorm.Model
	Title   string    `json:"title" gorm:"not null"`
	Body    string    `json:"body"`
	Enabled bool      `json:"enabled" gorm:"not null; default: true"`
	Date    time.Time `json:"date" gorm:"not null"`
}

type NewsCreationRequest struct {
	Title string    `json:"title" gorm:"not null"`
	Body  string    `json:"body"`
	Date  time.Time `json:"date" gorm:"not null"`
}
