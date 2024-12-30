package config

import (
	"employee-management/internal/model"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("failed to load .env file")
	}
}

func DBConn() *gorm.DB {
	db, err := gorm.Open(postgres.Open(os.Getenv("PSQL_URL")))
	if err != nil {
		log.Fatal(err)
		return nil
	}
	fmt.Println("Connected to postgres")
	if err := db.AutoMigrate(&model.Employee{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}
