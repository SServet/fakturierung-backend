package controllers

import (
	"errors"
	"strings"

	"fakturierung-backend/database"
	"fakturierung-backend/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// POST /api/customer
func CreateCustomer(c *fiber.Ctx) error {
	var data map[string]string
	if err := c.BodyParser(&data); err != nil {
		return err
	}

	tenantDB, err := database.GetTenantDB(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Internal Error",
			"error":   err.Error(),
		})
	}

	tx := tenantDB.Begin()

	customer := models.Customer{
		FirstName:    data["first_name"],
		LastName:     data["last_name"],
		Salutation:   data["salutation"],
		Title:        data["title"],
		PhoneNumber:  data["phone_number"],
		MobileNumber: data["mobile_number"],
		CompanyName:  data["company_name"],
		Address:      data["address"],
		City:         data["city"],
		Country:      data["country"],
		Zip:          data["zip"],
		Homepage:     data["homepage"],
		UID:          data["uid"],
		Email:        data["email"],
	}

	if err := tx.Create(&customer).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Could not create company",
			"error":   err.Error(),
		})
	}

	tx.Commit()
	tenantDB.First(&customer)

	return c.JSON(customer)
}

// GET /api/customer/:id
func GetCustomer(c *fiber.Ctx) error {
	id := c.Params("id")
	if strings.TrimSpace(id) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Customer not found"})
	}

	tenantDB, err := database.GetTenantDB(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Internal Error",
			"error":   err.Error(),
		})
	}

	var customer models.Customer
	tx := tenantDB.Begin()
	if err := tx.Model(&models.Customer{}).First(&customer, "id = ?", id).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Customer not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "db error"})
	}
	tx.Commit()

	return c.JSON(fiber.Map{
		"customer": customer,
		"message":  "success",
	})
}

// PUT /api/customer/:id  (updated to use :id + WHERE)
func UpdateCustomer(c *fiber.Ctx) error {
	id := c.Params("id")
	if strings.TrimSpace(id) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "missing customer id in path"})
	}

	var data map[string]string
	if err := c.BodyParser(&data); err != nil {
		return err
	}

	tenantDB, err := database.GetTenantDB(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Internal Error",
			"error":   err.Error(),
		})
	}

	// Ensure the record exists (for clear 404)
	var existing models.Customer
	if err := tenantDB.First(&existing, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "customer not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "db error"})
	}

	tx := tenantDB.Begin()

	updates := map[string]interface{}{
		"phone_number":  data["phone_number"],
		"mobile_number": data["mobile_number"],
		"address":       data["address"],
		"city":          data["city"],
		"country":       data["country"],
		"zip":           data["zip"],
		"homepage":      data["homepage"],
		"uid":           data["uid"],
		"email":         data["email"],
		// company_name intentionally excluded from update key in this handler
	}

	if err := tx.Model(&models.Customer{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Could not update company",
			"error":   err.Error(),
		})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "commit failed"})
	}

	var out models.Customer
	if err := tenantDB.First(&out, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to reload customer"})
	}

	return c.JSON(out)
}

// GET /api/customers
func GetCustomers(c *fiber.Ctx) error {
	var customers []models.Customer

	tenantDB, err := database.GetTenantDB(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Internal Error",
			"error":   err.Error(),
		})
	}

	tx := tenantDB.Begin()
	tx.Model(&models.Customer{}).Find(&customers)
	tx.Commit()

	return c.JSON(fiber.Map{
		"customers": customers,
		"message":   "success",
	})
}
