// routes/routes.go
package routes

import (
	"go-ecommerce/controllers"
	"go-ecommerce/middleware"

	"github.com/gorilla/mux"
)

// RegisterRoutes sets up all the routes for the application
func RegisterRoutes(router *mux.Router, userController *controllers.UserController, productController *controllers.ProductController, cartController *controllers.CartController, orderController *controllers.OrderController) {
	// Public routes
	router.HandleFunc("/register", userController.Register).Methods("POST")
	router.HandleFunc("/login", userController.Login).Methods("POST")
	router.HandleFunc("/verify", userController.VerifyEmail).Methods("GET")

	// Protected routes
	protected := router.PathPrefix("/").Subrouter()
	protected.Use(middleware.AuthMiddleware)
	protected.HandleFunc("/profile", userController.GetProfile).Methods("GET")

	// Product routes
	router.HandleFunc("/products", productController.GetProducts).Methods("GET")
	router.HandleFunc("/products/{id}", productController.GetProductByID).Methods("GET")

	// Admin routes
	admin := router.PathPrefix("/products").Subrouter()
	admin.Use(middleware.AuthMiddleware)
	admin.Use(middleware.AdminMiddleware)
	admin.HandleFunc("", productController.CreateProduct).Methods("POST")
	admin.HandleFunc("/{id}", productController.UpdateProduct).Methods("PUT")
	admin.HandleFunc("/{id}", productController.DeleteProduct).Methods("DELETE")

	// Cart Routes
	router.HandleFunc("/cart", cartController.AddToCart).Methods("POST")
	router.HandleFunc("/cart", cartController.GetCart).Methods("GET")
	router.HandleFunc("/cart", cartController.RemoveFromCart).Methods("DELETE")

	//Order Routes
	router.HandleFunc("/orders", orderController.GetOrders).Methods("GET")
	router.HandleFunc("/order", orderController.CreateOrder).Methods("POST")
	router.HandleFunc("/order/{id}", orderController.UpdateOrderPaymentStatus).Methods("UPADTE")
}
