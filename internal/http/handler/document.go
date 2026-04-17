package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/MehrnazM/cloud-native-docs/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RegisterDocumentRoutes(r *gin.RouterGroup, svc *service.DocumentService) {
	r.POST("/documents", CreateDocument(svc))
	r.GET("/documents/:id", GetDocumentByID(svc))
}

type CreateDocumentRequest struct {
	Name string `json:"name" binding:"required,min=1,max=255"`
}

func CreateDocument(svc *service.DocumentService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateDocumentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			slog.Error("failed to read and validate the body: ", "error", err)
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
		id, err := svc.CreateDocument(c.Request.Context(), req.Name)
		if err != nil {
			slog.Error("failed to create document: ", "error", err)
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

func GetDocumentByID(svc *service.DocumentService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		uid, err := uuid.Parse(id)
		if err != nil {
			slog.Error("invalid document id: ", "error", err)
			c.JSON(400, ErrorResponse{
				Error:   "invalid document id",
				Code:    ErrCodeValidation,
				Details: err.Error(),
			})
			return
		}
		doc, err := svc.GetDocumentByID(c.Request.Context(), uid)
		if err != nil {
			slog.Error("failed to get document: ", "error", err)
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
