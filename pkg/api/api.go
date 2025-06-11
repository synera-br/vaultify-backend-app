package api

import "github.com/gin-gonic/gin"

// API defines the interface for API services.
type API interface {
	RegisterRoutes(router *gin.Engine)
	Run(addr string) error
}
