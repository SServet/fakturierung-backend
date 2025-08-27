package routes

import (
	"github.com/gofiber/fiber/v2"

	"fakturierung-backend/controllers"
	"fakturierung-backend/middlewares"
)

func Register(app *fiber.App) {
	api := app.Group("/api")

	// Public auth endpoints
	api.Post("/registration", controllers.Register)
	api.Post("/login", controllers.Login)
	api.Post("/logout", controllers.Logout)

	// Protected endpoints
	protected := api.Group("")
	protected.Use(middlewares.IsAuthenticatedHeader)

	// Customers
	protected.Post("/customer", controllers.CreateCustomer)
	protected.Put("/customer/:id", controllers.UpdateCustomer) // << changed to :id
	protected.Get("/customers", controllers.GetCustomers)

	// Articles
	protected.Post("/article", controllers.CreateArticles)
	protected.Put("/articles/:id", controllers.UpdateArticle) // << changed to :id
	protected.Get("/articles", controllers.GetArticles)

	// Invoices (versioned)
	protected.Post("/invoice", controllers.CreateInvoice)
	protected.Put("/invoices/:id", controllers.UpdateInvoice)
	protected.Get("/invoices", controllers.GetInvoices)
	protected.Get("/invoice/:id", controllers.GetInvoice)

	protected.Put("/invoices/:id/convert", controllers.ConvertInvoice)
	protected.Put("/invoices/:id/publish", controllers.PublishInvoice)

	protected.Get("/invoices/:id/versions", controllers.GetInvoiceVersions)

	protected.Post("/invoices/:id/payments", controllers.CreatePayment)
	protected.Get("/invoices/:id/payments", controllers.ListPayments)

}
