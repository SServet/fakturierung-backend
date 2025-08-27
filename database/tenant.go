package database

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// DB is your global *gorm.DB initialized at startup (see database/init or main.go).

// GetTenantDB returns a tenant-scoped *gorm.DB based on auth middleware locals.
// It verifies that the user belongs to the claimed schema and sets search_path.
func GetTenantDB(c *fiber.Ctx) (*gorm.DB, error) {
	schema, _ := c.Locals("schema").(string)
	userID, _ := c.Locals("userID").(string)
	if schema == "" || userID == "" {
		return nil, errors.New("missing auth context")
	}

	// Verify the user ↔ schema mapping in the public.users table.
	var n int64
	if err := DB.
		Table(`public.users`).
		Where(`id = ? AND schema_name = ?`, userID, schema).
		Count(&n).Error; err != nil {
		return nil, fmt.Errorf("user check failed: %w", err)
	}
	if n == 0 {
		return nil, errors.New("schema does not belong to user")
	}

	// Set the search_path for this session to the tenant schema (then public).
	tenant := DB.Session(&gorm.Session{})
	if err := tenant.Exec(`SET search_path = "` + schema + `", public`).Error; err != nil {
		return nil, fmt.Errorf("set search_path failed: %w", err)
	}
	return tenant, nil
}
