package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Address represents a user's address for delivery
type Address struct {
	Street  string `bson:"street" json:"street"`
	City    string `bson:"city" json:"city"`
	State   string `bson:"state" json:"state"`
	ZipCode string `bson:"zipcode" json:"zipcode"`
}

// User represents a user in the system
type User struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name              string             `bson:"name" json:"name"`
	Email             string             `bson:"email" json:"email"`
	Password          string             `bson:"password,omitempty" json:"-"`
	Address           Address            `bson:"address" json:"address"`
	Role              string             `bson:"role" json:"role"` // "user" or "admin"
	IsVerified        bool               `bson:"is_verified" json:"is_verified"`
	VerificationToken string             `bson:"verification_token" json:"-"`
}
