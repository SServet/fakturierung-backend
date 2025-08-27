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

type ArticleDTO struct {
	Name        string  `json:"name" validate:"required,min=1"`
	Description string  `json:"description" validate:"omitempty"`
	UnitPrice   float64 `json:"unit_price" validate:"required,gt=0"`
	Active      bool    `json:"active" validate:"required"`
}

// Pointer-based for partial updates
type ArticleUpdateDTO struct {
	Name        *string  `json:"name" validate:"omitempty,min=0"`
	Description *string  `json:"description" validate:"omitempty"`
	UnitPrice   *float64 `json:"unit_price" validate:"omitempty,gt=0"`
	Active      *bool    `json:"active" validate:"omitempty"`
}

// POST /api/article  (batch create: accepts JSON array of ArticleDTO)
func CreateArticles(c *fiber.Ctx) error {
	var inputs []ArticleDTO
	if err := c.BodyParser(&inputs); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if len(inputs) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "no articles provided")
	}

	for i := range inputs {
		if err := middlewares.ValidateStruct(inputs[i]); err != nil {
			return err
		}
		utils.NormalizeDTO(&inputs[i])
	}

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "tenant db unavailable")
	}

	articles := make([]models.Article, 0, len(inputs))
	for _, in := range inputs {
		articles = append(articles, models.Article{
			Name:        in.Name,
			Description: in.Description,
			UnitPrice:   in.UnitPrice, // already rounded in NormalizeDTO
			Active:      in.Active,
		})
	}

	if err := db.CreateInBatches(&articles, 100).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not create articles")
	}
	return c.Status(fiber.StatusCreated).JSON(articles)
}

// PUT /api/articles/:id
func UpdateArticle(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing article id in path")
	}

	var in ArticleUpdateDTO
	if err := middlewares.BindAndValidate(c, &in); err != nil {
		return err
	}
	utils.NormalizePtrDTO(&in)

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "tenant db unavailable")
	}

	// Ensure exists
	var existing models.Article
	if err := db.First(&existing, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "article not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "db error")
	}

	// Only non-nil fields are updated
	if err := db.Model(&models.Article{}).Where("id = ?", id).Updates(in).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "could not update article")
	}

	var out models.Article
	if err := db.First(&out, "id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to reload article")
	}
	return c.JSON(out)
}

// GET /api/articles?q=...&active=true|false&limit=50&offset=0
func GetArticles(c *fiber.Ctx) error {
	var articles []models.Article

	q := strings.TrimSpace(c.Query("q"))
	activeStr := strings.TrimSpace(c.Query("active"))
	limit := parseIntDefault(c.Query("limit"), 50)
	offset := parseIntDefault(c.Query("offset"), 0)

	db, err := database.GetTenantDB(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "tenant db unavailable")
	}

	query := db.Model(&models.Article{})
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ?", like, like)
	}
	if activeStr != "" {
		if active, err := strconv.ParseBool(activeStr); err == nil {
			query = query.Where("active = ?", active)
		}
	}

	if err := query.Limit(limit).Offset(offset).Find(&articles).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "db error")
	}
	return c.JSON(fiber.Map{
		"articles": articles,
		"message":  "success",
	})
}

func parseIntDefault(s string, def int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && v >= 0 {
		return v
	}
	return def
}
