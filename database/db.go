package database

import (
	"fakturierung-backend/models"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// GetTenantDB returns a new DB session with search_path set to the user's schema
func GetTenantDB(schema string) (*gorm.DB, error) {
	schema = strings.TrimSpace(schema)

	// You could also validate the schema name here (optional)
	if schema == "" {
		return nil, fmt.Errorf("empty schema name")
	}

	tenantDB := DB.Session(&gorm.Session{NewDB: true})
	if err := tenantDB.Exec("SET search_path TO " + schema).Error; err != nil {
		return nil, err
	}

	return tenantDB, nil
}

func MigrateTenantSchema(schema string) error {
	tenantDB, err := GetTenantDB(schema)
	if err != nil {
		return err
	}

	return tenantDB.AutoMigrate(
		&models.Supplier{}, &models.Article{}, &models.Customer{},
		&models.Invoice{}, &models.InvoiceItem{})
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
