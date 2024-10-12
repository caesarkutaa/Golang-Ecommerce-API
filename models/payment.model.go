package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Payment represents a payment for an order
type Payment struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	OrderID       primitive.ObjectID `bson:"order_id" json:"order_id"`
	PaymentMethod string             `bson:"payment_method" json:"payment_method"` // "card" or "crypto"
	Amount        float64            `bson:"amount" json:"amount"`
	Status        string             `bson:"status" json:"status"`                           // "Pending", "Completed"
	ProofURL      string             `bson:"proof_url,omitempty" json:"proof_url,omitempty"` // For crypto payments
}
