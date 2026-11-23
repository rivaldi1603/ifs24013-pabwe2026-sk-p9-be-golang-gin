package response

import "github.com/gin-gonic/gin"

// APIResponse represents the standardized JSON response structure.
type APIResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success sends a JSON success response with optional data payload.
func Success(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(code, APIResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

// Fail sends a JSON fail response for client errors (4xx).
func Fail(c *gin.Context, code int, message string) {
	c.JSON(code, APIResponse{
		Status:  "fail",
		Message: message,
	})
}

// Error sends a JSON error response for server errors (5xx).
func Error(c *gin.Context, code int, message string) {
	c.JSON(code, APIResponse{
		Status:  "error",
		Message: message,
	})
}

// AbortFail aborts the Gin context chain with a JSON fail response.
func AbortFail(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, APIResponse{
		Status:  "fail",
		Message: message,
	})
}

// AbortError aborts the Gin context chain with a JSON error response.
func AbortError(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, APIResponse{
		Status:  "error",
		Message: message,
	})
}
