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

// ===== DTOs =====

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
	Homepage     string `json:"homepage" validate:"omitempty"`
	UID          string `json:"uid" validate:"omitempty"`
	Email        string `json:"email" validate:"required,email"`
}

// Pointer-based partial update; requires optimistic-lock version
type CustomerUpdateDTO struct {
	Version      uint    `json:"version" validate:"required,gt=0"`
	FirstName    *string `json:"first_name" validate:"omitempty"`
	LastName     *string `json:"last_name" validate:"omitempty"`
	Salutation   *string `json:"salutation" validate:"omitempty"`
	Title        *string `json:"title" validate:"omitempty"`
	PhoneNumber  *string `json:"phone_number" validate:"omitempty"`
	MobileNumber *string `json:"mobile_number" validate:"omitempty"`
	CompanyName  *string `json:"company_name" validate:"omitempty"`
	Address      *string `json:"address" validate:"omitempty"`
	City         *string `json:"city" validate:"omitempty"`
	Country      *string `json:"country" validate:"omitempty"`
	Zip          *string `json:"zip" validate:"omitempty"`
	Homepage     *string `json:"homepage" validate:"omitempty"`
	UID          *string `json:"uid" validate:"omitempty"`
	Email        *string `json:"email" validate:"omitempty,email"`
}

// ===== Handlers =====

// POST /api/customer
func CreateCustomer(c *fiber.Ctx) error {
	var in CustomerCreateDTO
	if err := middlewares.BindAndValidate(c, &in); err != nil {
		return err
	}
	utils.NormalizeDTO(&in)

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "tenant db unavailable")
	}

	customer := models.Customer{
		FirstName:    in.FirstName,
		LastName:     in.LastName,
		Salutation:   in.Salutation,
		Title:        in.Title,
		PhoneNumber:  in.PhoneNumber,
		MobileNumber: in.MobileNumber,
		CompanyName:  in.CompanyName,
		Address:      in.Address,
		City:         in.City,
		Country:      in.Country,
		Zip:          in.Zip,
		Homepage:     in.Homepage,
		UID:          in.UID,
		Email:        in.Email,
	}
	if err := db.Create(&customer).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "could not create customer")
	}
	return c.Status(fiber.StatusCreated).JSON(customer)
}

// GET /api/customer/:id
func GetCustomer(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "customer not found")
	}

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "tenant db unavailable")
	}

	var customer models.Customer
	if err := db.Model(&models.Customer{}).First(&customer, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "customer not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "db error")
	}
	return c.JSON(fiber.Map{"customer": customer, "message": "success"})
}

// PUT /api/customer/:id
func UpdateCustomer(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Params("id"))
	if idStr == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing customer id in path")
	}
	if _, err := strconv.Atoi(idStr); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid customer id")
	}

	var in CustomerUpdateDTO
	if err := middlewares.BindAndValidate(c, &in); err != nil {
		return err
	}
	utils.NormalizePtrDTO(&in)

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "tenant db unavailable")
	}

	// Ensure exists first (for clean 404)
	var existing models.Customer
	if err := db.First(&existing, "id = ?", idStr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "customer not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "db error")
	}

	updates := utils.UpdatesFromPtrDTO(&in, nil)
	if len(updates) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "no fields to update")
	}
	updates["version"] = gorm.Expr("version + 1")

	res := db.Model(&models.Customer{}).
		Where("id = ? AND version = ?", idStr, in.Version).
		Updates(updates)
	if res.Error != nil {
		return fiber.NewError(fiber.StatusBadRequest, "could not update customer")
	}
	if res.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusConflict, "stale update, please reload")
	}

	var out models.Customer
	if err := db.First(&out, "id = ?", idStr).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to reload customer")
	}
	return c.JSON(out)
}

// GET /api/customers?limit=50&offset=0
func GetCustomers(c *fiber.Ctx) error {
	limit := utils.ParseIntDefault(c.Query("limit"), 50)
	offset := utils.ParseIntDefault(c.Query("offset"), 0)

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "tenant db unavailable")
	}

	var customers []models.Customer
	if err := db.Model(&models.Customer{}).Limit(limit).Offset(offset).Find(&customers).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "db error")
	}
	return c.JSON(fiber.Map{"customers": customers, "message": "success"})
}
