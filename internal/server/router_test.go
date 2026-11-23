package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/config"
	"app/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupRouter(t *testing.T) {
	cfg := &config.Config{
		AppEnv:               "production",
		AppKey:               "test-secret-key",
		DBConnection:         "sqlite",
		DBDatabase:           ":memory:",
		SearchEngineEndpoint: "https://delcom.org/api/v1/search",
		OpenAIEndpoint:       "https://delcom.org/api/v1",
	}
	db, err := database.Connect(cfg)
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))

	r := SetupRouter(cfg, db)
	gin.SetMode(gin.TestMode)

	// /api.json with http and https header
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api.json", nil)
	req.Host = "localhost:8081"
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Delcom Blog REST API")

	wTLS := httptest.NewRecorder()
	reqTLS := httptest.NewRequest(http.MethodGet, "/api.json", nil)
	reqTLS.Header.Set("X-Forwarded-Proto", "https")
	r.ServeHTTP(wTLS, reqTLS)
	assert.Equal(t, http.StatusOK, wTLS.Code)
	assert.Contains(t, wTLS.Body.String(), `"https"`)

	// Swagger UI
	wSwag := httptest.NewRecorder()
	r.ServeHTTP(wSwag, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	assert.Equal(t, http.StatusOK, wSwag.Code)

	// Public categories & posts endpoints mounted
	wCat := httptest.NewRecorder()
	r.ServeHTTP(wCat, httptest.NewRequest(http.MethodGet, "/api/public/categories", nil))
	assert.Equal(t, http.StatusOK, wCat.Code)

	wPub := httptest.NewRecorder()
	r.ServeHTTP(wPub, httptest.NewRequest(http.MethodGet, "/api/public/posts", nil))
	assert.Equal(t, http.StatusOK, wPub.Code)

	// NoRoute 404
	w404 := httptest.NewRecorder()
	r.ServeHTTP(w404, httptest.NewRequest(http.MethodGet, "/unknown/path", nil))
	assert.Equal(t, http.StatusNotFound, w404.Code)
}
