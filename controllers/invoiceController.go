package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"fakturierung-backend/database"
	"fakturierung-backend/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ---------- Helpers

type versionSnapshot struct {
	InvoiceNumber string               `json:"invoice_number"`
	CustomerID    uint                 `json:"customer_id"`
	Subtotal      float64              `json:"subtotal"`
	TaxTotal      float64              `json:"tax_total"`
	Total         float64              `json:"total"`
	Draft         bool                 `json:"draft"`     // true => quotation, false => invoice
	Published     bool                 `json:"published"` // for reference
	PublishedAt   *time.Time           `json:"published_at"`
	Items         []models.InvoiceItem `json:"items"`
	PaidTotal     float64              `json:"paid_total"`
}

func kindFromDraft(draft bool) string {
	if draft {
		return "quotation"
	}
	return "invoice"
}

func nextVersionNo(tx *gorm.DB, invoiceID uint) (int, error) {
	var n int
	err := tx.Model(&models.InvoiceVersion{}).
		Where("invoice_id = ?", invoiceID).
		Select("COALESCE(MAX(version_no),0)").Scan(&n).Error
	if err != nil {
		return 0, err
	}
	return n + 1, nil
}

func snapshotInvoice(tx *gorm.DB, inv *models.Invoice) error {
	verNo, err := nextVersionNo(tx, inv.ID)
	if err != nil {
		return err
	}

	// Load items (just in case controller caller didn't preload)
	var items []models.InvoiceItem
	if err := tx.Model(&models.InvoiceItem{}).Where("invoice_id = ?", inv.ID).Find(&items).Error; err != nil {
		return err
	}

	snap := versionSnapshot{
		InvoiceNumber: inv.InvoiceNumber,
		CustomerID:    inv.CId,
		Subtotal:      inv.Subtotal,
		TaxTotal:      inv.TaxTotal,
		Total:         inv.Total,
		Draft:         inv.Draft,
		Published:     inv.Published,
		PublishedAt:   inv.PublishedAt,
		Items:         items,
		PaidTotal:     inv.PaidTotal,
	}
	js, err := json.Marshal(snap)
	if err != nil {
		return err
	}

	record := models.InvoiceVersion{
		InvoiceID: inv.ID,
		VersionNo: verNo,
		Kind:      kindFromDraft(inv.Draft),
		Snapshot:  js,
	}

	return tx.Create(&record).Error
}

func recalcPaidTotal(tx *gorm.DB, invoiceID uint) (float64, error) {
	var sum float64
	// Postgres SUM on empty set returns NULL => coalesce to 0
	if err := tx.
		Model(&models.Payment{}).
		Where("invoice_id = ?", invoiceID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&sum).Error; err != nil {
		return 0, err
	}
	if err := tx.Model(&models.Invoice{}).Where("id = ?", invoiceID).Update("paid_total", sum).Error; err != nil {
		return 0, err
	}
	return sum, nil
}

// ---------- Core endpoints

// POST /api/invoice
// Body may include: type="quotation"|"invoice" (preferred) or legacy draft=true|false
func CreateInvoice(c *fiber.Ctx) error {
	var data map[string]string
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid input"})
	}

	customerID, err := strconv.Atoi(data["customer_id"])
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid customer ID"})
	}

	docType := strings.ToLower(strings.TrimSpace(data["type"]))
	draft := false
	switch docType {
	case "quotation":
		draft = true
	case "invoice":
		draft = false
	default:
		draft, _ = strconv.ParseBool(data["draft"])
	}

	items, subtotal, taxTotal, err := extractInvoiceItems(data)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}

	db, err := database.GetTenantDB(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "tenant db unavailable"})
	}

	var inv models.Invoice
	err = db.Transaction(func(tx *gorm.DB) error {
		inv = models.Invoice{
			InvoiceNumber: "", // assigned on publish
			CId:           uint(customerID),
			Items:         items,
			Subtotal:      subtotal,
			TaxTotal:      taxTotal,
			Total:         subtotal + taxTotal,
			Draft:         draft,
			Published:     false,
			PublishedAt:   nil,
			PaidTotal:     0,
		}
		if err := tx.Create(&inv).Error; err != nil {
			return err
		}
		return snapshotInvoice(tx, &inv) // v1
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Could not create invoice", "error": err.Error()})
	}

	return c.JSON(inv)
}

