package middlewares

import (
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

const (
	authHeader   = "Authorization"
	bearerPrefix = "Bearer "
)

// Claims is our custom JWT payload (subject=userID, plus tenant schema).
type Claims struct {
	Schema string `json:"schema"`
	jwt.RegisteredClaims
}

var (
	secretOnce sync.Once
	jwtSecret  []byte
	secretErr  error
)

func loadJWTSecret() error {
	secretOnce.Do(func() {
		// Prefer JWT_SECRET_KEY, fallback to JWT_SECRET
		sec := os.Getenv("JWT_SECRET_KEY")
		if strings.TrimSpace(sec) == "" {
			sec = os.Getenv("JWT_SECRET")
		}
		if strings.TrimSpace(sec) == "" {
			secretErr = errors.New("JWT secret not configured (set JWT_SECRET_KEY or JWT_SECRET)")
			return
		}
		jwtSecret = []byte(sec)
	})
	return secretErr
}

// IsAuthenticatedHeader validates a Bearer token, enforces HS256, and populates c.Locals("userID","schema").
func IsAuthenticatedHeader() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := loadJWTSecret(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "server auth not configured",
			})
		}

		h := c.Get(authHeader)
		if h == "" || !strings.HasPrefix(strings.ToLower(h), strings.ToLower(bearerPrefix)) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "missing/invalid Authorization header"})
		}
		raw := strings.TrimSpace(h[len(bearerPrefix):])
		if raw == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "invalid bearer token"})
		}

		parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		var claims Claims
		token, err := parser.ParseWithClaims(raw, &claims, func(t *jwt.Token) (interface{}, error) {
			// Parser already restricts to HS256; this is just defense-in-depth.
			if t.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "invalid or expired token"})
		}
		// Basic payload checks
		if strings.TrimSpace(claims.Subject) == "" || strings.TrimSpace(claims.Schema) == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "token missing subject/schema"})
		}

		// Stash tenant context for the request
		c.Locals("userID", claims.Subject)
		c.Locals("schema", claims.Schema)

		return c.Next()
	}
}

// GenerateJWT signs a new HS256 token for the given user & schema, expiring in 24h.
func GenerateJWT(userID, schema string) (string, error) {
	if err := loadJWTSecret(); err != nil {
		return "", err
	}
	now := time.Now()
	claims := &Claims{
		Schema: schema,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			// (Optional) set Issuer/Audience here if you want stricter validation
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
