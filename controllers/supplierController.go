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

type SupplierCreateDTO struct {
	CompanyName  string `json:"company_name" validate:"required,min=1"`
	Address      string `json:"address" validate:"required,min=1"`
	City         string `json:"city" validate:"required,min=1"`
	Country      string `json:"country" validate:"required,min=1"`
	Zip          string `json:"zip" validate:"required,min=1"`
	Homepage     string `json:"homepage" validate:"omitempty,url"`
	UID          string `json:"uid" validate:"omitempty"`
	Email        string `json:"email" validate:"required,email"`
	PhoneNumber  string `json:"phone_number" validate:"required"`
	MobileNumber string `json:"mobile_number" validate:"required"`
}

type SupplierUpdateDTO struct {
	Address      string `json:"address" validate:"omitempty"`
	City         string `json:"city" validate:"omitempty"`
	Country      string `json:"country" validate:"omitempty"`
	Zip          string `json:"zip" validate:"omitempty"`
	Homepage     string `json:"homepage" validate:"omitempty,url"`
	UID          string `json:"uid" validate:"omitempty"`
	Email        string `json:"email" validate:"omitempty,email"`
	PhoneNumber  string `json:"phone_number" validate:"omitempty"`
	MobileNumber string `json:"mobile_number" validate:"omitempty"`
}

// POST /api/supplier
func CreateSupplier(c *fiber.Ctx) error {
	var in SupplierCreateDTO
	if err := middlewares.BindAndValidate(c, &in); err != nil {
		return err
	}

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "tenant db unavailable")
	}

	supplier := models.Supplier{
		CompanyName:  strings.TrimSpace(in.CompanyName),
		Address:      strings.TrimSpace(in.Address),
		City:         strings.TrimSpace(in.City),
		Country:      strings.TrimSpace(in.Country),
		Zip:          strings.TrimSpace(in.Zip),
		Homepage:     strings.TrimSpace(in.Homepage),
		UID:          strings.TrimSpace(in.UID),
		Email:        strings.TrimSpace(in.Email),
		PhoneNumber:  strings.TrimSpace(in.PhoneNumber),
		MobileNumber: strings.TrimSpace(in.MobileNumber),
	}

	if err := db.Create(&supplier).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "could not create supplier")
	}
	return c.JSON(supplier)
}

// PUT /api/supplier/:id
func UpdateSupplier(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing supplier id in path")
	}

	var in SupplierUpdateDTO
	if err := middlewares.BindAndValidate(c, &in); err != nil {
		return err
	}

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "tenant db unavailable")
	}

	// Ensure exists
	var existing models.Supplier
	if err := db.First(&existing, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "supplier not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "db error")
	}

	updates := map[string]interface{}{
		"address":       strings.TrimSpace(in.Address),
		"city":          strings.TrimSpace(in.City),
		"country":       strings.TrimSpace(in.Country),
		"zip":           strings.TrimSpace(in.Zip),
		"homepage":      strings.TrimSpace(in.Homepage),
		"uid":           strings.TrimSpace(in.UID),
		"email":         strings.TrimSpace(in.Email),
		"phone_number":  strings.TrimSpace(in.PhoneNumber),
		"mobile_number": strings.TrimSpace(in.MobileNumber),
	}

	if err := db.Model(&models.Supplier{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "could not update supplier")
	}

	var out models.Supplier
	if err := db.First(&out, "id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to reload supplier")
	}
	return c.JSON(out)
}
