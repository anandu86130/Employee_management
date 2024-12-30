package main

import (
	"employee-management/internal/config"
	"employee-management/internal/handler"
	"log"
	"os"

	"github.com/labstack/echo/v4"
)

func main() {
	config.LoadEnv()
	db := config.DBConn()

	//Initialize Echo
	e := echo.New()

	//Set up HTTP handlers
	handler.NewUserHandler(e, db)

	port := os.Getenv("PORT")
	if port == "" {
		port = ":8080"
	}
	log.Printf("Starting server on port %s", port)
	if err := e.Start(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
