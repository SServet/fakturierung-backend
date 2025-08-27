package controllers

import (
	"errors"
	"strings"

	"fakturierung-backend/database"
	"fakturierung-backend/middlewares"
	"fakturierung-backend/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type CustomerCreateDTO struct {
	FirstName    string `json:"first_name" validate:"required,min=1"`
	LastName     string `json:"last_name" validate:"required,min=1"`
	Salutation   string `json:"salutation" validate:"omitempty"`
	Title        string `json:"title" validate:"omitempty"`
	PhoneNumber  string `json:"phone_number" validate:"omitempty"`
	MobileNumber string `json:"mobile_number" validate:"omitempty"`
	CompanyName  string `json:"company_name" validate:"required,min=1"`
	Address      string `json:"address" validate:"required,min=1"`
	City         string `json:"city" validate:"required,min=1"`
	Country      string `json:"country" validate:"required,min=1"`
	Zip          string `json:"zip" validate:"required,min=1"`
	Homepage     string `json:"homepage" validate:"omitempty,url"`
	UID          string `json:"uid" validate:"omitempty"`
	Email        string `json:"email" validate:"required,email"`
}

type CustomerUpdateDTO struct {
	PhoneNumber  string `json:"phone_number" validate:"omitempty"`
	MobileNumber string `json:"mobile_number" validate:"omitempty"`
	Address      string `json:"address" validate:"omitempty"`
	City         string `json:"city" validate:"omitempty"`
	Country      string `json:"country" validate:"omitempty"`
	Zip          string `json:"zip" validate:"omitempty"`
	Homepage     string `json:"homepage" validate:"omitempty,url"`
	UID          string `json:"uid" validate:"omitempty"`
	Email        string `json:"email" validate:"omitempty,email"`
}

// POST /api/customer
func CreateCustomer(c *fiber.Ctx) error {
	var in CustomerCreateDTO
	if err := middlewares.BindAndValidate(c, &in); err != nil {
		return err
	}

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "tenant db unavailable")
	}

	customer := models.Customer{
		FirstName:    strings.TrimSpace(in.FirstName),
		LastName:     strings.TrimSpace(in.LastName),
		Salutation:   strings.TrimSpace(in.Salutation),
		Title:        strings.TrimSpace(in.Title),
		PhoneNumber:  strings.TrimSpace(in.PhoneNumber),
		MobileNumber: strings.TrimSpace(in.MobileNumber),
		CompanyName:  strings.TrimSpace(in.CompanyName),
		Address:      strings.TrimSpace(in.Address),
		City:         strings.TrimSpace(in.City),
		Country:      strings.TrimSpace(in.Country),
		Zip:          strings.TrimSpace(in.Zip),
		Homepage:     strings.TrimSpace(in.Homepage),
		UID:          strings.TrimSpace(in.UID),
		Email:        strings.TrimSpace(in.Email),
	}

	if err := db.Create(&customer).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "could not create customer")
	}
	return c.JSON(customer)
}

// GET /api/customer/:id
func GetCustomer(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "customer not found")
	}

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "tenant db unavailable")
	}

	var customer models.Customer
	if err := db.First(&customer, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "customer not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "db error")
	}
	return c.JSON(fiber.Map{
		"customer": customer,
		"message":  "success",
	})
}

// PUT /api/customer/:id
func UpdateCustomer(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing customer id in path")
	}

	var in CustomerUpdateDTO
	if err := middlewares.BindAndValidate(c, &in); err != nil {
		return err
	}

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "tenant db unavailable")
	}

	// Ensure exists
	var existing models.Customer
	if err := db.First(&existing, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "customer not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "db error")
	}

	updates := map[string]interface{}{
		"phone_number":  strings.TrimSpace(in.PhoneNumber),
		"mobile_number": strings.TrimSpace(in.MobileNumber),
		"address":       strings.TrimSpace(in.Address),
		"city":          strings.TrimSpace(in.City),
		"country":       strings.TrimSpace(in.Country),
		"zip":           strings.TrimSpace(in.Zip),
		"homepage":      strings.TrimSpace(in.Homepage),
		"uid":           strings.TrimSpace(in.UID),
		"email":         strings.TrimSpace(in.Email),
	}
	if err := db.Model(&models.Customer{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "could not update customer")
	}

	var out models.Customer
	if err := db.First(&out, "id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to reload customer")
	}
	return c.JSON(out)
}

// GET /api/customers
func GetCustomers(c *fiber.Ctx) error {
	var customers []models.Customer

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "tenant db unavailable")
	}

	if err := db.Model(&models.Customer{}).Find(&customers).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "db error")
	}

	return c.JSON(fiber.Map{
		"customers": customers,
		"message":   "success",
	})
}
