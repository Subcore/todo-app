package model

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// Todo — модель задачи, хранящаяся в таблице todos
type Todo struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`          // Уникальный идентификатор
	Title     string         `gorm:"size:255;not null" json:"title"`              // Заголовок задачи (макс. 255 символов)
	Completed bool           `gorm:"default:false" json:"completed"`              // Статус выполнения
	DueDate   *time.Time     `json:"due_date,omitempty" swaggertype:"string" format:"date-time" example:"2023-12-31T23:59:59Z"` // Срок выполнения
	Tags      pq.StringArray `gorm:"type:text[]" json:"tags" swaggertype:"array,string"`                                        // Теги задачи
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at" swaggertype:"string" format:"date-time"`                   // Время создания
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at" swaggertype:"string" format:"date-time"`                   // Время последнего обновления
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty" swaggertype:"string" format:"date-time"`                  // Время мягкого удаления
}

// TodoFilter — структура для удобной передачи фильтров в репозиторий
type TodoFilter struct {
	Completed *bool
	DueBefore *time.Time
	DueAfter  *time.Time
	Search    string
}
