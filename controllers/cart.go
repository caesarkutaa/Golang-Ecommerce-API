package controllers

import (
	"context"
	"encoding/json"
	"go-ecommerce/middleware"
	"go-ecommerce/models"
	"go-ecommerce/utils"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// CartController handles cart-related requests
type CartController struct {
	Collection *mongo.Collection
}

// NewCartController creates a new CartController
func NewCartController(client *mongo.Client) *CartController {
	collection := client.Database("ecommerce").Collection("carts")
	return &CartController{
		Collection: collection,
	}
}

// AddToCart adds a product to the user's cart
func (cc *CartController) AddToCart(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*utils.Claims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var item models.CartItem
	err := json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Find user ID from email
	userCollection := cc.Collection.Database().Collection("users")
	var user models.User
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = userCollection.FindOne(ctx, bson.M{"email": claims.Email}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Check if cart exists
	var cart models.Cart
	err = cc.Collection.FindOne(ctx, bson.M{"user_id": user.ID}).Decode(&cart)
	if err != nil {
		// Create new cart
		cart = models.Cart{
			UserID: user.ID,
			Items:  []models.CartItem{item},
		}
		_, err := cc.Collection.InsertOne(ctx, cart)
		if err != nil {
			http.Error(w, "Error creating cart", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode("Item added to cart")
		return
	}

	// Update existing cart
	updated := false
	for i, existingItem := range cart.Items {
		if existingItem.ProductID == item.ProductID {
			cart.Items[i].Quantity += item.Quantity
			updated = true
			break
		}
	}

	if !updated {
		cart.Items = append(cart.Items, item)
	}

	_, err = cc.Collection.UpdateOne(ctx, bson.M{"_id": cart.ID}, bson.M{"$set": bson.M{"items": cart.Items}})
	if err != nil {
		http.Error(w, "Error updating cart", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode("Item added to cart")
}

// RemoveFromCart removes a product from the user's cart
func (cc *CartController) RemoveFromCart(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*utils.Claims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	params := mux.Vars(r)
	productID, err := primitive.ObjectIDFromHex(params["product_id"])
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	// Find user ID from email
	userCollection := cc.Collection.Database().Collection("users")
	var user models.User
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = userCollection.FindOne(ctx, bson.M{"email": claims.Email}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Find cart
	var cart models.Cart
	err = cc.Collection.FindOne(ctx, bson.M{"user_id": user.ID}).Decode(&cart)
	if err != nil {
		http.Error(w, "Cart not found", http.StatusNotFound)
		return
	}

	// Remove the item
	updatedItems := []models.CartItem{}
	for _, item := range cart.Items {
		if item.ProductID != productID {
			updatedItems = append(updatedItems, item)
		}
	}

	_, err = cc.Collection.UpdateOne(ctx, bson.M{"_id": cart.ID}, bson.M{"$set": bson.M{"items": updatedItems}})
	if err != nil {
		http.Error(w, "Error updating cart", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode("Item removed from cart")
}

// GetCart retrieves the user's cart
func (cc *CartController) GetCart(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*utils.Claims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Find user ID from email
	userCollection := cc.Collection.Database().Collection("users")
	var user models.User
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := userCollection.FindOne(ctx, bson.M{"email": claims.Email}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Find cart
	var cart models.Cart
	err = cc.Collection.FindOne(ctx, bson.M{"user_id": user.ID}).Decode(&cart)
	if err != nil {
		http.Error(w, "Cart not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cart)
}
