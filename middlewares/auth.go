package middlewares

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/joho/godotenv"
)

func loadSecretKey() string {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	return os.Getenv("JWT_SECRET_KEY")
}

type Claims struct {
	SchemaName string `json:"schema"`
	jwt.RegisteredClaims
}

func GenerateJWT(userID, schemaName string) (string, error) {
	claims := Claims{
		SchemaName: schemaName,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(loadSecretKey()))
}

func IsAuthenticatedHeader(c *fiber.Ctx) error {
	auth := c.Get("Authorization")
	if auth == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Missing Authorization header"})
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid Authorization header"})
	}
	tokenStr := parts[1]

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(loadSecretKey()), nil
	})
	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid or expired token"})
	}

	claims := token.Claims.(*Claims)
	// make schema and user ID available to handlers
	c.Locals("schema", claims.SchemaName)
	c.Locals("userID", claims.Subject)

	return c.Next()
}

func GetUserId(c *fiber.Ctx) (string, error) {
	cookie := c.Cookies("jwt")

	token, err := jwt.ParseWithClaims(cookie, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(loadSecretKey()), nil
	})

	if err != nil {
		return "", err
	}

	payload := token.Claims.(*Claims)

	id := payload.Subject

	return id, nil

}

/*
func GetSchema(c *fiber.Ctx) (string, error) {
	cookie := c.Cookies("jwt")

	token, err := jwt.ParseWithClaims(cookie, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(loadSecretKey()), nil
	})

	if err != nil {
		return "", err
	}

	payload := token.Claims.(*Claims)

	schema := payload.Schema
	return schema, nil
}
*/
