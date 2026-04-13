package handler

import "github.com/gin-gonic/gin"

func RegisterAdminRoutes(r *gin.RouterGroup, workerConnCheck func() bool) {

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

}
