package http

import (
	"log/slog"
	"net/http"

	"github.com/MehrnazM/cloud-native-docs/internal/http/handler"
	"github.com/MehrnazM/cloud-native-docs/internal/http/middleware"
	"github.com/MehrnazM/cloud-native-docs/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func NewRouter(docSvc *service.DocumentService, authSvc *service.AuthService, workerConnCheck func() bool,
	logger *slog.Logger, tracerName, jwtSecret string) *gin.Engine {
	router := gin.Default()

	h := handler.NewHandler(docSvc, authSvc, logger, tracerName)

	v1 := router.Group("/api/v1")
	v1.Use(requestID())

	protected := v1.Group("/")
	protected.Use((middleware.AuthMiddleware([]byte(jwtSecret))))
	h.RegisterDocumentRoutes(protected)
	h.RegisterAuthRoutes(v1.Group("/auth"))
	h.RegisterAdminRoutes(v1, workerConnCheck)

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
