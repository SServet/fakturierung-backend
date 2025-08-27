package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"fakturierung-backend/database"
	"fakturierung-backend/middlewares"
	"fakturierung-backend/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// envInt reads an int env var with a default fallback.
func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func main() {
	// ---- Database (public)
	database.Connect()
	database.AutoMigrate()

	// ---- Limits (configurable via env)
	// Fiber default BodyLimit is 4 * 1024 * 1024 bytes if unset (per docs).
	// We allow overriding with BODY_LIMIT_BYTES or BODY_LIMIT_MB.
	bodyLimitBytes := envInt("BODY_LIMIT_BYTES", 0)
	if bodyLimitBytes <= 0 {
		bodyLimitBytes = envInt("BODY_LIMIT_MB", 4) * 1024 * 1024
	}

	// ---- Fiber app with global error handler + body limit
	app := fiber.New(fiber.Config{
		ErrorHandler: middlewares.ErrorHandler,
		BodyLimit:    bodyLimitBytes,
	})

	// ---- CORS
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "*"
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowCredentials: false, // using Bearer tokens, not cookies
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Tenant-Schema",
	}))

	// ---- Global rate limiter (applies to all routes; tune via env)
	rlMax := envInt("RATE_LIMIT_MAX", 60)                                            // default 60 reqs
	rlWindow := time.Duration(envInt("RATE_LIMIT_WINDOW_SECONDS", 60)) * time.Second // default 60s window
	app.Use(limiter.New(limiter.Config{
		Max:        rlMax,
		Expiration: rlWindow,
		// Default KeyGenerator = client IP; default 429 handler is fine.
		// See: https://docs.gofiber.io/api/middleware/limiter
	}))

	// ---- Routes
	routes.Register(app)

	// ---- Start
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := app.Listen(":" + port); err != nil {
		panic(err)
	}
	fmt.Println("API server started on port", port)
}
