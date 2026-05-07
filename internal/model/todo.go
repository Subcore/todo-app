package model

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// Todo is the task record stored in the todos table.
type Todo struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Title     string         `gorm:"size:255;not null" json:"title"`
	Completed bool           `gorm:"default:false" json:"completed"`
	DueDate   *time.Time     `json:"due_date,omitempty" swaggertype:"string" format:"date-time" example:"2023-12-31T23:59:59Z"`
	Tags      pq.StringArray `gorm:"type:text[]" json:"tags" swaggertype:"array,string"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at" swaggertype:"string" format:"date-time"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at" swaggertype:"string" format:"date-time"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty" swaggertype:"string" format:"date-time"`
}

// TodoFilter groups the optional filters applied when listing todos.
type TodoFilter struct {
	Completed *bool
	DueBefore *time.Time
	DueAfter  *time.Time
	Search    string
}