// PUT /api/invoices/:id
// Updates the live invoice (only if found) and snapshots a new version.
func UpdateInvoice(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid invoice id"})
	}

	var data map[string]string
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid input"})
	}

	customerID, err := strconv.Atoi(data["customer_id"])
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid customer ID"})
	}

	items, subtotal, taxTotal, err := extractInvoiceItems(data)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}

	db, err := database.GetTenantDB(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "tenant db unavailable"})
	}

	var out models.Invoice
	err = db.Transaction(func(tx *gorm.DB) error {
		var existing models.Invoice
		if err := tx.Preload("Items").First(&existing, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "invoice not found")
			}
			return err
		}

		// Update header
		if err := tx.Model(&models.Invoice{}).
			Where("id = ?", id).
			Updates(map[string]any{
				"c_id":      uint(customerID),
				"subtotal":  subtotal,
				"tax_total": taxTotal,
				"total":     subtotal + taxTotal,
			}).Error; err != nil {
			return err
		}

		// Replace items
		if err := tx.Model(&existing).Association("Items").Replace(items); err != nil {
			return err
		}

		// Reload and snapshot
		if err := tx.Preload(clause.Associations).First(&out, "id = ?", id).Error; err != nil {
			return err
		}
		return snapshotInvoice(tx, &out)
	})
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return c.Status(fe.Code).JSON(fiber.Map{"message": fe.Message})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "update failed", "error": err.Error()})
	}

	return c.JSON(out)
}

// PUT /api/invoices/:id/convert
// Body: {"target":"quotation"} or {"target":"invoice"}
// We allow conversion regardless of payments/published; everything is versioned.
func ConvertInvoice(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid invoice id"})
	}

	var body map[string]string
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid input"})
	}
	target := strings.ToLower(strings.TrimSpace(body["target"]))
	if target != "quotation" && target != "invoice" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "target must be 'quotation' or 'invoice'"})
	}

	db, err := database.GetTenantDB(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "tenant db unavailable"})
	}

	var out models.Invoice
	err = db.Transaction(func(tx *gorm.DB) error {
		// Toggle Draft based on target
		newDraft := (target == "quotation")
		if err := tx.Model(&models.Invoice{}).Where("id = ?", id).Update("draft", newDraft).Error; err != nil {
			return err
		}
		if err := tx.Preload(clause.Associations).First(&out, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.ErrNotFound
			}
			return err
		}
		return snapshotInvoice(tx, &out)
	})
	if err != nil {
		if errors.Is(err, fiber.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "invoice not found"})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "conversion failed", "error": err.Error()})
	}

	return c.JSON(out)
}

// PUT /api/invoices/:id/publish
// Assign a number (if empty), set Published=true & PublishedAt=now; snapshot.
func PublishInvoice(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid invoice id"})
	}

	var payload map[string]string
	_ = c.BodyParser(&payload) // optional: { "invoice_number": "..." }

	db, err := database.GetTenantDB(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "tenant db unavailable"})
	}

	var out models.Invoice
	err = db.Transaction(func(tx *gorm.DB) error {
		var inv models.Invoice
		if err := tx.First(&inv, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.ErrNotFound
			}
			return err
		}

		now := time.Now().UTC()
		number := strings.TrimSpace(payload["invoice_number"])
		if number == "" && inv.InvoiceNumber == "" {
			number = generateInvoiceNumber()
		} else if number == "" {
			number = inv.InvoiceNumber
		}

		if err := tx.Model(&models.Invoice{}).
			Where("id = ?", id).
			Updates(map[string]any{
				"invoice_number": number,
				"published":      true,
				"published_at":   &now,
				"draft":          false, // publishing implies invoice form
			}).Error; err != nil {
			return err
		}

		if err := tx.Preload(clause.Associations).First(&out, "id = ?", id).Error; err != nil {
			return err
		}
		return snapshotInvoice(tx, &out)
	})
	if err != nil {
		if errors.Is(err, fiber.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "invoice not found"})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "publish failed", "error": err.Error()})
	}

	return c.JSON(out)
}

// GET /api/invoices?type=quotation|invoice|published&limit=50&offset=0
func GetInvoices(c *fiber.Ctx) error {
	var invoices []models.Invoice

	typ := strings.ToLower(strings.TrimSpace(c.Query("type")))
	limit := parseIntDefault(c.Query("limit"), 50)
	offset := parseIntDefault(c.Query("offset"), 0)

	db, err := database.GetTenantDB(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "tenant db unavailable"})
	}

	q := db.Model(&models.Invoice{}).Preload("Customer")
	switch typ {
	case "quotation":
		q = q.Where("draft = ?", true)
	case "invoice":
		q = q.Where("draft = ? AND published = ?", false, false)
	case "published":
		q = q.Where("published = ?", true)
	}
	if err := q.Limit(limit).Offset(offset).Find(&invoices).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "db error"})
	}

	return c.JSON(fiber.Map{
		"invoices": invoices,
		"message":  "success",
	})
}

