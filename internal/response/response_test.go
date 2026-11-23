package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestResponses(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		Success(c, http.StatusOK, "ok", map[string]string{"key": "val"})

		assert.Equal(t, http.StatusOK, w.Code)
		var body APIResponse
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, "success", body.Status)
		assert.Equal(t, "ok", body.Message)
		assert.NotNil(t, body.Data)
	})

	t.Run("Fail", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		Fail(c, http.StatusBadRequest, "bad input")

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var body APIResponse
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, "fail", body.Status)
		assert.Equal(t, "bad input", body.Message)
		assert.Nil(t, body.Data)
	})

	t.Run("Error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		Error(c, http.StatusInternalServerError, "server error")

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		var body APIResponse
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, "error", body.Status)
		assert.Equal(t, "server error", body.Message)
	})

	t.Run("AbortFail", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		AbortFail(c, http.StatusUnauthorized, "unauthorized")

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var body APIResponse
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, "fail", body.Status)
	})

	t.Run("AbortError", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		AbortError(c, http.StatusServiceUnavailable, "unavailable")

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		var body APIResponse
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, "error", body.Status)
	})
}
