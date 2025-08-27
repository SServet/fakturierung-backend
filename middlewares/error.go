package middlewares

import (
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// ErrorHandler centralizes error responses and keeps messages sanitized.
func ErrorHandler(c *fiber.Ctx, err error) error {
	// 1) Fiber errors (use their status code + message)
	if fe, ok := err.(*fiber.Error); ok {
		return c.Status(fe.Code).JSON(fiber.Map{"message": fe.Message})
	}

	// 2) Validation errors (422 + per-field info)
	if ve, ok := err.(validator.ValidationErrors); ok {
		out := make(map[string]string, len(ve))
		for _, fe := range ve {
			// fe.Field() is struct field name; you can map to json tag if you prefer
			out[fe.Field()] = fe.Tag()
		}
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"message": "validation failed",
			"errors":  out,
		})
	}

	// 3) Unknown errors (500)
	log.Printf("internal error: %v", err)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"message": "internal server error",
	})
}
