package controllers

import (
	"fakturierung-backend/database"
	"fakturierung-backend/models"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm/clause"
)

func CreateInvoice(c *fiber.Ctx) error {
	var data map[string]string
	if err := c.BodyParser(&data); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid input"})
	}

	customerID, err := strconv.Atoi(data["customer_id"])
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid customer ID"})
	}

	draft, _ := strconv.ParseBool(data["draft"])         // optional
	published, _ := strconv.ParseBool(data["published"]) // optional

	// Parse article items
	items, subtotal, taxTotal, err := extractInvoiceItems(data)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": err.Error()})
	}

	invoice := models.Invoice{
		InvoiceNumber: data["invoice_number"],
		CId:           uint(customerID),
		Items:         items,
		Subtotal:      subtotal,
		TaxTotal:      taxTotal,
		Total:         subtotal + taxTotal,
		Draft:         draft,
		Published:     published,
	}

	schema := c.Locals("schema").(string)
	if schema == "" {
		return c.Status(400).JSON(fiber.Map{
			"message": "Could not retrieve tenant schema",
		})
	}

	tenantDB, _ := database.GetTenantDB(schema)

	tx := tenantDB.Begin()

	if err := tx.Create(&invoice).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(fiber.Map{"message": "Could not create invoice", "error": err.Error()})
	}

	tx.Commit()

	return c.JSON(invoice)
}

func UpdateInvoice(c *fiber.Ctx) error {
	var data map[string]string
	if err := c.BodyParser(&data); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid input"})
	}

	customerID, err := strconv.Atoi(data["customer_id"])
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid customer ID"})
	}

	draft, _ := strconv.ParseBool(data["draft"])         // optional
	published, _ := strconv.ParseBool(data["published"]) // optional

	// Parse article items
	items, subtotal, taxTotal, err := extractInvoiceItems(data)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": err.Error()})
	}

	invoice := models.Invoice{
		InvoiceNumber: data["invoice_number"],
		CId:           uint(customerID),
		Items:         items,
		Subtotal:      subtotal,
		TaxTotal:      taxTotal,
		Total:         subtotal + taxTotal,
		Draft:         draft,
		Published:     published,
	}

	schema := c.Locals("schema").(string)
	if schema == "" {
		return c.Status(400).JSON(fiber.Map{
			"message": "Could not retrieve tenant schema",
		})
	}

	tenantDB, _ := database.GetTenantDB(schema)

	tx := tenantDB.Begin()

	if err := tx.Model(&invoice).Updates(&invoice).Error; err != nil {
		tx.Rollback()
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Could not update invoice",
			"error":   err.Error(),
		})
	}
	tx.Commit()
	return c.JSON(invoice)
}

func extractInvoiceItems(data map[string]string) ([]models.InvoiceItem, float64, float64, error) {
	var items []models.InvoiceItem
	var subtotal float64
	var taxTotal float64

	taxRate := 0.2 // 20% VAT

	for i := 0; ; i++ {
		prefix := fmt.Sprintf("articles[%d]", i)

		articleID, ok := data[prefix+"[article_id]"]
		if !ok {
			break // No more articles
		}

		amountStr := data[prefix+"[amount]"]
		unitPriceStr := data[prefix+"[unit_price]"]
		activeStr := data[prefix+"[active]"]
		description := data[prefix+"[description]"]

		// Parse amount
		amount, err := strconv.Atoi(amountStr)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("Invalid amount at index %d", i)
		}

		// Parse unit price
		unitPrice, err := strconv.ParseFloat(unitPriceStr, 64)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("Invalid unit price at index %d", i)
		}

		// Parse active (optional, if used in model)
		_, err = strconv.ParseBool(activeStr)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("Invalid active value at index %d", i)
		}

		// Compute prices
		netPrice := unitPrice * float64(amount)
		taxAmount := netPrice * taxRate
		grossPrice := netPrice + taxAmount

		subtotal += netPrice
		taxTotal += taxAmount

		items = append(items, models.InvoiceItem{
			ArticleID:   articleID,
			Description: description,
			Amount:      amount,
			UnitPrice:   unitPrice,
			TaxRate:     taxRate,
			NetPrice:    netPrice,
			TaxAmount:   taxAmount,
			GrossPrice:  grossPrice,
		})
	}

	return items, subtotal, taxTotal, nil
}

func GetInvoices(c *fiber.Ctx) error {
	var invoices []models.Invoice
	//var customer models.Customer
	schema := c.Locals("schema").(string)
	if schema == "" {
		return c.Status(400).JSON(fiber.Map{
			"message": "Could not retrieve tenant schema",
		})
	}

	tenantDB, err := database.GetTenantDB(schema)
	if err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Internal Error",
			"error":   err.Error(),
		})
	}

	tx := tenantDB.Begin()

	tx.Model(&models.Invoice{}).Preload("Customer").Find(&invoices)
	tx.Commit()
	return c.JSON(fiber.Map{
		"invoices": invoices,
		"message":  "success",
	})
}

func GetInvoice(c *fiber.Ctx) error {
	var invoice models.Invoice

	id, err := c.ParamsInt("id")

	if err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Invoice not found",
			"error":   err.Error(),
		})
	}

	schema := c.Locals("schema").(string)
	if schema == "" {
		return c.Status(400).JSON(fiber.Map{
			"message": "Could not retrieve tenant schema",
		})
	}

	tenantDB, err := database.GetTenantDB(schema)
	if err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Internal Error",
			"error":   err.Error(),
		})
	}

	tx := tenantDB.Begin()
	tx.Model(&models.Invoice{}).Preload(clause.Associations).Find(&invoice, id)
	tx.Commit()
	return c.JSON(fiber.Map{
		"invoice": invoice,
		"message": "success",
	})

}

func PublishInvoice(c *fiber.Ctx) error {
	var data map[string]string
	if err := c.BodyParser(&data); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid input"})
	}

	schema := c.Locals("schema").(string)
	if schema == "" {
		return c.Status(400).JSON(fiber.Map{
			"message": "Could not retrieve tenant schema",
		})
	}

	invoice := models.Invoice{
		InvoiceNumber: data["invoice_number"],
	}

	tenantDB, _ := database.GetTenantDB(schema)

	tx := tenantDB.Begin()

	if err := tx.Model(&invoice).Updates(&invoice).Error; err != nil {
		tx.Rollback()
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Could not publish invoice",
			"error":   err.Error(),
		})
	}
	tx.Commit()

	return c.JSON(fiber.Map{
		"message": "success",
	})
}
