package middlewares

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

var validate = validator.New()

// BindAndValidate parses the request body into dst and validates it.
// Returns fiber.ErrBadRequest for parse errors and a validator.ValidationErrors for validation issues.
func BindAndValidate(c *fiber.Ctx, dst interface{}) error {
	if err := c.BodyParser(dst); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	// NOTE: For slices/arrays, call ValidateStruct per-element in the controller.
	return validate.Struct(dst)
}

// ValidateStruct validates any struct value using the shared validator instance.
func ValidateStruct(v interface{}) error {
	return validate.Struct(v)
}
