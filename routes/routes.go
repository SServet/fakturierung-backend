package routes

import (
	"github.com/gofiber/fiber/v2"

	"fakturierung-backend/controllers"
	"fakturierung-backend/middlewares"
)

// Register wires all HTTP routes.
func Register(app *fiber.App) {
	api := app.Group("/api")

	// Public auth endpoints
	api.Post("/registration", controllers.Register)
	api.Post("/login", controllers.Login)
	api.Post("/logout", controllers.Logout)

	// Protected endpoints (JWT auth)
	protected := api.Group("")
	protected.Use(middlewares.IsAuthenticatedHeader())

	// Idempotency guard FIRST (not tied to request TX)
	protected.Use(middlewares.Idempotency())

	// Then per-request tenant transaction (pins search_path and commits/rolls back)
	protected.Use(middlewares.TenantTx())

	// Customers
	protected.Post("/customer", controllers.CreateCustomer)
	protected.Get("/customers", controllers.GetCustomers)
	protected.Get("/customer/:id", controllers.GetCustomer)
	protected.Put("/customer/:id", controllers.UpdateCustomer)

	// Suppliers
	protected.Post("/supplier", controllers.CreateSupplier)
	protected.Put("/supplier/:id", controllers.UpdateSupplier)

	// Articles
	protected.Post("/article", controllers.CreateArticles) // batch create
	protected.Get("/articles", controllers.GetArticles)
	protected.Put("/articles/:id", controllers.UpdateArticle)

	// Invoices (versioned model with payments)
	protected.Post("/invoice", controllers.CreateInvoice)
	protected.Get("/invoices", controllers.GetInvoices)
	protected.Get("/invoice/:id", controllers.GetInvoice)
	protected.Put("/invoices/:id", controllers.UpdateInvoice)
	protected.Put("/invoices/:id/convert", controllers.ConvertInvoice)
	protected.Put("/invoices/:id/publish", controllers.PublishInvoice)
	protected.Get("/invoices/:id/versions", controllers.GetInvoiceVersions)
	protected.Post("/invoices/:id/payments", controllers.CreatePayment)
	protected.Get("/invoices/:id/payments", controllers.ListPayments)
}
