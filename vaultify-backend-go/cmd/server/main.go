package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings" // Added for strings.ToLower
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	// "go.uber.org/zap/zapcore" // Optional: For deeper log level config if not using NewDevelopment/Production

	"vaultify-backend-go/internal/api"
	"vaultify-backend-go/internal/config"
	"vaultify-backend-go/internal/core"
	"vaultify-backend-go/internal/db"
	"vaultify-backend-go/internal/middleware"
)

func main() {
	// --- 1. Initialize Logger (Zap) ---
	// Using NewDevelopment for more verbose, human-readable output during development.
	// For production, consider zap.NewProduction() or a custom configuration.
	// zap.NewDevelopmentConfig().Build() is also a good choice for dev.
	zapLogger, err := zap.NewDevelopment() // Simpler dev logger
	if err != nil {
		log.Fatalf("CRITICAL_ERROR: Failed to initialize Zap logger: %v", err)
	}
	defer zapLogger.Sync() // Flushes buffer, if any. IMPORTANT for buffered loggers.
	zapLogger.Info("Zap logger initialized successfully.")

	// --- 2. Load Application Configuration ---
	appConfig, err := config.LoadConfig()
	if err != nil {
		zapLogger.Fatal("CRITICAL_ERROR: Failed to load application configuration", zap.Error(err))
	}
	zapLogger.Info("Application configuration loaded successfully.")

	// --- 3. Initialize Firebase Admin SDK (includes Firestore and Auth clients) ---
	initCtx, cancelInitCtx := context.WithTimeout(context.Background(), 15*time.Second) // Timeout for initialization
	defer cancelInitCtx()
	if err := db.InitFirestore(initCtx, appConfig); err != nil {
		zapLogger.Fatal("CRITICAL_ERROR: Failed to initialize Firestore and Firebase Admin SDK", zap.Error(err))
	}
	zapLogger.Info("Firebase Admin SDK (Firestore, Auth) initialized successfully.")

	// --- 4. Retrieve initialized clients ---
	firestoreClient := db.GetFirestoreClient()
	firebaseAuthClient := db.GetFirebaseAuthClient() // Needed for AuthMiddleware

	// Ensure clients are not nil (critical for application function)
	if firestoreClient == nil {
		zapLogger.Fatal("CRITICAL_ERROR: Firestore client is nil after initialization. Application cannot start.")
	}
	if firebaseAuthClient == nil {
		zapLogger.Fatal("CRITICAL_ERROR: Firebase Auth client is nil after initialization. Application cannot start.")
	}
	zapLogger.Info("Firestore and Firebase Auth clients retrieved successfully.")

	// --- 5. Initialize Repositories ---
	userRepo := db.NewFirestoreUserRepository(firestoreClient)
	auditRepo := db.NewFirestoreAuditRepository(firestoreClient)
	vaultRepo := db.NewFirestoreVaultRepository(firestoreClient)
	secretRepo := db.NewFirestoreSecretRepository(firestoreClient)
	zapLogger.Info("Repositories initialized successfully.")

	// --- 6. Initialize Services ---
	auditService := core.NewAuditService(auditRepo)
	userService := core.NewUserService(userRepo)
	encryptionService := core.NewEncryptionService() // Wrapper, no direct repo/client deps

	secretService, err := core.NewSecretService(secretRepo, vaultRepo, userRepo, encryptionService, auditService, appConfig)
	if err != nil {
		zapLogger.Fatal("CRITICAL_ERROR: Failed to initialize SecretService", zap.Error(err))
	}

	// VaultService might need appConfig for plan limits in the future if not hardcoded
	vaultService := core.NewVaultService(vaultRepo, secretRepo, userRepo, auditService /*, appConfig (if needed) */)

	// Initialize BillingService (currently a placeholder)
	// Actual dependencies (e.g., userRepo, appConfig for Stripe keys, actual Stripe client) would be passed here.
	billingService := core.NewBillingService( /* userRepo, appConfig */ )
	zapLogger.Info("Core services initialized successfully.")

	// --- 7. Setup Gin HTTP Engine ---
	if strings.ToLower(appConfig.GinMode) == "release" {
		gin.SetMode(gin.ReleaseMode)
		zapLogger.Info("Gin mode set to 'release'.")
	} else {
		gin.SetMode(gin.DebugMode) // Default or "debug"
		zapLogger.Info("Gin mode set to 'debug'.")
	}
	// Using gin.New() to have control over the middleware stack (e.g., not using gin.DefaultLogger if using custom Zap logger).
	router := gin.New()
	zapLogger.Info("Gin engine created.")

	// --- 8. Apply Global Middleware (Order is important) ---
	router.Use(middleware.RequestLogger(zapLogger))     // Log every request; should be early.
	router.Use(middleware.RecoveryMiddleware(zapLogger)) // Recover from panics; should be after logger, before other handlers.

	// Apply CORS middleware only if ClientURL is configured, otherwise log a warning.
	if appConfig.ClientURL != "" {
		router.Use(middleware.CORSMiddleware(appConfig))
		zapLogger.Info("CORS Middleware enabled", zap.String("clientURL", appConfig.ClientURL))
	} else {
		zapLogger.Warn("CORS Middleware SKIPPED: CLIENT_URL is not configured. API might not be accessible from a web frontend.")
	}

	// --- 9. Setup API Routes ---
	// The firebaseAuthClient is retrieved within SetupRoutes via db.GetFirebaseAuthClient(),
	// but it's confirmed available above.
	api.SetupRoutes(
		router,
		appConfig,
		zapLogger, // Pass logger for route setup logging or specific handler needs
		userService,
		vaultService,
		secretService,
		billingService,
	)
	// SetupRoutes logs its own success message.

	// --- 10. Configure and Start HTTP Server ---
	serverAddr := fmt.Sprintf(":%s", appConfig.Port)
	httpServer := &http.Server{
		Addr:    serverAddr,
		Handler: router,
		// Example: Set some timeouts for hardening the server.
		// ReadTimeout:  15 * time.Second,
		// WriteTimeout: 15 * time.Second,
		// IdleTimeout:  60 * time.Second,
	}

	zapLogger.Info("Starting HTTP server...", zap.String("address", serverAddr), zap.String("ginMode", gin.Mode()))

	// Goroutine for starting the server, so it doesn't block graceful shutdown logic.
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zapLogger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// --- 11. Graceful Shutdown Handling ---
	// Create a channel to listen for OS signals (SIGINT, SIGTERM).
	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received on the quitChannel.
	sig := <-quitChannel
	zapLogger.Info("Received shutdown signal", zap.String("signal", sig.String()))

	// Create a context with a timeout for the shutdown process.
	// This gives active connections time to finish before the server is forced to close.
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second) // Increased timeout slightly
	defer cancelShutdown()

	// Attempt to gracefully shut down the HTTP server.
	zapLogger.Info("Attempting graceful shutdown of HTTP server...")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		zapLogger.Fatal("Server forced to shutdown due to error during graceful shutdown", zap.Error(err))
	}

	zapLogger.Info("Server exiting gracefully.")
}
