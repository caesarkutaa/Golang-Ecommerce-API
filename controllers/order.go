// controllers/order.go
package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"go-ecommerce/middleware"
	"go-ecommerce/models"
	"go-ecommerce/utils"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// OrderController handles order-related requests
type OrderController struct {
	OrderCollection   *mongo.Collection
	CartCollection    *mongo.Collection
	ProductCollection *mongo.Collection
	UserCollection    *mongo.Collection
	EmailService      *utils.EmailService
}

// NewOrderController creates a new OrderController
func NewOrderController(client *mongo.Client, emailService *utils.EmailService) *OrderController {
	orderCollection := client.Database("ecommerce").Collection("orders")
	cartCollection := client.Database("ecommerce").Collection("carts")
	productCollection := client.Database("ecommerce").Collection("products")
	userCollection := client.Database("ecommerce").Collection("users")
	return &OrderController{
		OrderCollection:   orderCollection,
		CartCollection:    cartCollection,
		ProductCollection: productCollection,
		UserCollection:    userCollection,
		EmailService:      emailService,
	}
}

// CreateOrder creates a new order from the user's cart
func (oc *OrderController) CreateOrder(w http.ResponseWriter, r *http.Request) {
	// Extract JWT claims from the request context
	claims, ok := r.Context().Value(middleware.UserContextKey).(*utils.Claims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Find the user in the database
	var user models.User
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := oc.UserCollection.FindOne(ctx, bson.M{"email": claims.Email}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Find the user's cart
	var cart models.Cart
	err = oc.CartCollection.FindOne(ctx, bson.M{"user_id": user.ID}).Decode(&cart)
	if err != nil {
		http.Error(w, "Cart not found", http.StatusNotFound)
		return
	}

	if len(cart.Items) == 0 {
		http.Error(w, "Cart is empty", http.StatusBadRequest)
		return
	}

	// Parse payment method from request
	// Expecting JSON body with "payment_method": "card" or "crypto"
	var paymentRequest struct {
		PaymentMethod string `json:"payment_method"`
	}
	err = json.NewDecoder(r.Body).Decode(&paymentRequest)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	paymentMethod := strings.ToLower(paymentRequest.PaymentMethod)
	if paymentMethod != "card" && paymentMethod != "crypto" {
		http.Error(w, "Invalid payment method", http.StatusBadRequest)
		return
	}

	// Calculate total amount and check stock
	totalAmount := 0.0
	for _, item := range cart.Items {
		var product models.Product
		err := oc.ProductCollection.FindOne(ctx, bson.M{"_id": item.ProductID}).Decode(&product)
		if err != nil {
			http.Error(w, fmt.Sprintf("Product with ID %s not found", item.ProductID.Hex()), http.StatusNotFound)
			return
		}
		if product.Stock < item.Quantity {
			http.Error(w, fmt.Sprintf("Insufficient stock for product: %s", product.Name), http.StatusBadRequest)
			return
		}
		totalAmount += product.Price * float64(item.Quantity)
	}

	// Deduct stock for each product
	for _, item := range cart.Items {
		_, err := oc.ProductCollection.UpdateOne(ctx, bson.M{"_id": item.ProductID}, bson.M{
			"$inc": bson.M{"stock": -item.Quantity},
		})
		if err != nil {
			http.Error(w, "Failed to update product stock", http.StatusInternalServerError)
			return
		}
	}

	// Set delivery date to 7 working days from now
	deliveryDate := time.Now().AddDate(0, 0, 10) // Approximation: 7 working days ~10 calendar days

	// Create the order
	order := models.Order{
		UserID:        user.ID,
		Items:         cart.Items,
		TotalAmount:   totalAmount,
		DeliveryDate:  deliveryDate.Format("2006-01-02"),
		PaymentMethod: paymentMethod,
	}

	// Insert the order into the database
	orderResult, err := oc.OrderCollection.InsertOne(ctx, order)
	if err != nil {
		http.Error(w, "Failed to create order", http.StatusInternalServerError)
		return
	}

	// Handle crypto payment proof upload
	if paymentMethod == "crypto" {
		// Create a unique directory for the user if it doesn't exist
		uploadPath := filepath.Join("uploads", "payments", user.ID.Hex())
		err := os.MkdirAll(uploadPath, os.ModePerm)
		if err != nil {
			http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
			return
		}

		// Parse multipart form with a max memory of 10MB
		err = r.ParseMultipartForm(10 << 20)
		if err != nil {
			http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
			return
		}

		// Retrieve the file from form data
		file, handler, err := r.FormFile("crypto_proof")
		if err != nil {
			http.Error(w, "Failed to retrieve file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Create a unique filename
		filename := fmt.Sprintf("%s_%s", orderResult.InsertedID.(primitive.ObjectID).Hex(), handler.Filename)
		filePath := filepath.Join(uploadPath, filename)

		// Create the file on the server
		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Failed to create file on server", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		// Copy the uploaded file to the server
		_, err = io.Copy(dst, file)
		if err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

		// Update the order with the crypto proof path
		_, err = oc.OrderCollection.UpdateOne(ctx, bson.M{"_id": orderResult.InsertedID}, bson.M{
			"$set": bson.M{"crypto_proof": filePath},
		})
		if err != nil {
			http.Error(w, "Failed to update order with crypto proof", http.StatusInternalServerError)
			return
		}

		// Send notification email to user
		go func(email string) {
			subject := "Crypto Payment Received - E-commerce Platform"
			content := fmt.Sprintf("Dear %s,\n\nWe have received your cryptocurrency payment. Please upload the proof of payment to complete your order. Your order will be processed once the payment is verified.\n\nThank you for shopping with us!\n", user.Name)
			err := oc.EmailService.SendEmail(email, subject, content)
			if err != nil {
				log.Printf("Failed to send email to %s: %v", email, err)
			}
		}(user.Email)
	} else if paymentMethod == "card" {
		// For card payments, integrate with a payment gateway here
		// For simplicity, we'll assume the payment is successful
		_, err := oc.OrderCollection.UpdateOne(ctx, bson.M{"_id": orderResult.InsertedID}, bson.M{
			"$set": bson.M{"payment_status": "completed"},
		})
		if err != nil {
			http.Error(w, "Failed to update payment status", http.StatusInternalServerError)
			return
		}

		// Send confirmation email to user
		go func(email string) {
			subject := "Order Confirmation - E-commerce Platform"
			content := fmt.Sprintf("Dear %s,\n\nThank you for your purchase! Your order has been placed successfully and will be delivered by %s.\n\nTotal Amount: $%.2f\nPayment Method: %s\n\nThank you for shopping with us!\n", user.Name, deliveryDate.Format("2006-01-02"), totalAmount, paymentMethod)
			err := oc.EmailService.SendEmail(email, subject, content)
			if err != nil {
				log.Printf("Failed to send email to %s: %v", email, err)
			}
		}(user.Email)
	}

	// Clear the user's cart
	_, err = oc.CartCollection.DeleteOne(ctx, bson.M{"user_id": user.ID})
	if err != nil {
		http.Error(w, "Failed to clear cart", http.StatusInternalServerError)
		return
	}

	// Respond with the created order details
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"order_id":      orderResult.InsertedID,
		"total_amount":  totalAmount,
		"delivery_date": deliveryDate.Format("2006-01-02"),
		"message":       "Order created successfully. It will take 7 working days to arrive at your provided address.",
	})
}

// GetOrders retrieves all orders for the authenticated user
func (oc *OrderController) GetOrders(w http.ResponseWriter, r *http.Request) {
	// Extract JWT claims from the request context
	claims, ok := r.Context().Value(middleware.UserContextKey).(*utils.Claims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Find the user in the database
	var user models.User
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := oc.UserCollection.FindOne(ctx, bson.M{"email": claims.Email}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Find all orders for the user
	cursor, err := oc.OrderCollection.Find(ctx, bson.M{"user_id": user.ID})
	if err != nil {
		http.Error(w, "Failed to retrieve orders", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var orders []models.Order
	for cursor.Next(ctx) {
		var order models.Order
		err := cursor.Decode(&order)
		if err != nil {
			http.Error(w, "Error decoding order", http.StatusInternalServerError)
			return
		}
		orders = append(orders, order)
	}

	if err := cursor.Err(); err != nil {
		http.Error(w, "Cursor error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

// UpdateOrderPaymentStatus allows admin to update payment status
func (oc *OrderController) UpdateOrderPaymentStatus(w http.ResponseWriter, r *http.Request) {
	// Only admins should be able to update payment status
	claims, ok := r.Context().Value(middleware.UserContextKey).(*utils.Claims)
	if !ok || claims.Role != "admin" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	orderIDHex := vars["id"]
	orderID, err := primitive.ObjectIDFromHex(orderIDHex)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	// Parse the request body for new payment status
	var paymentUpdate struct {
		PaymentStatus string `json:"payment_status"` // "completed", "failed"
	}
	err = json.NewDecoder(r.Body).Decode(&paymentUpdate)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if paymentUpdate.PaymentStatus != "completed" && paymentUpdate.PaymentStatus != "failed" {
		http.Error(w, "Invalid payment status", http.StatusBadRequest)
		return
	}

	// Update the payment status in the order
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	update := bson.M{
		"$set": bson.M{
			"payment_status": paymentUpdate.PaymentStatus,
		},
	}
	result, err := oc.OrderCollection.UpdateOne(ctx, bson.M{"_id": orderID}, update)
	if err != nil {
		http.Error(w, "Failed to update payment status", http.StatusInternalServerError)
		return
	}

	if result.MatchedCount == 0 {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	// Optionally, send an email notification to the user about the payment status update
	var order models.Order
	err = oc.OrderCollection.FindOne(ctx, bson.M{"_id": orderID}).Decode(&order)
	if err != nil {
		http.Error(w, "Failed to retrieve updated order", http.StatusInternalServerError)
		return
	}

	var user models.User
	err = oc.UserCollection.FindOne(ctx, bson.M{"_id": order.UserID}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	subject := "Payment Status Updated - E-commerce Platform"
	content := fmt.Sprintf("Dear %s,\n\nYour order (ID: %s) payment status has been updated to '%s'.\n\nThank you for shopping with us!\n", user.Name, orderID.Hex(), paymentUpdate.PaymentStatus)
	err = oc.EmailService.SendEmail(user.Email, subject, content)
	if err != nil {
		log.Printf("Failed to send email to %s: %v", user.Email, err)
	}

	// Respond with success message
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Payment status updated successfully"})
}
