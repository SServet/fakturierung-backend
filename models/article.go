package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Article struct {
	Id          string  `json:"id" gorm:"primaryKey"`
	Name        string  `json:"name" gorm:"not null;index"` // index helps search/sort
	Description string  `json:"description"`
	UnitPrice   float64 `json:"unit_price"`
	Active      bool    `json:"active" gorm:"index"` // fixed (was json:"-")
}

func (article *Article) BeforeCreate(tx *gorm.DB) (err error) {
	article.Id = uuid.NewString()
	return
}
