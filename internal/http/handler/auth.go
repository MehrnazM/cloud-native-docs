package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/MehrnazM/cloud-native-docs/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterAuthRoutes(r *gin.RouterGroup) {
	r.POST("/register", h.Register())
	r.POST("/login", h.Login())
}

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Register() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, span := h.tracer.Start(c.Request.Context(), "POST /register")
		defer span.End()

		var req AuthRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			h.logger.Error("failed to read and validate the body: ", "error", err)
			c.JSON(400, ErrorResponse{
				Error:   "failed to read and validate the body",
				Code:    ErrCodeValidation,
				Details: err.Error(),
			})
			return
		}
		req.Email = strings.TrimSpace(req.Email)
		if req.Email == "" {
			c.JSON(400, ErrorResponse{
				Error: "name cannot be empty",
				Code:  ErrCodeValidation,
			})
			return
		}
		req.Password = strings.TrimSpace(req.Password)
		if req.Password == "" {
			c.JSON(400, ErrorResponse{
				Error: "password cannot be empty",
				Code:  ErrCodeValidation,
			})
			return
		}

		err := h.authSvc.Register(ctx, req.Email, req.Password)
		if err != nil {
			h.logger.Error("failed to register user: ", "error", err)
			if errors.Is(err, service.ErrEmailTaken) {
				c.JSON(http.StatusConflict, ErrorResponse{
					Error: service.ErrEmailTaken.Error(),
					Code:  ErrCodeConflict,
				})
				return
			}

			c.JSON(500, ErrorResponse{
				Error: "failed to register user",
				Code:  ErrCodeInternal,
			})
			return
		}

		c.JSON(http.StatusCreated, SuccessResponse{
			Message: "user registered successfully",
		})
	}
}

func (h *Handler) Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, span := h.tracer.Start(c.Request.Context(), "POST /login")
		defer span.End()

		var req AuthRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			h.logger.Error("failed to read and validate the body: ", "error", err)
			c.JSON(400, ErrorResponse{
				Error:   "failed to read and validate the body",
				Code:    ErrCodeValidation,
				Details: err.Error(),
			})
			return
		}
		req.Email = strings.TrimSpace(req.Email)
		if req.Email == "" {
			c.JSON(400, ErrorResponse{
				Error: "name cannot be empty",
				Code:  ErrCodeValidation,
			})
			return
		}
		req.Password = strings.TrimSpace(req.Password)
		if req.Password == "" {
			c.JSON(400, ErrorResponse{
				Error: "password cannot be empty",
				Code:  ErrCodeValidation,
			})
			return
		}

		token, err := h.authSvc.Login(ctx, req.Email, req.Password)
		if err != nil {
			h.logger.Error("failed to login user: ", "error", err)
			if errors.Is(err, service.ErrInvalidCredentials) {
				c.JSON(http.StatusUnauthorized, ErrorResponse{
					Error: service.ErrInvalidCredentials.Error(),
					Code:  ErrCodeUnauthorized,
				})
				return
			}
			c.JSON(500, ErrorResponse{
				Error: "failed to login user",
				Code:  ErrCodeInternal,
			})
			return
		}

		c.JSON(http.StatusOK, SuccessResponse{
			Message: "user logged in successfully",
			Data:    gin.H{"token": token},
		})
	}
}