// GET /api/invoice/:id
func GetInvoice(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invoice not found"})
	}

	db, err := database.GetTenantDB(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "tenant db unavailable"})
	}

	var invoice models.Invoice
	if err := db.Model(&models.Invoice{}).Preload(clause.Associations).First(&invoice, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Invoice not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "db error"})
	}

	return c.JSON(fiber.Map{
		"invoice": invoice,
		"message": "success",
	})
}

// ---------- Versions

// GET /api/invoices/:id/versions
func GetInvoiceVersions(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid invoice id"})
	}

	db, err := database.GetTenantDB(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "tenant db unavailable"})
	}

	var versions []models.InvoiceVersion
	if err := db.Where("invoice_id = ?", id).Order("version_no ASC").Find(&versions).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "db error"})
	}
	return c.JSON(fiber.Map{"versions": versions})
}

// ---------- Payments

// POST /api/invoices/:id/payments
// Body: { "amount":"123.45", "method":"bank-transfer", "reference":"...", "note":"...", "paid_at":"2025-08-27T10:00:00Z" }
func CreatePayment(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid invoice id"})
	}

	var body map[string]string
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid input"})
	}

	amount, err := strconv.ParseFloat(strings.TrimSpace(body["amount"]), 64)
	if err != nil || amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid amount"})
	}

	paidAt := time.Now().UTC()
	if ts := strings.TrimSpace(body["paid_at"]); ts != "" {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			paidAt = t.UTC()
		}
	}

	db, err := database.GetTenantDB(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "tenant db unavailable"})
	}

	var p models.Payment
	err = db.Transaction(func(tx *gorm.DB) error {
		// ensure invoice exists
		var inv models.Invoice
		if err := tx.First(&inv, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.ErrNotFound
			}
			return err
		}

		p = models.Payment{
			InvoiceID: uint(id),
			Amount:    amount,
			Method:    strings.TrimSpace(body["method"]),
			Reference: strings.TrimSpace(body["reference"]),
			Note:      strings.TrimSpace(body["note"]),
			PaidAt:    paidAt,
		}
		if err := tx.Create(&p).Error; err != nil {
			return err
		}

		// update summary on invoice
		if _, err := recalcPaidTotal(tx, uint(id)); err != nil {
			return err
		}

		// snapshot the invoice after payment change
		if err := tx.First(&inv, "id = ?", id).Error; err == nil {
			return snapshotInvoice(tx, &inv)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, fiber.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "invoice not found"})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "payment failed", "error": err.Error()})
	}

	return c.JSON(p)
}

// GET /api/invoices/:id/payments
func ListPayments(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid invoice id"})
	}

	db, err := database.GetTenantDB(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "tenant db unavailable"})
	}

	var payments []models.Payment
	if err := db.Where("invoice_id = ?", id).Order("paid_at ASC, id ASC").Find(&payments).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "db error"})
	}
	return c.JSON(fiber.Map{"payments": payments})
}

// ---------- Shared utilities

func extractInvoiceItems(data map[string]string) ([]models.InvoiceItem, float64, float64, error) {
	var items []models.InvoiceItem
	var subtotal float64
	var taxTotal float64

	taxRate := 0.2 // TODO: move to tenant/product config

	for i := 0; ; i++ {
		prefix := fmt.Sprintf("articles[%d]", i)

		articleID, ok := data[prefix+"[article_id]"]
		if !ok {
			break
		}

		amountStr := data[prefix+"[amount]"]
		unitPriceStr := data[prefix+"[unit_price]"]
		description := data[prefix+"[description]"]

		amount, err := strconv.Atoi(amountStr)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("Invalid amount at index %d", i)
		}
		unitPrice, err := strconv.ParseFloat(unitPriceStr, 64)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("Invalid unit price at index %d", i)
		}

		net := unitPrice * float64(amount)
		tax := net * taxRate
		gross := net + tax

		subtotal += net
		taxTotal += tax

		items = append(items, models.InvoiceItem{
			ArticleID:   articleID,
			Description: description,
			Amount:      amount,
			UnitPrice:   unitPrice,
			TaxRate:     taxRate,
			NetPrice:    net,
			TaxAmount:   tax,
			GrossPrice:  gross,
		})
	}
	return items, subtotal, taxTotal, nil
}

func parseIntDefault(s string, def int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && v >= 0 {
		return v
	}
	return def
}

func generateInvoiceNumber() string {
	return time.Now().Format("20060102-150405.000")
}
