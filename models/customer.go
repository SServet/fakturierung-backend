package models

type Customer struct {
	Id           uint   `json:"id" gorm:"primaryKey"`
	CompanyName  string `json:"company_name" gorm:"not null;unique"`
	Address      string `json:"address" gorm:"not null"`
	City         string `json:"city" gorm:"not null"`
	Country      string `json:"country" gorm:"not null"`
	Zip          string `json:"zip" gorm:"not null"`
	Homepage     string `json:"homepage" gorm:"null"`
	UID          string `json:"uid" gorm:"null"`
	Email        string `json:"email" gorm:"unique;not null"`
	FirstName    string `json:"first_name" gorm:"not null"`
	LastName     string `json:"last_name" gorm:"not null"`
	PhoneNumber  string `json:"phone_number" gorm:"not null"`
	MobileNumber string `json:"mobile_number" gorm:"not null"`
	Salutation   string `json:"saluatation" gorm:"not null"`
	Title        string `json:"title" gorm:"not null"`
	Active       bool   `json:"-"`
}
