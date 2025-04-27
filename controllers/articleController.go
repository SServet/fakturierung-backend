package controllers

import (
	"fakturierung-backend/database"
	"fakturierung-backend/models"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type ArticleInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	UnitPrice   string `json:"unit_price"`
	Active      string `json:"active"`
}

func CreateArticles(c *fiber.Ctx) error {
	var inputs []ArticleInput

	if err := c.BodyParser(&inputs); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"message": "Invalid request body",
			"error":   err.Error(),
		})
	}

	schema := c.Locals("schema").(string)
	if schema == "" {
		return c.Status(400).JSON(fiber.Map{
			"message": "Could not retrieve tenant schema",
		})
	}

	tenantDB, err := database.GetTenantDB(schema)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
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
			return c.Status(400).JSON(fiber.Map{
				"message": fmt.Sprintf("Invalid unit price at index %d", i),
			})
		}

		active, err := strconv.ParseBool(strings.TrimSpace(input.Active))
		if err != nil {
			tx.Rollback()
			return c.Status(400).JSON(fiber.Map{
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
			return c.Status(500).JSON(fiber.Map{
				"message": fmt.Sprintf("Could not create article at index %d", i),
				"error":   err.Error(),
			})
		}

		created = append(created, article)
	}

	tx.Commit()

	return c.Status(201).JSON(created)
}

func UpdateArticle(c *fiber.Ctx) error {
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

	unitPriceStr := data["unit_price"]
	unitPrice, err := strconv.ParseFloat(unitPriceStr, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"message": "Invalid unit price format",
		})
	}

	activeStr := data["active"]
	active, err := strconv.ParseBool(activeStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"message": "Invalid active value",
		})
	}

	tx := tenantDB.Begin()

	article := models.Article{
		Name:        data["name"],
		Description: data["description"],
		UnitPrice:   unitPrice,
		Active:      active,
	}

	if err := tx.Model(&article).Updates(&article).Error; err != nil {
		tx.Rollback()
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Could not update company",
			"error":   err.Error(),
		})
	}
	tx.Commit()
	return c.JSON(article)
}
