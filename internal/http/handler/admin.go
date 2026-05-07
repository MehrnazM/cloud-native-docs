package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/MehrnazM/cloud-native-docs/shared/events"
	"github.com/MehrnazM/cloud-native-docs/shared/util"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (h *Handler) RegisterAdminRoutes(r, protected *gin.RouterGroup, workerConnCheck func() bool) {

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
	protected.GET("/admin/dlq", h.GetDLQMessages())

}

func (h *Handler) GetDLQMessages() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, span := h.tracer.Start(c.Request.Context(), "GET /admin/dlq")
		defer span.End()

		limit, err := strconv.Atoi(c.DefaultQuery("limit", util.GetStringEnv("DLQ_LIMIT", "50")))
		if err != nil || limit < 1 || limit > 1000 {
			limit = 50
		}

		consumer, err := h.conn.JS.CreateOrUpdateConsumer(ctx, "DOCUMENT_DLQ", jetstream.ConsumerConfig{
			DeliverPolicy: jetstream.DeliverAllPolicy,
			AckPolicy:     jetstream.AckNonePolicy,
			FilterSubject: "documents.dlq",
		})
		if err != nil {
			h.logger.Error("failed to create DLQ consumer", "error", err)
			c.JSON(500, ErrorResponse{Error: "failed to read DLQ", Code: ErrCodeInternal})
			return
		}

		msgBatch, err := consumer.Fetch(limit, jetstream.FetchMaxWait(500*time.Millisecond))
		if err != nil {
			h.logger.Error("failed to fetch DLQ messages", "error", err)
			c.JSON(500, ErrorResponse{Error: "failed to read DLQ", Code: ErrCodeInternal})
			return
		}

		var dlqEvents []events.DLQEvent
		for msg := range msgBatch.Messages() {
			var dlqEvent events.DLQEvent
			if err := json.Unmarshal(msg.Data(), &dlqEvent); err != nil {
				h.logger.Warn("failed to unmarshal DLQ message", "error", err)
				continue
			}
			dlqEvents = append(dlqEvents, dlqEvent)
		}
		if dlqEvents == nil {
			dlqEvents = []events.DLQEvent{}
		}

		c.JSON(http.StatusOK, SuccessResponse{
			Message: "DLQ messages retrieved successfully",
			Data:    gin.H{"messages": dlqEvents, "count": len(dlqEvents)},
		})
	}
}
