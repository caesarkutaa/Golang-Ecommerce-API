package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Order represents a user's order
type Order struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID        primitive.ObjectID `bson:"user_id" json:"user_id"`
	Items         []CartItem         `bson:"items" json:"items"`
	TotalAmount   float64            `bson:"total_amount" json:"total_amount"`
	Address       Address            `bson:"address" json:"address"`
	PaymentMethod string             `bson:"payment_method" json:"payment_method"`
	Status        string             `bson:"status" json:"status"` // e.g., "Pending", "Shipped"
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	DeliveryDate  string             `bson:"delivery_date" json:"delivery_date"` // e.g., "7 working days"
}
