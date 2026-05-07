package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/MehrnazM/cloud-native-docs/internal/messaging"
	"github.com/MehrnazM/cloud-native-docs/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type Handler struct {
	logger  *slog.Logger
	docSvc  *service.DocumentService
	authSvc *service.AuthService
	tracer  trace.Tracer
	conn    *messaging.Connection
}

func NewHandler(docSvc *service.DocumentService, authSvc *service.AuthService, logger *slog.Logger, tracerName string, conn *messaging.Connection) *Handler {
	return &Handler{
		docSvc:  docSvc,
		authSvc: authSvc,
		logger:  logger,
		tracer:  otel.Tracer(tracerName),
		conn:    conn,
	}
}

func (h *Handler) RegisterDocumentRoutes(r *gin.RouterGroup) {
	r.POST("/documents", h.CreateDocument())
	r.GET("/documents/:id", h.GetDocumentByID())
}

type CreateDocumentRequest struct {
	Name string `json:"name" binding:"required,min=1,max=255"`
}

func (h *Handler) CreateDocument() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, span := h.tracer.Start(c.Request.Context(), "POST /documents")
		defer span.End()

		var req CreateDocumentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			h.logger.Error("failed to read and validate the body: ", "error", err)
			c.JSON(400, ErrorResponse{
				Error:   "failed to read and validate the body",
				Code:    ErrCodeValidation,
				Details: err.Error(),
			})
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			c.JSON(400, ErrorResponse{
				Error: "name cannot be empty",
				Code:  ErrCodeValidation,
			})
			return
		}
		id, err := h.docSvc.CreateDocument(ctx, req.Name, c.GetString("requestID"))
		if err != nil {
			h.logger.Error("failed to create document: ", "error", err)
			c.JSON(500, ErrorResponse{
				Error: "failed to create document",
				Code:  ErrCodeInternal,
			})
			return
		}

		c.JSON(http.StatusCreated, SuccessResponse{
			Message: "Document created successfully",
			Data:    gin.H{"id": id},
		})
	}
}

func (h *Handler) GetDocumentByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, span := h.tracer.Start(c.Request.Context(), "GET /documents/:id")
		defer span.End()

		id := c.Param("id")
		uid, err := uuid.Parse(id)
		if err != nil {
			h.logger.Error("invalid document id: ", "error", err)
			c.JSON(400, ErrorResponse{
				Error:   "invalid document id",
				Code:    ErrCodeValidation,
				Details: err.Error(),
			})
			return
		}
		doc, err := h.docSvc.GetDocumentByID(ctx, uid)
		if err != nil {
			h.logger.Error("failed to get document: ", "error", err)
			c.JSON(500, ErrorResponse{
				Error: "failed to get document",
				Code:  ErrCodeInternal,
			})
			return
		}
		if doc == nil {
			c.JSON(404, ErrorResponse{
				Error: "document not found",
				Code:  ErrCodeNotFound,
			})
			return
		}
		c.JSON(http.StatusOK, SuccessResponse{
			Message: "Document retrieved successfully",
			Data:    doc,
		})
	}
}
