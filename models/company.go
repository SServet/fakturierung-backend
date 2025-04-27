package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Company struct {
	Id            string        `json:"id" gorm:"primaryKey"`
	CompanyName   string        `json:"company_name" gorm:"not null;unique"`
	Address       string        `json:"address" gorm:"not null"`
	City          string        `json:"city" gorm:"not null"`
	Country       string        `json:"country" gorm:"not null"`
	Zip           string        `json:"zip" gorm:"not null"`
	Homepage      string        `json:"homepage" gorm:"null"`
	UID           string        `json:"uid" gorm:"null"`
	UserId        string        `json:"-"`
	User          User          `json:"user" gorm:"foreignKey:UserId;references:Id"`
	PId           uint          `json:"-"`
	ContactPerson ContactPerson `json:"contact_person" gorm:"foreignKey:PId;references:Id"`
	SchemaName    string        `json:"-"`
}

func (company *Company) BeforeCreate(tx *gorm.DB) (err error) {
	// UUID version 4
	company.Id = uuid.NewString()
	return
}
