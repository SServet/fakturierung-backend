package controllers

import (
	"fakturierung-backend/database"
	"fakturierung-backend/models"

	"github.com/gofiber/fiber/v2"
)

func CreateCustomer(c *fiber.Ctx) error {
	var data map[string]string

	if err := c.BodyParser(&data); err != nil {
		return err
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
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Could not create company",
			"error":   err.Error(),
		})
	}

	tx.Commit()
	tenantDB.First(&customer)
	return c.JSON(customer)
}

func UpdateCustomer(c *fiber.Ctx) error {
	var data map[string]string

	if err := c.BodyParser(&data); err != nil {
		return err
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

	customer := models.Customer{
		PhoneNumber:  data["phone_number"],
		MobileNumber: data["mobile_number"],
		Address:      data["address"],
		City:         data["city"],
		Country:      data["country"],
		Zip:          data["zip"],
		Homepage:     data["homepage"],
		UID:          data["uid"],
		Email:        data["email"],
	}

	if err := tx.Model(&customer).Where("company_name = ?", data["company_name"]).Updates(&customer).Error; err != nil {
		tx.Rollback()
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Could not update company",
			"error":   err.Error(),
		})
	}
	tx.Commit()
	return c.JSON(customer)
}

func GetCustomers(c *fiber.Ctx) error {
	var customers []models.Customer

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

	tx.Model(&models.Customer{}).Find(&customers)
	tx.Commit()
	return c.JSON(fiber.Map{
		"customers": customers,
		"message":   "success",
	})
}
