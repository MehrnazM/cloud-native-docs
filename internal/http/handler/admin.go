package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (h *Handler) RegisterAdminRoutes(r *gin.RouterGroup, workerConnCheck func() bool) {

	r.GET("/health", func(c *gin.Context) {
		if !workerConnCheck() {
			c.JSON(503, ErrorResponse{
				Error: "Worker connection is not healthy",
				Code:  ErrCodeServiceUnavailable,
			})
			return
		}
		c.JSON(200, SuccessResponse{
			Message: "OK",
		})
	})

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

}
