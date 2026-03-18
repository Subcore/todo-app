package model

import (
	"time"
)

type Todo struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Title     string    `gorm:"size:255;not null" json:"title"`
	Completed bool      `gorm:"default:false" json:"completed"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
