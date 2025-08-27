package main

import (
	"fakturierung-backend/database"
	"fakturierung-backend/routes"
	"fmt"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	database.Connect()
	database.AutoMigrate()

	app := fiber.New()

	// CORS: allow origins from environment or default to all (for dev)
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "*"
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowCredentials: false, // no cookies needed (using token auth)
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Tenant-Schema",
	}))

	routes.Setup(app)

	// Listen on port from environment or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	app.Listen(":" + port)
	fmt.Println("🚀 API server started on port", port)
}
