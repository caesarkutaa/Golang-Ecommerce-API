package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CartItem represents an item in the cart
type CartItem struct {
	ProductID primitive.ObjectID `bson:"product_id" json:"product_id"`
	Quantity  int                `bson:"quantity" json:"quantity"`
}

// Cart represents a user's shopping cart
type Cart struct {
	ID     primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID primitive.ObjectID `bson:"user_id" json:"user_id"`
	Items  []CartItem         `bson:"items" json:"items"`
}
