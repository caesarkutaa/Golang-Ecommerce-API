// main.go
package main

import (
	"context"
	"fmt"
	"go-ecommerce/controllers"
	"go-ecommerce/middleware"
	"go-ecommerce/routes"
	"go-ecommerce/utils"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found. Proceeding with environment variables.")
	}

	// Set the JWT secret key
	utils.JwtKey = []byte(os.Getenv("JWT_SECRET"))

	// Initialize EmailService
	emailService := utils.NewEmailService()

	// Connect to MongoDB
	client := utils.ConnectDB()
	defer func() {
		if err = client.Disconnect(context.TODO()); err != nil {
			log.Fatal(err)
		}
	}()

	// Initialize controllers
	userController := controllers.NewUserController(client, emailService)
	productController := controllers.NewProductController(client)
	cartController := controllers.NewCartController(client)
	orderController := controllers.NewOrderController(client, emailService)
	// Set up the router
	router := mux.NewRouter()
	// Register routes
	routes.RegisterRoutes(router, userController, productController, cartController, orderController)

	// Apply middleware for authentication (optional)
	router.Use(middleware.AuthMiddleware)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	fmt.Printf("Server is running on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
