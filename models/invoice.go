package models

import (
	"time"

	"gorm.io/datatypes"
)

// Invoice is the current/live state of a commercial document.
type Invoice struct {
	ID            uint     `json:"id" gorm:"primaryKey"`
	InvoiceNumber string   `json:"invoice_number" gorm:"unique"`
	CId           uint     `json:"-"`
	Customer      Customer `json:"customer" gorm:"foreignKey:CId;references:Id"`

	// Live items (latest state)
	Items    []InvoiceItem `json:"articles" gorm:"foreignKey:InvoiceID;constraint:OnDelete:CASCADE"`
	Subtotal float64       `json:"subtotal" gorm:"type:numeric(12,2)"`
	TaxTotal float64       `json:"tax_total" gorm:"type:numeric(12,2)"`
	Total    float64       `json:"total" gorm:"type:numeric(12,2)"`

	// State
	Draft       bool       `json:"draft"`
	Published   bool       `json:"published"`
	PublishedAt *time.Time `json:"published_at"`

	// Payments rollup
	PaidTotal float64 `json:"paid_total" gorm:"type:numeric(12,2)"`

	CreatedAt time.Time `json:"created_at"`
}

type InvoiceItem struct {
	ID          uint    `json:"id" gorm:"primaryKey"`
	InvoiceID   uint    `json:"-" gorm:"index"`                   // fast join
	ArticleID   string  `json:"article_id" gorm:"not null;index"` // FK to articles.id (see Article & migrator)
	Article     Article `json:"-" gorm:"foreignKey:ArticleID;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
	Description string  `json:"description"`
	Amount      int     `json:"amount"`
	UnitPrice   float64 `json:"unit_price" gorm:"type:numeric(12,2)"`
	TaxRate     float64 `json:"tax_rate"` // rate stays float
	NetPrice    float64 `json:"net_price" gorm:"type:numeric(12,2)"`
	TaxAmount   float64 `json:"tax_amount" gorm:"type:numeric(12,2)"`
	GrossPrice  float64 `json:"gross_price" gorm:"type:numeric(12,2)"`
}

// Immutable snapshot
type InvoiceVersion struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	InvoiceID uint           `json:"invoice_id" gorm:"index:idx_invoice_versions_invoice_id_version_no,unique,priority:1"`
	VersionNo int            `json:"version_no" gorm:"not null;index:idx_invoice_versions_invoice_id_version_no,unique,priority:2"`
	Kind      string         `json:"kind" gorm:"type:VARCHAR(20)"` // "quotation" | "invoice"
	Snapshot  datatypes.JSON `json:"snapshot" gorm:"type:jsonb"`
	CreatedAt time.Time      `json:"created_at"`
}

// Payment survives conversions; linked to invoice.
type Payment struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	InvoiceID uint      `json:"invoice_id" gorm:"index:idx_payments_invoice_paid_at,priority:1"`
	Amount    float64   `json:"amount" gorm:"type:numeric(12,2)"`
	Method    string    `json:"method"`
	Reference string    `json:"reference"`
	Note      string    `json:"note"`
	PaidAt    time.Time `json:"paid_at" gorm:"index:idx_payments_invoice_paid_at,priority:2"`
	CreatedAt time.Time `json:"created_at"`
}
