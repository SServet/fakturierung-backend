package models

import (
	"time"
)

type Invoice struct {
	ID            uint          `json:"id" gorm:"primaryKey"`
	InvoiceNumber string        `json:"invoice_number" gorm:"unique"`
	CId           uint          `json:"-"`
	Customer      Customer      `json:"customer" gorm:"foreignKey:CId;references:Id"`
	Items         []InvoiceItem `json:"articles" gorm:"foreignKey:InvoiceID;constraint:OnDelete:CASCADE"`
	Subtotal      float64       `json:"subtotal"`
	TaxTotal      float64       `json:"tax_total"`
	Total         float64       `json:"total"`
	Draft         bool          `json:"draft"`
	Published     bool          `json:"published"`
	CreatedAt     time.Time     `json:"created_at"`
}

type InvoiceItem struct {
	ID          uint    `json:"id" gorm:"primaryKey"`
	InvoiceID   uint    `json:"-"`
	ArticleID   string  `json:"article_id"`
	Description string  `json:"description"`
	Amount      int     `json:"amount"`
	UnitPrice   float64 `json:"unit_price"`
	TaxRate     float64 `json:"tax_rate"`
	NetPrice    float64 `json:"net_price"`
	TaxAmount   float64 `json:"tax_amount"`
	GrossPrice  float64 `json:"gross_price"`
}
