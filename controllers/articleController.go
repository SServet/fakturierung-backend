package controllers

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"fakturierung-backend/database"
	"fakturierung-backend/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type ArticleInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	UnitPrice   string `json:"unit_price"`
	Active      string `json:"active"`
}

// POST /api/article  (batch create: accepts []ArticleInput)
func CreateArticles(c *fiber.Ctx) error {
	var inputs []ArticleInput
	if err := c.BodyParser(&inputs); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
			"error":   err.Error(),
		})
	}

	tenantDB, err := database.GetTenantDB(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Database error",
			"error":   err.Error(),
		})
	}

	tx := tenantDB.Begin()
	var created []models.Article

	for i, input := range inputs {
		unitPrice, err := strconv.ParseFloat(strings.TrimSpace(input.UnitPrice), 64)
		if err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": fmt.Sprintf("Invalid unit price at index %d", i),
			})
		}
		active, err := strconv.ParseBool(strings.TrimSpace(input.Active))
		if err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": fmt.Sprintf("Invalid active value at index %d", i),
			})
		}

		article := models.Article{
			Name:        strings.TrimSpace(input.Name),
			Description: strings.TrimSpace(input.Description),
			UnitPrice:   unitPrice,
			Active:      active,
		}

		if err := tx.Create(&article).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": fmt.Sprintf("Could not create article at index %d", i),
				"error":   err.Error(),
			})
		}
		created = append(created, article)
	}

	tx.Commit()
	return c.Status(fiber.StatusCreated).JSON(created)
}

// PUT /api/articles/:id  (updated to use :id + WHERE)
func UpdateArticle(c *fiber.Ctx) error {
	id := c.Params("id")
	if strings.TrimSpace(id) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "missing article id in path"})
	}

	var data map[string]string
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "invalid body"})
	}

	unitPriceStr := data["unit_price"]
	unitPrice, err := strconv.ParseFloat(unitPriceStr, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid unit price format"})
	}

	activeStr := data["active"]
	active, err := strconv.ParseBool(activeStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid active value"})
	}

	tenantDB, err := database.GetTenantDB(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Internal Error",
			"error":   err.Error(),
		})
	}

	// Ensure the record exists (to return a clean 404)
	var existing models.Article
	if err := tenantDB.First(&existing, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "article not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "db error"})
	}

	tx := tenantDB.Begin()

	updates := map[string]interface{}{
		"name":        data["name"],
		"description": data["description"],
		"unit_price":  unitPrice,
		"active":      active,
	}

	if err := tx.Model(&models.Article{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Could not update article",
			"error":   err.Error(),
		})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "commit failed"})
	}

	var out models.Article
	if err := tenantDB.First(&out, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to reload article"})
	}

	return c.JSON(out)
}

// GET /api/articles
func GetArticles(c *fiber.Ctx) error {
	var articles []models.Article

	tenantDB, err := database.GetTenantDB(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Internal Error",
			"error":   err.Error(),
		})
	}

	tx := tenantDB.Begin()
	tx.Model(&models.Article{}).Find(&articles)
	tx.Commit()

	return c.JSON(fiber.Map{
		"articles": articles,
		"message":  "success",
	})
}
