package database

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// DB is initialized in your existing Connect() (keep your current implementation).

// GetTenantDB returns a *gorm.DB bound to the request's tenant.
// Prefer an existing per-request TX (middlewares.TenantTx), else fall back to a session
// where we set the search_path for the connection.
func GetTenantDB(c *fiber.Ctx) (*gorm.DB, error) {
	// If a per-request transaction exists, use it.
	if v := c.Locals("tx"); v != nil {
		if tx, ok := v.(*gorm.DB); ok && tx != nil {
			return tx, nil
		}
	}

	// Otherwise, prepare a tenant-scoped session.
	schema, _ := c.Locals("schema").(string)
	if strings.TrimSpace(schema) == "" {
		return nil, errors.New("tenant schema missing")
	}
	if DB == nil {
		return nil, errors.New("database not initialized")
	}

	// Use a dedicated session; pin search_path for this connection.
	sess := DB.Session(&gorm.Session{})
	// Try SET LOCAL (no-op outside TX) and then hard SET as fallback.
	if err := sess.Exec(`SET LOCAL search_path = "` + schema + `", public`).Error; err != nil {
		// Ignore error; SET LOCAL can be invalid outside TX. Use SET instead.
	}
	if err := sess.Exec(`SET search_path = "` + schema + `", public`).Error; err != nil {
		return nil, fmt.Errorf("set search_path failed: %w", err)
	}
	return sess, nil
}
