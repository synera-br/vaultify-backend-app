// Package mailer provides functionality to send emails.
// Currently, it is configured to use Mailtrap (smtp.mailtrap.io) as the SMTP server,
// which is useful for development and testing environments.
//
// To use this package, you will need Mailtrap credentials.
// You can obtain these by signing up at https://mailtrap.io/ and finding the
// SMTP credentials for your inbox.
package mailer

import (
	"fmt"
	"net/smtp"
	"strings"
)

// SendEmail sends an email using Mailtrap's SMTP server.
//
// It requires valid Mailtrap credentials (username and password) to authenticate with the SMTP server.
// These credentials should be obtained from your Mailtrap account (see https://mailtrap.io/).
//
// Parameters:
//   recipient: The email address of the recipient (e.g., "user@example.com"). Cannot be empty.
//   sender:    The email address of the sender (e.g., "noreply@example.com"). Cannot be empty.
//              This address should typically be one that Mailtrap permits for your specific inbox.
//   subject:   The subject line of the email. Cannot be empty.
//   body:      The content of the email. This can be plain text or HTML.
//              The function attempts to infer the Content-Type based on basic HTML tags (<html>, <p>).
//   smtpUser:  The Mailtrap SMTP username. This is a REQUIRED field and must not be empty.
//              This is part of your Mailtrap inbox credentials.
//   smtpPass:  The Mailtrap SMTP password. This is a REQUIRED field and must not be empty.
//              This is part of your Mailtrap inbox credentials.
//
// Returns:
//   An error if any of the following occurs:
//     - Any of the required parameters (recipient, sender, subject, smtpUser, smtpPass) are empty.
//     - Connection to the SMTP server (smtp.mailtrap.io:2525) fails.
//     - SMTP authentication fails (e.g., incorrect smtpUser or smtpPass).
//     - The email sending command fails on the server.
//   If the email is sent successfully, it returns nil.
func SendEmail(recipient, sender, subject, body, smtpUser, smtpPass string) error {
	// SMTP server configuration
	smtpHost := "smtp.mailtrap.io"
	smtpPort := "2525"
	smtpAddr := smtpHost + ":" + smtpPort

	// Basic validation
	if recipient == "" {
		return fmt.Errorf("recipient email address cannot be empty")
	}
	if sender == "" {
		return fmt.Errorf("sender email address cannot be empty")
	}
	if subject == "" {
		return fmt.Errorf("email subject cannot be empty")
	}
	if smtpUser == "" || smtpPass == "" {
		return fmt.Errorf("SMTP username and password must be provided")
	}

	// Message construction
	// To send HTML mail, the Content-Type header must be set to text/html.
	// For plain text, it's text/plain. We'll try to infer based on simple body content.
	contentType := "text/plain; charset=UTF-8"
	if strings.Contains(strings.ToLower(body), "<html>") || strings.Contains(strings.ToLower(body), "<p>") {
		contentType = "text/html; charset=UTF-8"
	}

	message := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: %s\r\n"+
		"\r\n"+
		"%s\r\n", recipient, sender, subject, contentType, body))

	// Authentication
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	// Sending the email
	err := smtp.SendMail(smtpAddr, auth, sender, []string{recipient}, message)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// Example usage (can be removed or moved to a test file)
/*
func main() {
	// IMPORTANT: Replace with your actual Mailtrap credentials or load from env
	testUser := "your_mailtrap_username"
	testPass := "your_mailtrap_password"

	recipient := "recipient@example.com"
	sender := "sender@example.com" // Should be an address Mailtrap allows for your inbox
	subject := "Test Email from Go"
	htmlBody := "<h1>Hello!</h1><p>This is a <b>test email</b> sent from a Go application using Mailtrap.</p>"
	// plainTextBody := "Hello!\nThis is a test email sent from a Go application using Mailtrap."

	fmt.Printf("Sending email to %s...\n", recipient)
	err := SendEmail(recipient, sender, subject, htmlBody, testUser, testPass)
	if err != nil {
		fmt.Printf("Error sending email: %v\n", err)
		return
	}
	fmt.Println("Email sent successfully!")
}
*/
