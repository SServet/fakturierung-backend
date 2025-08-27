package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"fakturierung-backend/database"
	"fakturierung-backend/middlewares"
	"fakturierung-backend/models"
	"fakturierung-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ====== DTOs ======

type InvoiceItemDTO struct {
	ArticleID   string  `json:"article_id" validate:"required"`
	Description string  `json:"description" validate:"omitempty"`
	Amount      int     `json:"amount" validate:"required,gt=0"`
	UnitPrice   float64 `json:"unit_price" validate:"required,gt=0"`
}

type InvoiceCreateDTO struct {
	Type       string           `json:"type" validate:"omitempty,oneof=quotation invoice"`
	Draft      *bool            `json:"draft" validate:"omitempty"`
	CustomerID uint             `json:"customer_id" validate:"required,gt=0"`
	Items      []InvoiceItemDTO `json:"items" validate:"required,min=1,dive"`
}

type InvoiceUpdateDTO struct {
	CustomerID uint             `json:"customer_id" validate:"required,gt=0"`
	Items      []InvoiceItemDTO `json:"items" validate:"required,min=1,dive"`
}

type PaymentCreateDTO struct {
	Amount    float64 `json:"amount" validate:"required,gt=0"`
	Method    string  `json:"method" validate:"omitempty"`
	Reference string  `json:"reference" validate:"omitempty"`
	Note      string  `json:"note" validate:"omitempty"`
	PaidAt    string  `json:"paid_at" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
}

// ====== Helpers ======

func toItems(items []InvoiceItemDTO, taxRate float64) ([]models.InvoiceItem, float64, float64) {
	var out []models.InvoiceItem
	var subtotal, taxTotal float64
	for _, it := range items {
		unit := utils.Round2(it.UnitPrice)
		net := utils.Round2(unit * float64(it.Amount))
		tax := utils.Round2(net * taxRate)
		gross := utils.Round2(net + tax)

		subtotal = utils.Round2(subtotal + net)
		taxTotal = utils.Round2(taxTotal + tax)

		out = append(out, models.InvoiceItem{
			ArticleID:   it.ArticleID,
			Description: it.Description,
			Amount:      it.Amount,
			UnitPrice:   unit,
			TaxRate:     taxRate,
			NetPrice:    net,
			TaxAmount:   tax,
			GrossPrice:  gross,
		})
	}
	return out, subtotal, taxTotal
}

// Backward-compatible parser for old x-www-form-urlencoded bracket keys.
func extractInvoiceItems(data map[string]string) ([]models.InvoiceItem, float64, float64, error) {
	var items []models.InvoiceItem
	var subtotal float64
	var taxTotal float64
	taxRate := 0.2 // TODO: tenant config

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
			return nil, 0, 0, fmt.Errorf("invalid amount at index %d", i)
		}
		unitPrice, err := strconv.ParseFloat(unitPriceStr, 64)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("invalid unit price at index %d", i)
		}

		unit := utils.Round2(unitPrice)
		net := utils.Round2(unit * float64(amount))
		tax := utils.Round2(net * taxRate)
		gross := utils.Round2(net + tax)

		subtotal = utils.Round2(subtotal + net)
		taxTotal = utils.Round2(taxTotal + tax)

		items = append(items, models.InvoiceItem{
			ArticleID:   articleID,
			Description: description,
			Amount:      amount,
			UnitPrice:   unit,
			TaxRate:     taxRate,
			NetPrice:    net,
			TaxAmount:   tax,
			GrossPrice:  gross,
		})
	}
	return items, subtotal, taxTotal, nil
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
		Select("COALESCE(MAX(version_no), 0)").Scan(&n).Error
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
	var items []models.InvoiceItem
	if err := tx.Model(&models.InvoiceItem{}).Where("invoice_id = ?", inv.ID).Find(&items).Error; err != nil {
		return err
	}

	type versionSnapshot struct {
		InvoiceNumber string               `json:"invoice_number"`
		CustomerID    uint                 `json:"customer_id"`
		Subtotal      float64              `json:"subtotal"`
		TaxTotal      float64              `json:"tax_total"`
		Total         float64              `json:"total"`
		Draft         bool                 `json:"draft"`
		Published     bool                 `json:"published"`
		PublishedAt   *time.Time           `json:"published_at"`
		Items         []models.InvoiceItem `json:"items"`
		PaidTotal     float64              `json:"paid_total"`
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
	if err := tx.Model(&models.Payment{}).
		Where("invoice_id = ?", invoiceID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&sum).Error; err != nil {
		return 0, err
	}
	sum = utils.Round2(sum)
	if err := tx.Model(&models.Invoice{}).Where("id = ?", invoiceID).Update("paid_total", sum).Error; err != nil {
		return 0, err
	}
	return sum, nil
}

// Ensure all referenced Article IDs exist (and are active by default).
func validateArticleRefs(tx *gorm.DB, items []models.InvoiceItem, requireActive bool) error {
	if len(items) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "no items provided")
	}
	idsSet := make(map[string]struct{}, len(items))
	for _, it := range items {
		if strings.TrimSpace(it.ArticleID) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "article_id is required")
		}
		idsSet[it.ArticleID] = struct{}{}
	}
	ids := make([]string, 0, len(idsSet))
	for id := range idsSet {
		ids = append(ids, id)
	}

	type row struct{ Id string }
	var rows []row
	q := tx.Model(&models.Article{}).Select("id").Where("id IN ?", ids)
	if requireActive {
		q = q.Where("active = ?", true)
	}
	if err := q.Find(&rows).Error; err != nil {
		return err
	}
	found := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		found[r.Id] = struct{}{}
	}
	var missing []string
	for _, id := range ids {
		if _, ok := found[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		return fiber.NewError(fiber.StatusBadRequest, "one or more article_id do not exist or are inactive")
	}
	return nil
}

// ====== Core endpoints ======

// POST /api/invoice
func CreateInvoice(c *fiber.Ctx) error {
	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "tenant db unavailable")
	}

	var draft bool
	var items []models.InvoiceItem
	var subtotal, taxTotal float64
	var customerID uint

	if strings.Contains(strings.ToLower(c.Get("Content-Type")), "application/json") {
		var in InvoiceCreateDTO
		if err := middlewares.BindAndValidate(c, &in); err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(in.Type)) {
		case "quotation":
			draft = true
		case "invoice":
			draft = false
		default:
			if in.Draft != nil {
				draft = *in.Draft
			}
		}
		items, subtotal, taxTotal = toItems(in.Items, 0.2)
		customerID = in.CustomerID
	} else {
		var data map[string]string
		if err := c.BodyParser(&data); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if v := strings.TrimSpace(data["type"]); v != "" {
			draft = strings.ToLower(v) == "quotation"
		} else {
			d, _ := strconv.ParseBool(data["draft"])
			draft = d
		}
		cid, err := strconv.Atoi(data["customer_id"])
		if err != nil || cid <= 0 {
			return fiber.NewError(fiber.StatusBadRequest, "invalid customer id")
		}
		customerID = uint(cid)
		items, subtotal, taxTotal, err = extractInvoiceItems(data)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
	}

	total := utils.Round2(subtotal + taxTotal)

	var out models.Invoice
	err = db.Transaction(func(tx *gorm.DB) error {
		// Validate that all article IDs are present and active
		if err := validateArticleRefs(tx, items, true); err != nil {
			return err
		}

		invoice := models.Invoice{
			InvoiceNumber: "",
			CId:           customerID,
			Items:         items,
			Subtotal:      utils.Round2(subtotal),
			TaxTotal:      utils.Round2(taxTotal),
			Total:         total,
			Draft:         draft,
			Published:     false,
			PublishedAt:   nil,
			PaidTotal:     0,
		}
		if err := tx.Create(&invoice).Error; err != nil {
			return err
		}
		if err := snapshotInvoice(tx, &invoice); err != nil {
			return err
		}
		out = invoice
		return nil
	})
	if err != nil {
		// If it’s an FK violation or validation, we surface a clean 400/409 message via ErrorHandler.
		return err
	}
	return c.JSON(out)
}

// PUT /api/invoices/:id
func UpdateInvoice(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid invoice id")
	}

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "tenant db unavailable")
	}

	var existing models.Invoice
	if err := db.Preload("Items").First(&existing, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "invoice not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "db error")
	}

	var items []models.InvoiceItem
	var subtotal, taxTotal float64
	var customerID uint

	if strings.Contains(strings.ToLower(c.Get("Content-Type")), "application/json") {
		var in InvoiceUpdateDTO
		if err := middlewares.BindAndValidate(c, &in); err != nil {
			return err
		}
		items, subtotal, taxTotal = toItems(in.Items, 0.2)
		customerID = in.CustomerID
	} else {
		var data map[string]string
		if err := c.BodyParser(&data); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		cid, err := strconv.Atoi(data["customer_id"])
		if err != nil || cid <= 0 {
			return fiber.NewError(fiber.StatusBadRequest, "invalid customer id")
		}
		customerID = uint(cid)
		var e error
		items, subtotal, taxTotal, e = extractInvoiceItems(data)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, e.Error())
		}
	}

	total := utils.Round2(subtotal + taxTotal)

	var out models.Invoice
	err = db.Transaction(func(tx *gorm.DB) error {
		// Validate articles exist (and active)
		if err := validateArticleRefs(tx, items, true); err != nil {
			return err
		}

		if err := tx.Model(&models.Invoice{}).
			Where("id = ?", id).
			Updates(map[string]any{
				"c_id":      customerID,
				"subtotal":  utils.Round2(subtotal),
				"tax_total": utils.Round2(taxTotal),
				"total":     total,
			}).Error; err != nil {
			return err
		}
		if err := tx.Model(&existing).Association("Items").Replace(items); err != nil {
			return err
		}
		if err := tx.Preload(clause.Associations).First(&out, "id = ?", id).Error; err != nil {
			return err
		}
		return snapshotInvoice(tx, &out)
	})
	if err != nil {
		return err
	}
	return c.JSON(out)
}

// GET /api/invoices
func GetInvoices(c *fiber.Ctx) error {
	var invoices []models.Invoice

	typ := strings.ToLower(strings.TrimSpace(c.Query("type")))
	limit := parseIntDefault(c.Query("limit"), 50)
	offset := parseIntDefault(c.Query("offset"), 0)

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "tenant db unavailable")
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
		return fiber.NewError(fiber.StatusInternalServerError, "db error")
	}
	return c.JSON(fiber.Map{"invoices": invoices, "message": "success"})
}

// GET /api/invoice/:id
func GetInvoice(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invoice not found")
	}

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "tenant db unavailable")
	}

	var invoice models.Invoice
	if err := db.Model(&models.Invoice{}).Preload(clause.Associations).First(&invoice, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "invoice not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "db error")
	}
	return c.JSON(fiber.Map{"invoice": invoice, "message": "success"})
}

// PUT /api/invoices/:id/convert
func ConvertInvoice(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid invoice id")
	}
	var body struct {
		Target string `json:"target" validate:"required,oneof=quotation invoice"`
	}
	if err := middlewares.BindAndValidate(c, &body); err != nil {
		return err
	}
	newDraft := strings.ToLower(strings.TrimSpace(body.Target)) == "quotation"

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "tenant db unavailable")
	}

	var out models.Invoice
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Invoice{}).Where("id = ?", id).Update("draft", newDraft).Error; err != nil {
			return err
		}
		if err := tx.Preload(clause.Associations).First(&out, "id = ?", id).Error; err != nil {
			return err
		}
		return snapshotInvoice(tx, &out)
	})
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "conversion failed")
	}
	return c.JSON(out)
}

// PUT /api/invoices/:id/publish
func PublishInvoice(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid invoice id")
	}
	var payload map[string]string
	_ = json.Unmarshal(c.Body(), &payload)

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "tenant db unavailable")
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
				"draft":          false,
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
			return fiber.NewError(fiber.StatusNotFound, "invoice not found")
		}
		return fiber.NewError(fiber.StatusBadRequest, "publish failed")
	}
	return c.JSON(out)
}

// GET /api/invoices/:id/versions
func GetInvoiceVersions(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid invoice id")
	}
	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "tenant db unavailable")
	}
	var versions []models.InvoiceVersion
	if err := db.Where("invoice_id = ?", id).Order("version_no ASC").Find(&versions).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "db error")
	}
	return c.JSON(fiber.Map{"versions": versions})
}

// POST /api/invoices/:id/payments
func CreatePayment(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid invoice id")
	}
	var in PaymentCreateDTO
	if err := middlewares.BindAndValidate(c, &in); err != nil {
		return err
	}
	amount := utils.Round2(in.Amount)

	paidAt := time.Now().UTC()
	if strings.TrimSpace(in.PaidAt) != "" {
		t, err := time.Parse(time.RFC3339, in.PaidAt)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid paid_at format")
		}
		paidAt = t.UTC()
	}

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "tenant db unavailable")
	}

	var payment models.Payment
	err = db.Transaction(func(tx *gorm.DB) error {
		var inv models.Invoice
		if err := tx.First(&inv, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.ErrNotFound
			}
			return err
		}
		payment = models.Payment{
			InvoiceID: uint(id),
			Amount:    amount,
			Method:    strings.TrimSpace(in.Method),
			Reference: strings.TrimSpace(in.Reference),
			Note:      strings.TrimSpace(in.Note),
			PaidAt:    paidAt,
		}
		if err := tx.Create(&payment).Error; err != nil {
			return err
		}
		if _, err := recalcPaidTotal(tx, uint(id)); err != nil {
			return err
		}
		if err := tx.First(&inv, "id = ?", id).Error; err != nil {
			return err
		}
		return snapshotInvoice(tx, &inv)
	})
	if err != nil {
		if errors.Is(err, fiber.ErrNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "invoice not found")
		}
		return fiber.NewError(fiber.StatusBadRequest, "payment failed")
	}
	return c.JSON(payment)
}

// GET /api/invoices/:id/payments
func ListPayments(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid invoice id")
	}
	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "tenant db unavailable")
	}
	var payments []models.Payment
	if err := db.Where("invoice_id = ?", id).Order("paid_at ASC, id ASC").Find(&payments).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "db error")
	}
	return c.JSON(fiber.Map{"payments": payments})
}

// ====== Utils ======

func generateInvoiceNumber() string {
	return time.Now().Format("20060102-150405.000")
}
