package middlewares

import (
	"log"
	"strings"

	"fakturierung-backend/database"

	"github.com/gofiber/fiber/v2"
)

// TenantTx opens a per-request DB transaction pinned to the tenant schema.
// Order: run AFTER IsAuthenticatedHeader() (so schema/userID are present),
// and AFTER Idempotency() (so idempotency records aren't tied to the handler TX).
func TenantTx() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		schema, _ := c.Locals("schema").(string)
		if strings.TrimSpace(schema) == "" {
			// Public endpoints (e.g., /login) won’t have schema; just proceed.
			return c.Next()
		}

		// Begin TX on the shared DB connection.
		tx := database.DB.Begin()
		if tx.Error != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to begin transaction")
		}

		// Ensure we always cleanup.
		defer func() {
			if r := recover(); r != nil {
				_ = tx.Rollback()
				panic(r) // re-panic after rollback so Fiber's handler can catch
			}
			if err != nil {
				_ = tx.Rollback()
				return
			}
			if e := tx.Commit().Error; e != nil {
				log.Printf("tx commit failed: %v", e)
				err = fiber.NewError(fiber.StatusInternalServerError, "transaction commit failed")
			}
		}()

		// Pin the tenant schema for this TX only. SET LOCAL reverts at TX end.
		if e := tx.Exec(`SET LOCAL search_path = "` + schema + `", public`).Error; e != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "failed to set tenant schema")
		}

		// Make the TX available to handlers via GetTenantDB(c).
		c.Locals("tx", tx)

		// Run the handler chain inside this TX.
		err = c.Next()
		return err
	}
}
