package middlewares

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"fakturierung-backend/database"
	"fakturierung-backend/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Idempotency processes Idempotency-Key for mutating HTTP methods in a schema-safe way.
// It uses its own short transaction and SET LOCAL search_path to avoid leaking search_path
// on pooled connections.
func Idempotency() fiber.Handler {
	return func(c *fiber.Ctx) error {
		method := strings.ToUpper(c.Method())
		if method != fiber.MethodPost && method != fiber.MethodPut && method != fiber.MethodPatch && method != fiber.MethodDelete {
			return c.Next()
		}

		key := strings.TrimSpace(c.Get("Idempotency-Key"))
		if key == "" {
			return c.Next()
		}
		if len(key) > 128 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Idempotency-Key too long"})
		}

		schema, _ := c.Locals("schema").(string)
		userID, _ := c.Locals("userID").(string)
		if schema == "" || userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "auth context missing"})
		}

		path := c.OriginalURL() // includes query string
		body := c.Body()

		// Build deterministic request hash: method|path|body|schema|user
		h := sha256.New()
		h.Write([]byte(method))
		h.Write([]byte{'\n'})
		h.Write([]byte(path))
		h.Write([]byte{'\n'})
		h.Write(body)
		h.Write([]byte{'\n'})
		h.Write([]byte(schema))
		h.Write([]byte{'\n'})
		h.Write([]byte(userID))
		reqHash := hex.EncodeToString(h.Sum(nil))

		// ---- Phase 1: read/create "pending" under a short TX with SET LOCAL search_path
		var existing models.IdempotencyKey
		err := database.DB.Transaction(func(tx *gorm.DB) error {
			// pin tenant for this short tx only
			if err := tx.Exec(`SET LOCAL search_path = "` + schema + `", public`).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "idempotency schema pin failed")
			}

			// Try to find existing key
			if err := tx.Where("key = ?", key).First(&existing).Error; err != nil {
				if err != gorm.ErrRecordNotFound {
					return fiber.NewError(fiber.StatusInternalServerError, "idempotency lookup failed")
				}
				// Not found -> create "pending"
				rec := models.IdempotencyKey{
					Key:            key,
					RequestHash:    reqHash,
					Method:         method,
					Path:           path,
					TenantSchema:   schema,
					UserID:         userID,
					ResponseStatus: 0,
				}
				if e2 := tx.Create(&rec).Error; e2 != nil {
					// Could be unique race: read again
					if e3 := tx.Where("key = ?", key).First(&existing).Error; e3 != nil {
						return fiber.NewError(fiber.StatusInternalServerError, "idempotency create failed")
					}
					// fall-through to existing handling below
				} else {
					existing = rec
				}
			}

			// Validate existing
			if existing.RequestHash != reqHash {
				return fiber.NewError(fiber.StatusConflict, "Idempotency-Key reuse with different request")
			}
			if existing.ResponseStatus != 0 && existing.ResponseBody != nil {
				// We've got a completed response stored — short-circuit (no handler run)
				c.Status(existing.ResponseStatus)
				return c.Send(existing.ResponseBody)
			}

			// Pending/in-progress: let the request run; other concurrent calls will see "pending"
			return nil
		})
		if err != nil {
			// If we already sent a stored response in the TX above, err is nil; otherwise return the error
			return err
		}

		// If we reached here, we need to run the handler once.
		if err := c.Next(); err != nil {
			return err
		}

		// ---- Phase 2: store the response under another short TX
		_ = database.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(`SET LOCAL search_path = "` + schema + `", public`).Error; err != nil {
				return nil // best-effort: don't break the successful response
			}
			now := time.Now().UTC()
			status := c.Response().StatusCode()
			resp := c.Response().Body()
			blob := make([]byte, len(resp))
			copy(blob, resp)

			return tx.Model(&models.IdempotencyKey{}).
				Where("key = ?", key).
				Updates(map[string]any{
					"response_status": status,
					"response_body":   blob,
					"completed_at":    &now,
				}).Error
		})

		return nil
	}
}
