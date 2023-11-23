package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GormModel struct {
	ID        uuid.UUID      `json:"id" gorm:"primaryKey; unique; type:varchar(100); NOT NULL;"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}
