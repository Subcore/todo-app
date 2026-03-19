package model

import (
	"time"
	"gorm.io/gorm"
)

type Todo struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Title     string    `gorm:"size:255;not null" json:"title"`
	Completed bool      `gorm:"default:false" json:"completed"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at" swaggertype:"string" format:"date-time" example:"2023-10-27T10:00:00Z"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at" swaggertype:"string" format:"date-time" example:"2023-10-27T10:00:00Z"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty" swaggertype:"string" format:"date-time" example:"2023-10-27T10:00:00Z"`
}
