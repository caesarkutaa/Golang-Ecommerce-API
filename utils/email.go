// utils/email.go
package utils

import (
	"fmt"
	"go-ecommerce/models"
	"os"

	"github.com/keighl/postmark"
)

// EmailService handles sending emails using Postmark
type EmailService struct {
	client *postmark.Client
}

// NewEmailService initializes and returns a new EmailService instance
func NewEmailService() *EmailService {
	apiToken := os.Getenv("POSTMARK_API_TOKEN")
	if apiToken == "" {
		panic("POSTMARK_API_TOKEN is not set in environment variables")
	}
	client := postmark.NewClient(apiToken, "") // Include a valid sender if needed
	return &EmailService{
		client: client,
	}
}

// SendEmail sends a basic email to the specified recipient
func (es *EmailService) SendEmail(toEmail, subject, htmlContent string) error {
	_, err := es.client.SendEmail(postmark.Email{
		From:     os.Getenv("EMAIL_SENDER"),
		To:       toEmail,
		Subject:  subject,
		HtmlBody: htmlContent, // Use the provided HTML content
		TextBody: htmlContent, // Optional: You can provide plain text if needed
	})

	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Println("Email sent successfully!")
	return nil
}

// SendVerificationEmail sends an email verification link to the user
func (es *EmailService) SendVerificationEmail(toEmail, token string) error {
	subject := "Verify Your Email"
	verificationLink := fmt.Sprintf("http://localhost:%s/verify-email?token=%s", os.Getenv("PORT"), token)
	htmlContent := fmt.Sprintf(
		"<strong>Please verify your email by clicking on the following link:</strong> <a href=\"%s\">Verify Email</a>",
		verificationLink,
	)

	return es.SendEmail(toEmail, subject, htmlContent) // Ensure htmlContent is used
}

// SendOrderConfirmationEmail sends an order confirmation email to the user
func (es *EmailService) SendOrderConfirmationEmail(toEmail string, order models.Order) error {
	subject := "Order Confirmation"
	htmlContent := fmt.Sprintf(
		"<strong>Dear Customer,</strong><br><br>Thank you for your purchase! Your order (ID: %s) has been placed successfully and will be delivered by <strong>%s</strong>.<br><br>Total Amount: <strong>$%.2f</strong><br>Payment Method: <strong>%s</strong><br><br>Thank you for shopping with us!",
		order.ID.Hex(),
		order.DeliveryDate,
		order.TotalAmount,
		order.PaymentMethod,
	)

	return es.SendEmail(toEmail, subject, htmlContent) // Ensure htmlContent is used
}
