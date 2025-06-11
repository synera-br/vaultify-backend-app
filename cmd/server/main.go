package main

import (
	"context" // Added for fbApp.Auth
	"log"
	"net/http"
	"os"
	"github.com/example/myapp/internal/crypto"
	"github.com/example/myapp/internal/firebase"
	"github.com/example/myapp/internal/middleware"
	"github.com/example/myapp/internal/api" // Import for API routes

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file. In production, environment variables should be set directly.
	if os.Getenv("GIN_MODE") != "release" {
		err := godotenv.Load()
		if err != nil {
			log.Println("Warning: Error loading .env file:", err)
		}
	}

	// Initialize Firebase Admin SDK
	fbApp, err := firebase.InitFirebase()
	if err != nil {
		log.Fatalf("Error initializing Firebase Admin SDK: %v", err)
	}

	// Get Firebase Auth client
	fbAuth, err := fbApp.Auth(context.Background())
	if err != nil {
		log.Fatalf("Error getting Firebase Auth client: %v", err)
	}

	// Retrieve and validate encryption key
	// This is done at startup to ensure the key is valid.
	encryptionKey, err := crypto.GetEncryptionKeyFromEnv()
	if err != nil {
		log.Fatalf("Error with encryption key: %v", err)
	}

	// Set Gin mode
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Initialize Gin router
	router := gin.New() // Using gin.New() to have more control over middleware

	// Apply global middleware
	router.Use(middleware.LoggingMiddleware())
	router.Use(middleware.RecoveryMiddleware())
	router.Use(middleware.CORSMiddleware()) // Apply CORS globally

	// Ping endpoint for health check (does not require auth)
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// API v1 route group
	// All routes under /api/v1 will be subject to authentication by default,
	// unless specific routes are configured otherwise.
	apiV1 := router.Group("/api/v1")
	apiV1.Use(middleware.AuthMiddleware(fbAuth)) // Protect all /api/v1 routes

	// Register API routes from internal/api
	api.RegisterRoutes(apiV1, encryptionKey) // Pass the encryption key

	// Old placeholder removed:
	// apiV1.GET("/hello", func(c *gin.Context) {
	// 	userID, exists := c.Get("userID")
	// 	if !exists {
	// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "UserID not found in context"})
	// 		return
	// 	}
	// 	c.JSON(http.StatusOK, gin.H{"message": "Hello, authenticated user!", "userID": userID})
	// })


	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
	}
	log.Printf("Server starting on port %s in %s mode...", port, gin.Mode())
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
