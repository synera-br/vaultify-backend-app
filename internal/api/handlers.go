package api

import (
	"encoding/json"
	"log"
	"net/http"
	"github.com/example/myapp/internal/crypto" // Assuming go.mod module name is 'myapp'

	"github.com/gin-gonic/gin"
)

// EncryptedPayload represents the expected structure of an encrypted request body.
type EncryptedPayload struct {
	Payload string `json:"payload" binding:"required"`
}

// ExampleData represents the structure of the data after decryption.
// Adjust this according to your actual data structure.
type ExampleData struct {
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// HandleEncryptedData is an example handler for a protected route that decrypts a payload.
func HandleEncryptedData(encryptionKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "UserID not found in context"})
			return
		}

		var requestPayload EncryptedPayload
		if err := c.ShouldBindJSON(&requestPayload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
			return
		}

		// Decrypt the payload
		decryptedJSONString, err := crypto.Decrypt(requestPayload.Payload, encryptionKey)
		if err != nil {
			log.Printf("Error decrypting payload for user %s: %v", userID, err)
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Failed to decrypt payload"})
			return
		}

		// Log the decrypted JSON string for debugging (optional, be careful with sensitive data)
		// log.Printf("Decrypted JSON for user %s: %s", userID, decryptedJSONString)

		// Unmarshal the decrypted JSON string into your target struct
		var data ExampleData
		if err := json.Unmarshal([]byte(decryptedJSONString), &data); err != nil {
			log.Printf("Error unmarshalling decrypted JSON for user %s: %v. JSON: %s", userID, err, decryptedJSONString)
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Failed to parse decrypted data"})
			return
		}

		// At this point, 'data' contains the actual data sent by the client.
		// You can now process it.
		c.JSON(http.StatusOK, gin.H{
			"message":        "Payload successfully decrypted and processed",
			"userID":         userID,
			"decrypted_data": data,
		})
	}
}

// Placeholder for another handler
func GetUserDataHandler(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "UserID not found in context"})
		return
	}
	// In a real application, you would fetch user-specific data here
	c.JSON(http.StatusOK, gin.H{"message": "User data endpoint", "userID": userID})
}
