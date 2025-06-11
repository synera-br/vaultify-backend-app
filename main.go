package main

import (
	"app/pkg/mailer" // Assuming module name is 'app'
	"fmt"
	"log"
)

func main() {
	// --- Configuration - Replace with your actual details or load from environment ---
	// IMPORTANT: Fill these in with your Mailtrap credentials and desired email details.
	// It's recommended to use environment variables for sensitive data like passwords.
	smtpUser := "YOUR_MAILTRAP_USERNAME" // Replace with your Mailtrap username
	smtpPass := "YOUR_MAILTRAP_PASSWORD" // Replace with your Mailtrap password

	senderEmail := "sender@example.com"       // Replace with a sender email address configured in your Mailtrap inbox
	recipientEmail := "recipient@example.com" // Replace with the recipient's email address

	emailSubject := "Test Email from main.go"
	htmlBody := `
<html>
  <body>
    <h1>Hello from main.go!</h1>
    <p>This is a test email sent using the mailer package.</p>
    <p>If you see this, it means the <code>SendEmail</code> function in <code>pkg/mailer/mailer.go</code> is working (at least in terms of sending to Mailtrap).</p>
  </body>
</html>`
	// plainTextBody := "Hello from main.go!\nThis is a test email sent using the mailer package."

	// --- Basic check for placeholder credentials ---
	if smtpUser == "YOUR_MAILTRAP_USERNAME" || smtpPass == "YOUR_MAILTRAP_PASSWORD" {
		log.Println("WARNING: Mailtrap username or password are set to default placeholders.")
		log.Println("Please update them in main.go with your actual Mailtrap credentials to send a real test email.")
		// You might want to exit here if you don't want to proceed with placeholder credentials
		// return
	}

	// --- Send the email ---
	fmt.Printf("Attempting to send email to %s via Mailtrap...\n", recipientEmail)
	err := mailer.SendEmail(recipientEmail, senderEmail, emailSubject, htmlBody, smtpUser, smtpPass)

	if err != nil {
		log.Fatalf("Error sending email: %v", err)
	}

	fmt.Println("Email sending process initiated successfully!")
	fmt.Println("Please check your Mailtrap inbox to see if the email was delivered.")
}
