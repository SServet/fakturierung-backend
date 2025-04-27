package main

import (
	"fakturierung-backend/database"
	"fakturierung-backend/routes"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	database.Connect()
	database.AutoMigrate()

	app := fiber.New()

	// 1) CORS: allow only your React origin and the headers you'll send
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000",
		AllowCredentials: false, // no cookies
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Tenant-Schema",
	}))

	routes.Setup(app)
	app.Listen(":8080")
	fmt.Println("🚀 started here we go")
}
