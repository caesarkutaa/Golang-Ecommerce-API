package controllers

import (
	"context"
	"encoding/json"
	"go-ecommerce/middleware"
	"go-ecommerce/models"
	"go-ecommerce/utils"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

// UserController handles user-related requests
type UserController struct {
	Collection   *mongo.Collection
	EmailService *utils.EmailService
}

// NewUserController creates a new UserController with EmailService
func NewUserController(client *mongo.Client, emailService *utils.EmailService) *UserController {
	collection := client.Database("ecommerce").Collection("users")
	return &UserController{
		Collection:   collection,
		EmailService: emailService,
	}
}

// Register handles user registration
func (uc *UserController) Register(w http.ResponseWriter, r *http.Request) {
	var user models.User
	// Decode the request body into user
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Check if user already exists
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	count, err := uc.Collection.CountDocuments(ctx, bson.M{"email": user.Email})
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if count > 0 {
		http.Error(w, "User already exists", http.StatusBadRequest)
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}
	user.Password = string(hashedPassword)
	user.Role = "user" // Default role
	user.IsVerified = false

	// Generate verification token
	verificationToken, err := utils.GenerateJWT(user.Email, user.Role)
	if err != nil {
		http.Error(w, "Error generating verification token", http.StatusInternalServerError)
		return
	}
	user.VerificationToken = verificationToken

	// Insert the user into the database
	_, err = uc.Collection.InsertOne(ctx, user)
	if err != nil {
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	// Send verification email
	err = uc.EmailService.SendVerificationEmail(user.Email, verificationToken)
	if err != nil {
		http.Error(w, "Error sending verification email", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode("User registered successfully. Please check your email to verify your account.")
}

// VerifyEmail handles email verification
func (uc *UserController) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Verification token missing", http.StatusBadRequest)
		return
	}

	claims := &utils.Claims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return utils.JwtKey, nil
	})

	if err != nil {
		http.Error(w, "Invalid token", http.StatusBadRequest)
		return
	}

	// Find the user with the verification token
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var user models.User
	err = uc.Collection.FindOne(ctx, bson.M{"verification_token": token}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found or already verified", http.StatusBadRequest)
		return
	}

	// Update the user's verification status
	_, err = uc.Collection.UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{
		"$set": bson.M{
			"is_verified":        true,
			"verification_token": "",
		},
	})
	if err != nil {
		http.Error(w, "Error updating user verification status", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode("Email verified successfully. You can now log in.")
}

// Login handles user authentication
func (uc *UserController) Login(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	// Decode the request body
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Find the user in the database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var user models.User
	err = uc.Collection.FindOne(ctx, bson.M{"email": creds.Email}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Check if email is verified
	if !user.IsVerified {
		http.Error(w, "Email not verified", http.StatusUnauthorized)
		return
	}

	// Compare the hashed password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(creds.Password))
	if err != nil {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token, err := utils.GenerateJWT(user.Email, user.Role)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	// Return the token
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// GetProfile retrieves the authenticated user's profile
func (uc *UserController) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Extract user information from context
	claims, ok := r.Context().Value(middleware.UserContextKey).(*utils.Claims)
	if !ok {
		http.Error(w, "Could not parse user from context", http.StatusUnauthorized)
		return
	}

	// Find the user in the database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var user models.User
	err := uc.Collection.FindOne(ctx, bson.M{"email": claims.Email}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Return the user profile (excluding password and verification token)
	user.Password = ""
	user.VerificationToken = ""
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
