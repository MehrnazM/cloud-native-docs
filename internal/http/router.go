package http

import (
	"net/http"

	"github.com/MehrnazM/cloud-native-docs/internal/http/handler"
	"github.com/MehrnazM/cloud-native-docs/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func NewRouter(svc *service.DocumentService, workerConnCheck func() bool) *gin.Engine {
	router := gin.Default()
	v1 := router.Group("/api/v1")
	v1.Use(requestID())
	handler.RegisterDocumentRoutes(v1, svc)
	handler.RegisterAdminRoutes(v1, workerConnCheck)
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "route not found",
		})
	})
	return router
}

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("requestID", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}
