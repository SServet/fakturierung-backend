package main

import (
	"fakturierung-backend/database"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func main() {
	database.Connect()

	app := fiber.New()

	//routes.Setup(app)
	app.Listen(":8080")
	fmt.Println("🚀 started here we go")
}
