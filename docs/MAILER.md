# Go Mailtrap Email Sending Library

This project provides a simple Go package for sending emails using Mailtrap's SMTP server. Mailtrap is a service commonly used for testing email functionality during development by capturing emails sent from your application in a virtual inbox.

## Features

*   Provides a `SendEmail` function to easily send emails.
*   Configured for Mailtrap's SMTP server (`smtp.mailtrap.io:2525`).
*   Supports HTML or plain text email bodies.
*   Includes a basic example (`main.go`) to demonstrate usage.

## Prerequisites

*   Go (version 1.18 or higher recommended).
*   A Mailtrap account and your SMTP credentials.

## Obtaining Mailtrap Credentials

1.  Sign up or log in to your Mailtrap account at [https://mailtrap.io/](https://mailtrap.io/).
2.  Navigate to your inbox (or create one).
3.  Under "SMTP Settings" (or a similar section), you will find your **Username** and **Password**. These are the `smtpUser` and `smtpPass` values you'll need.

## Setup and Dependencies

This project uses Go Modules for dependency management. While the current version only relies on standard library packages, it's good practice to ensure your module is up-to-date:

```bash
go mod tidy
```

This command will ensure that your `go.mod` and `go.sum` files are consistent with the project's dependencies.

## Usage

The core functionality is provided by the `SendEmail` function in the `pkg/mailer` package.

### Function Signature

```go
package mailer

// SendEmail(recipient, sender, subject, body, smtpUser, smtpPass string) error
```

### Parameters

*   `recipient` (string): The recipient's email address.
*   `sender` (string): The sender's email address (should be allowed by your Mailtrap inbox).
*   `subject` (string): The email subject.
*   `body` (string): The email body (can be plain text or HTML).
*   `smtpUser` (string): Your Mailtrap SMTP username.
*   `smtpPass` (string): Your Mailtrap SMTP password.

### Example

Here's how you can use the `SendEmail` function:

```go
package main

import (
	"app/pkg/mailer" // Assuming your module is 'app'
	"fmt"
	"log"
)

func main() {
	smtpUsername := "YOUR_MAILTRAP_USERNAME" // Replace with your actual Mailtrap username
	smtpPassword := "YOUR_MAILTRAP_PASSWORD" // Replace with your actual Mailtrap password

	senderEmail := "sender@example.com"
	recipientEmail := "recipient@example.com"
	emailSubject := "My Test Email"
	htmlEmailBody := "<h1>Hello!</h1><p>This is a test from the Go Mailtrap library.</p>"

	err := mailer.SendEmail(
		recipientEmail,
		senderEmail,
		emailSubject,
		htmlEmailBody,
		smtpUsername,
		smtpPassword,
	)

	if err != nil {
		log.Fatalf("Failed to send email: %v", err)
	}
	fmt.Println("Email sent successfully to Mailtrap!")
}
```
**(For this example to be runnable, ensure your project is initialized as a Go module, e.g., `go mod init myapp` at the project root if `app` is not your module name, and `app/pkg/mailer` matches your module path structure).**

## Running the Included Example (`main.go`)

The project includes an example file `main.go` in the root directory that demonstrates how to use the `mailer.SendEmail` function.

1.  **Configure Credentials:**
    Open `main.go` and replace the placeholder values for `smtpUser`, `smtpPass`, `senderEmail`, and `recipientEmail` with your actual Mailtrap credentials and desired email details.

    ```go
    // In main.go:
    smtpUser := "YOUR_MAILTRAP_USERNAME" // Replace this!
    smtpPass := "YOUR_MAILTRAP_PASSWORD" // Replace this!
    senderEmail := "from@example.com"      // Replace with your Mailtrap sender
    recipientEmail := "to@example.com"    // Replace with a test recipient
    ```

2.  **Run the example:**
    Navigate to the project root directory in your terminal and run:

    ```bash
    go run main.go
    ```

    If successful, you'll see a confirmation message, and the email will appear in your Mailtrap inbox. If there are errors (e.g., incorrect credentials), they will be printed to the console.

## Error Handling

The `SendEmail` function returns an `error` type. You should always check this error to handle potential issues like:
*   Invalid input parameters.
*   Network connection problems.
*   SMTP authentication failure (incorrect credentials).
*   Errors from the Mailtrap server.
```
