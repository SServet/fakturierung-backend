package routes

import (
	"fakturierung-backend/controllers"
	"fakturierung-backend/middlewares"

	"github.com/gofiber/fiber/v2"
)

func Setup(app *fiber.App) {
	api := app.Group("api")

	api.Post("registration", controllers.Register)
	api.Post("login", controllers.Login)
	api.Post("logout", controllers.Logout)

	authenticated := api.Use(middlewares.IsAuthenticatedHeader)

	//Customer Routes
	authenticated.Post("customer", controllers.CreateCustomer)
	authenticated.Put("customer", controllers.UpdateCustomer)
	authenticated.Get("customers", controllers.GetCustomers)
	authenticated.Get("customers/:id", controllers.GetCustomer)

	//Article Routes
	authenticated.Post("article", controllers.CreateArticles)
	authenticated.Put("article", controllers.UpdateArticle)

	//Invoice Routes
	authenticated.Post("invoice", controllers.CreateInvoice)
	authenticated.Put("inovice", controllers.UpdateInvoice)
	authenticated.Get("invoices", controllers.GetInvoices)
	authenticated.Get("invoices/:id", controllers.GetInvoice)
	authenticated.Put("invoices/publish/:id", controllers.PublishInvoice)
}
