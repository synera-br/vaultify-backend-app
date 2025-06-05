package api

import (
	"net/http" // For http.StatusOK in health check

	"github.com/gin-gonic/gin"
	"go.uber.org/zap" // For logger in middleware and function params

	"vaultify-backend-go/internal/config"
	"vaultify-backend-go/internal/core"
	"vaultify-backend-go/internal/db" // For db.GetFirebaseAuthClient()
	"vaultify-backend-go/internal/middleware"
	// "firebase.google.com/go/v4/auth" // Type not needed directly if using db.GetFirebaseAuthClient()
)

// SetupRoutes configures all the application routes with their handlers and middleware.
// It's expected that global middleware (Logging, Recovery, CORS) are applied to the `router`
// instance *before* this function is called, typically in `main.go`.
func SetupRoutes(
	router *gin.Engine, // The Gin engine instance
	appConfig *config.Config,
	logger *zap.Logger, // Logger for any route setup logging or passed to specific needs
	userService core.UserService,
	vaultService core.VaultService,
	secretService core.SecretService,
	billingService core.BillingService, // Assumes placeholder or real service
) {
	// --- Initialize Middleware requiring dependencies ---
	// Get Firebase Auth client. This must be available after db.InitFirestore().
	firebaseAuthClient := db.GetFirebaseAuthClient()
	if firebaseAuthClient == nil {
		// This is a critical failure. The application cannot secure routes.
		// Log and panic to prevent starting with a misconfiguration.
		logger.Fatal("CRITICAL_SETUP_ERROR: Firebase Auth client is not initialized. AuthMiddleware cannot be created, and routes will not be set up.")
		// The panic below will stop execution if logger.Fatal doesn't exit.
		panic("Firebase Auth client is nil during route setup. Ensure db.InitFirestore() was called and succeeded.")
	}
	authMW := middleware.NewAuthMiddleware(firebaseAuthClient)

	// --- Initialize Handlers ---
	authHandler := NewAuthHandler(userService)
	userHandler := NewUserHandler(userService)
	vaultHandler := NewVaultHandler(vaultService)
	secretHandler := NewSecretHandler(secretService)
	billingHandler := NewBillingHandler(billingService)

	// --- Define API Route Groups ---
	// Base group for API version 1
	apiV1 := router.Group("/api/v1")
	{
		// --- User and Authentication Endpoints ---
		userAuthGroup := apiV1.Group("/users")
		{
			// POST /api/v1/users/initialize - Requires auth to identify the user.
			// Called after client-side Firebase login/signup to ensure backend profile exists.
			userAuthGroup.POST("/initialize", authMW.VerifyToken(), authHandler.InitializeUserProfile)

			// GET /api/v1/users/me - Requires auth to get current user's profile.
			userAuthGroup.GET("/me", authMW.VerifyToken(), userHandler.GetCurrentUserProfile)
		}

		// --- Vault Endpoints ---
		// All vault operations require authentication.
		vaultsRouteGroup := apiV1.Group("/vaults", authMW.VerifyToken())
		{
			vaultsRouteGroup.POST("", vaultHandler.CreateVault)
			vaultsRouteGroup.GET("", vaultHandler.ListVaults) // Lists vaults for the authenticated user
			vaultsRouteGroup.GET("/:vaultId", vaultHandler.GetVault)
			vaultsRouteGroup.PUT("/:vaultId", vaultHandler.UpdateVault)
			vaultsRouteGroup.DELETE("/:vaultId", vaultHandler.DeleteVault)

			// Vault Sharing Endpoints (nested under a specific vault)
			sharingRouteGroup := vaultsRouteGroup.Group("/:vaultId/share")
			{
				// POST /api/v1/vaults/{vaultId}/share
				sharingRouteGroup.POST("", vaultHandler.ShareVault)
				// PUT /api/v1/vaults/{vaultId}/share/{targetUserId}
				sharingRouteGroup.PUT("/:targetUserId", vaultHandler.UpdateShare)
				// DELETE /api/v1/vaults/{vaultId}/share/{targetUserId}
				sharingRouteGroup.DELETE("/:targetUserId", vaultHandler.RemoveShare)
			}
		}

		// --- Secret Endpoints ---
		// All secret operations are nested under a vault and require authentication.
		// The vault access (ownership/share) is checked within the SecretService methods.
		secretsRouteGroup := apiV1.Group("/vaults/:vaultId/secrets", authMW.VerifyToken())
		{
			secretsRouteGroup.POST("", secretHandler.CreateSecret)
			secretsRouteGroup.GET("", secretHandler.ListSecrets)
			secretsRouteGroup.GET("/:secretId", secretHandler.GetSecret)
			secretsRouteGroup.PUT("/:secretId", secretHandler.UpdateSecret)
			secretsRouteGroup.DELETE("/:secretId", secretHandler.DeleteSecret)
		}

		// --- Billing Endpoints ---
		billingRouteGroup := apiV1.Group("/billing")
		{
			// Authenticated endpoints for user-initiated billing actions
			billingRouteGroup.POST("/create-checkout-session", authMW.VerifyToken(), billingHandler.CreateCheckoutSession)
			billingRouteGroup.POST("/create-portal-session", authMW.VerifyToken(), billingHandler.CreatePortalSession)

			// Public webhook endpoint for Stripe (NO authMW.VerifyToken() middleware here)
			// Stripe authenticates webhooks via signature, handled by the service.
			billingRouteGroup.POST("/webhooks/stripe", billingHandler.HandleStripeWebhook)
		}
	}

	// --- General Health Check Endpoint ---
	// This endpoint is typically public and does not go under /api/v1 group.
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP", "message": "Vaultify backend is healthy."})
	})

	logger.Info("API routes configured successfully under /api/v1 and /health.")
}
