package controllers

import (
	"fakturierung-backend/database"
	"fakturierung-backend/models"

	"github.com/gofiber/fiber/v2"
)

func CreateSupplier(c *fiber.Ctx) error {
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

	supplier := models.Supplier{
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

	if err := tx.Create(&supplier).Error; err != nil {
		tx.Rollback()
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Could not create supplier",
			"error":   err.Error(),
		})
	}

	tx.Commit()
	tenantDB.First(&supplier)
	return c.JSON(supplier)
}

func UpdateSupplier(c *fiber.Ctx) error {
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

	supplier := models.Supplier{
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

	if err := tx.Model(&supplier).Where("company_name = ?", data["company_name"]).Updates(&supplier).Error; err != nil {
		tx.Rollback()
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Could not update company",
			"error":   err.Error(),
		})
	}
	tx.Commit()
	return c.JSON(supplier)
}
