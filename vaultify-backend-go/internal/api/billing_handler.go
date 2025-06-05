package api

import (
	"io/ioutil"
	"net/http"
	"errors"
	"log" // For logging

	"github.com/gin-gonic/gin"
	"vaultify-backend-go/internal/core"
	// "vaultify-backend-go/internal/config" // Would be needed for STRIPE_WEBHOOK_SECRET if verified here by handler
)

// BillingHandler handles billing-related API endpoints.
type BillingHandler struct {
	billingService core.BillingService
	// webhookSecret  string // If webhook signature verification were done in handler
}

// NewBillingHandler creates a new BillingHandler.
// In a real app, appConfig *config.Config would be passed to get webhookSecret if needed here.
func NewBillingHandler(bs core.BillingService /*, appConfig *config.Config */) *BillingHandler {
	return &BillingHandler{
		billingService: bs,
		// webhookSecret:  appConfig.StripeWebhookSecret, // If needed
	}
}

// --- Request DTOs ---

// CreateCheckoutSessionRequest defines the structure for creating a Stripe Checkout session.
type CreateCheckoutSessionRequest struct {
	PlanID  string `json:"planId" binding:"required"`
	PriceID string `json:"priceId" binding:"required"`
}

// --- Response DTOs ---

// CreateCheckoutSessionResponse returns the ID of the created Stripe Checkout session.
type CreateCheckoutSessionResponse struct {
	SessionID string `json:"sessionId"`
}

// CreatePortalSessionResponse returns the URL for the Stripe Customer Portal.
type CreatePortalSessionResponse struct {
	URL string `json:"url"`
}

// mapBillingErrorToStatus maps errors from core.BillingService to HTTP status codes and ErrorResponse.
func mapBillingErrorToStatus(c *gin.Context, err error) {
	var statusCode int
	var errResponse ErrorResponse

	switch {
	case errors.Is(err, core.ErrPlanNotFound):
		statusCode = http.StatusNotFound
		errResponse = ErrorResponse{Error: "Plan or Price not found", Details: err.Error()}
	case errors.Is(err, core.ErrStripeClient): // Generic Stripe client error
		statusCode = http.StatusServiceUnavailable // 503 suggests a problem with an upstream service (Stripe)
		errResponse = ErrorResponse{Error: "Payment provider error", Details: "Could not complete the operation with the payment provider."}
		log.Printf("Stripe Client Error: %v", err) // Log the original error for internal review
	case errors.Is(err, core.ErrWebhookSignature):
		statusCode = http.StatusBadRequest
		errResponse = ErrorResponse{Error: "Webhook signature verification failed"}
	case errors.Is(err, core.ErrWebhookProcessing):
		statusCode = http.StatusBadRequest // Or 500 if it's an internal processing issue after valid webhook
		errResponse = ErrorResponse{Error: "Webhook processing error", Details: err.Error()}
	case errors.Is(err, core.ErrUserStripeNotLinked):
		statusCode = http.StatusBadRequest // User needs to complete a purchase first or be linked
		errResponse = ErrorResponse{Error: "User not linked to payment provider", Details: err.Error()}
	// Add other specific errors from BillingService here
	default:
		log.Printf("Internal Server Error in BillingHandler: %v", err)
		statusCode = http.StatusInternalServerError
		errResponse = ErrorResponse{Error: "An unexpected internal server error occurred."}
	}
	c.JSON(statusCode, errResponse)
}

// CreateCheckoutSession handles POST /billing/create-checkout-session
func (h *BillingHandler) CreateCheckoutSession(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User ID not found in context"})
		return
	}

	var req CreateCheckoutSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request payload", Details: err.Error()})
		return
	}

	// Basic validation (more specific validation might be in service or using a validator library)
	if req.PlanID == "" || req.PriceID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "planId and priceId are required"})
		return
	}

	sessionID, err := h.billingService.CreateCheckoutSession(c.Request.Context(), userID.(string), req.PlanID, req.PriceID)
	if err != nil {
		mapBillingErrorToStatus(c, err)
		return
	}

	c.JSON(http.StatusOK, CreateCheckoutSessionResponse{SessionID: sessionID})
}

// CreatePortalSession handles POST /billing/create-portal-session
func (h *BillingHandler) CreatePortalSession(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User ID not found in context"})
		return
	}

	portalURL, err := h.billingService.CreatePortalSession(c.Request.Context(), userID.(string))
	if err != nil {
		mapBillingErrorToStatus(c, err)
		return
	}

	c.JSON(http.StatusOK, CreatePortalSessionResponse{URL: portalURL})
}

// HandleStripeWebhook handles POST /billing/webhooks/stripe
// This endpoint is public and does not require JWT authentication.
// Stripe authenticates webhooks using the 'Stripe-Signature' header.
func (h *BillingHandler) HandleStripeWebhook(c *gin.Context) {
	signature := c.GetHeader("Stripe-Signature")
	if signature == "" {
		log.Println("Stripe Webhook: Missing Stripe-Signature header.")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Missing Stripe-Signature header"})
		return
	}

	payload, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Stripe Webhook: Error reading request body: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to read webhook payload", Details: err.Error()})
		return
	}
	defer c.Request.Body.Close()

	// The billingService.HandleStripeWebhook is responsible for:
	// 1. Verifying the signature (using stripe.webhook.ConstructEvent and the webhook secret from config).
	// 2. Processing the event.
	err = h.billingService.HandleStripeWebhook(c.Request.Context(), signature, payload)
	if err != nil {
		// mapBillingErrorToStatus will handle signature errors, processing errors, etc.
		log.Printf("Stripe Webhook: Error handling webhook: %v", err)
		mapBillingErrorToStatus(c, err)
		return
	}

	// Stripe generally expects a 2xx response to acknowledge receipt of the webhook.
	// A 200 OK or 204 No Content are common.
	c.JSON(http.StatusOK, SuccessResponse{Message: "Webhook received successfully"})
	// Or c.Status(http.StatusNoContent) if no body is preferred for success.
}
