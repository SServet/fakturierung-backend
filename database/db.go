package database

import (
	"fakturierung-backend/models"
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func MigrateTenantSchema(c *fiber.Ctx) error {
	tenantDB, err := GetTenantDB(c)
	if err != nil {
		return err
	}

	return tenantDB.AutoMigrate(
		&models.Supplier{}, &models.Article{}, &models.Customer{},
		&models.Invoice{},
		&models.InvoiceItem{},
		&models.InvoiceVersion{}, // NEW
		&models.Payment{})
}

func Connect() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dsn := fmt.Sprintf("host=db user=%s password=%s dbname=%s port=5432 sslmode=disable TimeZone=Asia/Shanghai",
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		fmt.Println("I am here")
		fmt.Println(err)
		panic("Could not connect to database")
	}
}

func AutoMigrate() {
	DB.AutoMigrate(models.ContactPerson{}, models.Company{}, models.User{})
}
