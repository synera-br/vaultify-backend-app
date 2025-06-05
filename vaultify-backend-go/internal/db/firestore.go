package db

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os" // For checking env var like GOOGLE_APPLICATION_CREDENTIALS directly if needed, though config is preferred

	firebase "firebase.google.com/go/v4"
         "firebase.google.com/go/v4/auth"
         "cloud.google.com/go/firestore" // Correct import for the Firestore client type
	"google.golang.org/api/option"

	"vaultify-backend-go/internal/config" // To access config for Firebase Project ID, credentials
)

var (
	// fsClient is the global Firestore client instance.
	fsClient *firestore.Client
	// fbAuthClient is the global Firebase Auth client instance.
	fbAuthClient *auth.Client
)

// InitFirestore initializes the Firebase Admin SDK and sets up the Firestore client.
// It uses credentials and project ID from the provided appConfig.
func InitFirestore(ctx context.Context, appConfig *config.Config) error {
	if appConfig == nil {
		return fmt.Errorf("InitFirestore: appConfig cannot be nil")
	}

	var credsOption option.ClientOption
	var firebaseAppConfig *firebase.Config

	// Determine Firebase credentials option
	if appConfig.GoogleApplicationCredentials != "" {
		// Option 1: Path to service account file
		// Ensure the file path is accessible in the environment where the app runs.
		// Note: Viper might have already loaded this into FirebaseServiceAccountJSONBase64
		// if that was the ultimate source. Here we assume GoogleApplicationCredentials means a file path.
		log.Printf("Initializing Firebase with credentials file: %s", appConfig.GoogleApplicationCredentials)
		// Check if the file exists, otherwise option.WithCredentialsFile might not error clearly immediately
		if _, err := os.Stat(appConfig.GoogleApplicationCredentials); os.IsNotExist(err) {
			log.Printf("Warning: Credentials file specified in GOOGLE_APPLICATION_CREDENTIALS does not exist: %s", appConfig.GoogleApplicationCredentials)
			// Depending on strictness, could return error here.
			// Firebase SDK might still work if ADC are set up in the environment independently.
		}
		credsOption = option.WithCredentialsFile(appConfig.GoogleApplicationCredentials)
	} else if appConfig.FirebaseServiceAccountJSONBase64 != "" {
		// Option 2: Base64 encoded service account JSON
		log.Println("Initializing Firebase with Base64 encoded service account JSON.")
		decodedJSON, err := base64.StdEncoding.DecodeString(appConfig.FirebaseServiceAccountJSONBase64)
		if err != nil {
			log.Printf("Error decoding FirebaseServiceAccountJSONBase64: %v", err)
			return fmt.Errorf("failed to decode FirebaseServiceAccountJSONBase64: %w", err)
		}
		credsOption = option.WithCredentialsJSON(decodedJSON)
	} else {
		// Option 3: Rely on Application Default Credentials (ADC)
		// This is common for GCP environments (GCE, GKE, Cloud Run, Cloud Functions).
		// No explicit credsOption is needed if ADC is correctly set up.
		log.Println("Initializing Firebase using Application Default Credentials (ADC).")
		// credsOption will be nil, and firebase.NewApp will attempt to use ADC.
	}

	// Set ProjectID if provided in config and not inferred from credentials
	// (credentials usually contain ProjectID, making this redundant but harmless).
	if appConfig.FirebaseProjectID != "" {
		firebaseAppConfig = &firebase.Config{
			ProjectID: appConfig.FirebaseProjectID,
		}
		log.Printf("Firebase Project ID set from config: %s", appConfig.FirebaseProjectID)
	} else {
		log.Println("Firebase Project ID not explicitly set in config, relying on credentials or ADC.")
	}

	var app *firebase.App
	var err error

	if credsOption != nil {
		app, err = firebase.NewApp(ctx, firebaseAppConfig, credsOption)
	} else {
		// For ADC, credsOption is nil.
		// If firebaseAppConfig is also nil (no explicit ProjectID), NewApp tries full ADC.
		// If firebaseAppConfig has ProjectID, it's used along with ADC for credentials.
		app, err = firebase.NewApp(ctx, firebaseAppConfig)
	}

	if err != nil {
		log.Printf("Error initializing Firebase app: %v", err)
		return fmt.Errorf("firebase.NewApp: %w", err)
	}

	// Get Firestore client
	client, err := app.Firestore(ctx)
	if err != nil {
		log.Printf("Error getting Firestore client: %v", err)
		// It's useful to try and clean up the app if subsequent steps fail, though often not critical for client setup.
		// client.Close() is for the firestore client, not app.
		return fmt.Errorf("app.Firestore: %w", err)
	}
	fsClient = client
	log.Println("Firestore client initialized successfully.")

	// Optionally, initialize Firebase Auth client
	authCl, err := app.Auth(ctx)
	if err != nil {
		log.Printf("Error getting Firebase Auth client: %v", err)
		// Close Firestore client if Auth client fails, as Init is considered failed.
		if fsClient != nil {
			fsClient.Close() // Best effort close
		}
		return fmt.Errorf("app.Auth: %w", err)
	}
	fbAuthClient = authCl
	log.Println("Firebase Auth client initialized successfully.")

	return nil
}

// GetFirestoreClient returns the global Firestore client.
// Callers should check if the client is nil, implying InitFirestore hasn't been called or failed.
func GetFirestoreClient() *firestore.Client {
	if fsClient == nil {
		log.Println("Warning: GetFirestoreClient called before InitFirestore or InitFirestore failed.")
	}
	return fsClient
}

// GetFirebaseAuthClient returns the global Firebase Auth client.
// Callers should check if the client is nil.
func GetFirebaseAuthClient() *auth.Client {
	if fbAuthClient == nil {
		log.Println("Warning: GetFirebaseAuthClient called before InitFirestore or InitFirestore failed.")
	}
	return fbAuthClient
}
