package ai

import (
	"net/http"

	"app/internal/response"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP endpoints for AI blog post generation.
type Handler struct {
	service Service
}

// NewHandler creates a new AI Handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes mounts AI generator routes onto the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	rg.POST("/posts/ai-generate", authMiddleware, h.GeneratePost)
}

func (h *Handler) GeneratePost(c *gin.Context) {
	var req GeneratePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	res, err := h.service.GeneratePost(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "OpenAI failed to generate AI blog post: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "AI blog draft generated successfully", res)
}
