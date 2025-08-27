package controllers

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"fakturierung-backend/database"
	"fakturierung-backend/middlewares"
	"fakturierung-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func Register(c *fiber.Ctx) error {
	var data map[string]string
	if err := c.BodyParser(&data); err != nil {
		return err
	}

	var mailExist models.User
	database.DB.Where("email = ?", data["email"]).First(&mailExist)
	if mailExist.Email != "" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "email already exists",
		})
	}

	if data["password"] != data["password_confirm"] {
		c.Status(400)
		return c.JSON(fiber.Map{
			"message": "passwords do not match",
		})
	}

	tx := database.DB.Begin()

	user := models.User{
		FirstName: data["first_name"],
		LastName:  data["last_name"],
		Email:     data["email"],
	}
	user.SetPassword(data["password"])
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Could not create User",
			"error":   err.Error(),
		})
	}

	contactPerson := models.ContactPerson{
		FirstName:    data["first_name"],
		LastName:     data["last_name"],
		Salutation:   data["salutation"],
		Title:        data["title"],
		PhoneNumber:  data["phone_number"],
		MobileNumber: data["mobile_number"],
	}
	if err := tx.Create(&contactPerson).Error; err != nil {
		tx.Rollback()
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Could not create contact person",
			"error":   err.Error(),
		})
	}

	company := models.Company{
		CompanyName: data["company_name"],
		Address:     data["address"],
		City:        data["city"],
		Country:     data["country"],
		Zip:         data["zip"],
		Homepage:    data["homepage"],
		UID:         data["uid"],
		UserId:      user.Id,
		PId:         contactPerson.Id,
	}

	schemaName, err := createSchema(company.CompanyName)
	if err != nil {
		tx.Rollback()
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Registration failed due to internal error",
			"error":   err.Error(),
		})
	}
	company.SchemaName = schemaName

	if err := tx.Create(&company).Error; err != nil {
		tx.Rollback()
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Could not create company",
			"error":   err.Error(),
		})
	}

	user.SchemaName = schemaName
	if err := tx.Updates(&user).Error; err != nil {
		tx.Rollback()
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Registration failed",
			"error":   err.Error(),
		})
	}

	err = database.MigrateTenantSchema(schemaName)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Could not migrate tenant schema"})
	}

	tx.Commit()

	database.DB.Preload("User").Preload("ContactPerson").First(&company)
	return c.JSON(company)
}

func createSchema(customer string) (string, error) {
	safeName := strings.ToLower(strings.TrimSpace(customer))
	safeName = strings.ReplaceAll(safeName, " ", "_")
	// Validate schema name (only letters, numbers, underscores; must start with letter/underscore)
	valid := regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)
	if !valid.MatchString(safeName) {
		return "", fmt.Errorf("invalid schema name after sanitization: %s", safeName)
	}

	// Create schema if not exists
	if err := database.DB.Exec("CREATE SCHEMA IF NOT EXISTS " + safeName).Error; err != nil {
		return "", err
	}
	return safeName, nil
}

func Login(c *fiber.Ctx) error {
	var data map[string]string
	if err := c.BodyParser(&data); err != nil {
		return err
	}

	var user models.User

	if _, err := mail.ParseAddress(data["email"]); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Invaild email format",
		})
	}

	database.DB.Exec("SET search_path TO public")
	database.DB.Table("public.users").Where("email = ?", data["email"]).First(&user)

	if _, err := uuid.Parse(user.Id); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Invaild Credentials 1",
			"error":   err.Error(),
		})
	}

	if err := user.ComparePassword(data["password"]); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Invaild Credentials 2",
		})
	}

	token, err := middlewares.GenerateJWT(user.Id, user.SchemaName)
	if err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Invaild Credentials 3",
			"error":   err.Error(),
		})
	}

	/* cookie := fiber.Cookie{
		Name:     "jwt",
		Value:    token,
		Expires:  time.Now().Add(time.Hour * 24),
		HTTPOnly: true,
	}
	c.Cookie(&cookie) */

	err = database.MigrateTenantSchema(user.SchemaName)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Could not migrate tenant schema"})
	}

	return c.JSON(fiber.Map{
		"token":  token,
		"schema": user.SchemaName,
		"user": fiber.Map{
			"id":    user.Id,
			"name":  user.FirstName + " " + user.LastName,
			"email": user.Email,
		},
	})
}

func Logout(c *fiber.Ctx) error {
	cookie := fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
	}
	c.Cookie(&cookie)
	return c.JSON(fiber.Map{
		"message": "success",
	})
}
