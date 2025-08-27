package models

import (
	"time"

	"gorm.io/datatypes"
)

// Invoice is the current/live state of a commercial document.
// Draft=true means it's a Quotation; Draft=false means it's an Invoice.
// Published indicates a legally issued invoice (you can still convert; payments remain linked).
type Invoice struct {
	ID            uint     `json:"id" gorm:"primaryKey"`
	InvoiceNumber string   `json:"invoice_number" gorm:"unique"`
	CId           uint     `json:"-"`
	Customer      Customer `json:"customer" gorm:"foreignKey:CId;references:Id"`

	// Live items (latest state)
	Items    []InvoiceItem `json:"articles" gorm:"foreignKey:InvoiceID;constraint:OnDelete:CASCADE"`
	Subtotal float64       `json:"subtotal"`
	TaxTotal float64       `json:"tax_total"`
	Total    float64       `json:"total"`

	// State
	Draft       bool       `json:"draft"`     // true => Quotation, false => Invoice
	Published   bool       `json:"published"` // true => legally issued
	PublishedAt *time.Time `json:"published_at"`

	// Payments summary (for quick reads; maintained by controller)
	PaidTotal float64 `json:"paid_total"`

	CreatedAt time.Time `json:"created_at"`
}

// InvoiceItem belongs to the live Invoice (latest snapshot).
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

// InvoiceVersion is an immutable snapshot of an invoice at a point in time.
// We store a JSONB snapshot for simplicity and minimal schema churn.
type InvoiceVersion struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	InvoiceID uint           `json:"invoice_id" gorm:"index"`
	VersionNo int            `json:"version_no" gorm:"not null"`
	Kind      string         `json:"kind" gorm:"type:VARCHAR(20)"` // "quotation" | "invoice"
	Snapshot  datatypes.JSON `json:"snapshot" gorm:"type:jsonb"`
	CreatedAt time.Time      `json:"created_at"`
}

// Payment records money received against an Invoice (survives conversions).
type Payment struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	InvoiceID uint      `json:"invoice_id" gorm:"index"`
	Amount    float64   `json:"amount"`
	Method    string    `json:"method"`    // e.g., "bank-transfer", "card", "cash"
	Reference string    `json:"reference"` // bank ref, transaction id, etc.
	Note      string    `json:"note"`
	PaidAt    time.Time `json:"paid_at"`
	CreatedAt time.Time `json:"created_at"`
}
