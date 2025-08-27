package database

import (
	"fmt"

	"fakturierung-backend/models"

	"gorm.io/gorm"
)

// MigrateTenantSchema applies (idempotent) schema migrations for a single tenant schema.
// It pins search_path to the tenant and performs:
// - AutoMigrate (tables/columns)
// - Money column types (NUMERIC(12,2))
// - Indexes (versions, payments, invoice_items)
// - Foreign key: invoice_items.article_id → articles.id
// - Basic CHECK constraints
// - Idempotency keys table + unique index
func MigrateTenantSchema(schema string) error {
	if schema == "" {
		return fmt.Errorf("schema name is empty")
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		// Pin the tenant schema for this transaction
		if err := tx.Exec(`SET search_path = "` + schema + `", public`).Error; err != nil {
			return fmt.Errorf("set search_path failed: %w", err)
		}

		// --- AutoMigrate tables/columns/index tags (non-destructive) ---
		if err := tx.AutoMigrate(
			&models.Article{},
			&models.Customer{},
			&models.Supplier{},
			&models.Invoice{},
			&models.InvoiceItem{},
			&models.InvoiceVersion{},
			&models.Payment{},
			&models.IdempotencyKey{}, // NEW
		); err != nil {
			return fmt.Errorf("tenant automigrate failed: %w", err)
		}

		// --- Enforce money columns as NUMERIC(12,2) (idempotent ALTERs) ---
		alters := []string{
			`ALTER TABLE articles       ALTER COLUMN unit_price TYPE numeric(12,2)`,
			`ALTER TABLE invoices       ALTER COLUMN subtotal   TYPE numeric(12,2)`,
			`ALTER TABLE invoices       ALTER COLUMN tax_total  TYPE numeric(12,2)`,
			`ALTER TABLE invoices       ALTER COLUMN total      TYPE numeric(12,2)`,
			`ALTER TABLE invoices       ALTER COLUMN paid_total TYPE numeric(12,2)`,
			`ALTER TABLE invoice_items  ALTER COLUMN unit_price TYPE numeric(12,2)`,
			`ALTER TABLE invoice_items  ALTER COLUMN net_price  TYPE numeric(12,2)`,
			`ALTER TABLE invoice_items  ALTER COLUMN tax_amount TYPE numeric(12,2)`,
			`ALTER TABLE invoice_items  ALTER COLUMN gross_price TYPE numeric(12,2)`,
			`ALTER TABLE payments       ALTER COLUMN amount     TYPE numeric(12,2)`,
		}
		for _, stmt := range alters {
			if err := tx.Exec(stmt).Error; err != nil {
				return fmt.Errorf("money type migration failed on: %s - %w", stmt, err)
			}
		}

		// --- Composite / helpful indexes (idempotent) ---
		indexes := []string{
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_invoice_versions_invoice_id_version_no ON invoice_versions (invoice_id, version_no)`,
			`CREATE INDEX IF NOT EXISTS idx_payments_invoice_paid_at ON payments (invoice_id, paid_at)`,
			`CREATE INDEX IF NOT EXISTS idx_invoice_items_invoice ON invoice_items (invoice_id)`,
			`CREATE INDEX IF NOT EXISTS idx_invoice_items_article ON invoice_items (article_id)`,
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_idempotency_keys_key ON idempotency_keys (key)`,
		}
		for _, stmt := range indexes {
			if err := tx.Exec(stmt).Error; err != nil {
				return fmt.Errorf("index migration failed on: %s - %w", stmt, err)
			}
		}

		// --- Foreign key: invoice_items.article_id -> articles.id (RESTRICT/RESTRICT) ---
		fk := `
DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1
		FROM pg_constraint
		WHERE conrelid = 'invoice_items'::regclass
		  AND conname  = 'fk_invoice_items_article'
	) THEN
		ALTER TABLE invoice_items
		ADD CONSTRAINT fk_invoice_items_article
		FOREIGN KEY (article_id)
		REFERENCES articles(id)
		ON UPDATE RESTRICT
		ON DELETE RESTRICT;
	END IF;
END $$;`
		if err := tx.Exec(fk).Error; err != nil {
			return fmt.Errorf("foreign key migration failed: %w", err)
		}

		// --- NOT NULL for invoice_items.article_id (idempotent) ---
		if err := tx.Exec(`ALTER TABLE invoice_items ALTER COLUMN article_id SET NOT NULL`).Error; err != nil {
			return fmt.Errorf("set NOT NULL on invoice_items.article_id failed: %w", err)
		}

		// --- Basic CHECK constraints (idempotent) ---
		checks := []string{
			// Non-negative article price
			`DO $$
			BEGIN
				IF NOT EXISTS (
					SELECT 1 FROM pg_constraint
					WHERE conrelid = 'articles'::regclass
					  AND conname  = 'chk_articles_unit_price_nonneg'
				) THEN
					ALTER TABLE articles
					ADD CONSTRAINT chk_articles_unit_price_nonneg
					CHECK (unit_price >= 0);
				END IF;
			END $$;`,
			// Payments.amount >= 0
			`DO $$
			BEGIN
				IF NOT EXISTS (
					SELECT 1 FROM pg_constraint
					WHERE conrelid = 'payments'::regclass
					  AND conname  = 'chk_payments_amount_nonneg'
				) THEN
					ALTER TABLE payments
					ADD CONSTRAINT chk_payments_amount_nonneg
					CHECK (amount >= 0);
				END IF;
			END $$;`,
			// Invoice items: amount >= 0
			`DO $$
			BEGIN
				IF NOT EXISTS (
					SELECT 1 FROM pg_constraint
					WHERE conrelid = 'invoice_items'::regclass
					  AND conname  = 'chk_invoice_items_amount_nonneg'
				) THEN
					ALTER TABLE invoice_items
					ADD CONSTRAINT chk_invoice_items_amount_nonneg
					CHECK (amount >= 0);
				END IF;
			END $$;`,
		}
		for _, stmt := range checks {
			if err := tx.Exec(stmt).Error; err != nil {
				return fmt.Errorf("check constraint migration failed: %w", err)
			}
		}

		return nil
	})
}
