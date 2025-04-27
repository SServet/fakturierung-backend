package models

type ContactPerson struct {
	Id           uint   `json:"id" gorm:"primaryKey"`
	FirstName    string `json:"first_name" gorm:"not null"`
	LastName     string `json:"last_name" gorm:"not null"`
	PhoneNumber  string `json:"phone_number" gorm:"not null"`
	MobileNumber string `json:"mobile_number" gorm:"not null"`
	Salutation   string `json:"saluatation" gorm:"not null"`
	Title        string `json:"title" gorm:"not null"`
}
