package controllers

import (
	"errors"
	"strconv"
	"strings"

	"fakturierung-backend/database"
	"fakturierung-backend/middlewares"
	"fakturierung-backend/models"
	"fakturierung-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type SupplierCreateDTO struct {
	CompanyName  string `json:"company_name" validate:"required,min=1"`
	Address      string `json:"address" validate:"required,min=1"`
	City         string `json:"city" validate:"required,min=1"`
	Country      string `json:"country" validate:"required,min=1"`
	Zip          string `json:"zip" validate:"required,min=1"`
	PhoneNumber  string `json:"phone_number" validate:"omitempty"`
	MobileNumber string `json:"mobile_number" validate:"omitempty"`
	Homepage     string `json:"homepage" validate:"omitempty"`
	UID          string `json:"uid" validate:"omitempty"`
	Email        string `json:"email" validate:"omitempty,email"`
}

// Pointer-based for partial updates (only non-nil fields are updated)
type SupplierUpdateDTO struct {
	CompanyName  *string `json:"company_name" validate:"omitempty"`
	Address      *string `json:"address" validate:"omitempty"`
	City         *string `json:"city" validate:"omitempty"`
	Country      *string `json:"country" validate:"omitempty"`
	Zip          *string `json:"zip" validate:"omitempty"`
	PhoneNumber  *string `json:"phone_number" validate:"omitempty"`
	MobileNumber *string `json:"mobile_number" validate:"omitempty"`
	Homepage     *string `json:"homepage" validate:"omitempty"`
	UID          *string `json:"uid" validate:"omitempty"`
	Email        *string `json:"email" validate:"omitempty,email"`
}

// POST /api/supplier
func CreateSupplier(c *fiber.Ctx) error {
	var in SupplierCreateDTO
	if err := middlewares.BindAndValidate(c, &in); err != nil {
		return err
	}
	utils.NormalizeDTO(&in)

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "tenant db unavailable")
	}

	supplier := models.Supplier{
		CompanyName:  in.CompanyName,
		Address:      in.Address,
		City:         in.City,
		Country:      in.Country,
		Zip:          in.Zip,
		PhoneNumber:  in.PhoneNumber,
		MobileNumber: in.MobileNumber,
		Homepage:     in.Homepage,
		UID:          in.UID,
		Email:        in.Email,
	}
	if err := db.Create(&supplier).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "could not create supplier")
	}
	return c.Status(fiber.StatusCreated).JSON(supplier)
}

// PUT /api/supplier/:id
func UpdateSupplier(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Params("id"))
	if idStr == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing supplier id in path")
	}
	if _, err := strconv.Atoi(idStr); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid supplier id")
	}

	var in SupplierUpdateDTO
	if err := middlewares.BindAndValidate(c, &in); err != nil {
		return err
	}
	utils.NormalizePtrDTO(&in)

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "tenant db unavailable")
	}

	// Ensure exists first (clean 404)
	var existing models.Supplier
	if err := db.First(&existing, "id = ?", idStr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "supplier not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "db error")
	}

	// Only non-nil fields get applied
	if err := db.Model(&models.Supplier{}).Where("id = ?", idStr).Updates(in).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "could not update supplier")
	}

	var out models.Supplier
	if err := db.First(&out, "id = ?", idStr).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to reload supplier")
	}
	return c.JSON(out)
}
