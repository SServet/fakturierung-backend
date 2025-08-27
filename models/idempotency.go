package models

import "time"

// IdempotencyKey stores the first successful response for a given request hash.
// It is tenant-scoped (lives in the tenant schema).
type IdempotencyKey struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	Key            string     `json:"key" gorm:"size:128;uniqueIndex"` // header value
	RequestHash    string     `json:"request_hash" gorm:"size:64"`     // sha256 of method|path|body|schema|user
	Method         string     `json:"method" gorm:"size:10"`
	Path           string     `json:"path" gorm:"size:255"`
	TenantSchema   string     `json:"tenant_schema" gorm:"size:64"`
	UserID         string     `json:"user_id" gorm:"size:128"`
	ResponseStatus int        `json:"response_status"`     // 0 => not completed yet
	ResponseBody   []byte     `json:"-" gorm:"type:bytea"` // raw response body (JSON)
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at"`
}
