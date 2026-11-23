package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func makeToken(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte(secret))
	require.NoError(t, err)
	return s
}

func TestCORS(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	req2 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestAuthAndRequireRole(t *testing.T) {
	secret := "test-secret"
	r := gin.New()
	r.GET("/protected", Auth(secret), RequireRole("admin"), func(c *gin.Context) {
		u, _ := c.Get("user")
		c.JSON(http.StatusOK, u)
	})
	r.GET("/role-only", RequireRole("admin"), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	t.Run("missing header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid signing method", func(t *testing.T) {
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"user_id": 1})
		tokenStr, err := tok.SignedString(priv)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid user_id claim", func(t *testing.T) {
		tokenStr := makeToken(t, secret, jwt.MapClaims{"user_id": "not-a-number"})
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("forbidden role", func(t *testing.T) {
		tokenStr := makeToken(t, secret, jwt.MapClaims{
			"user_id":  1,
			"username": "budi",
			"email":    "budi@example.com",
			"role":     "member",
		})
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("require role without user in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/role-only", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("valid admin token", func(t *testing.T) {
		tokenStr := makeToken(t, secret, jwt.MapClaims{
			"user_id":  1,
			"username": "admin",
			"email":    "admin@delcom.org",
			"role":     "admin",
		})
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRateLimiter(t *testing.T) {
	rDefault := gin.New()
	rDefault.Use(RateLimiter())
	rDefault.GET("/ok", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	wDef := httptest.NewRecorder()
	rDefault.ServeHTTP(wDef, httptest.NewRequest(http.MethodGet, "/ok", nil))
	assert.Equal(t, http.StatusOK, wDef.Code)

	r := gin.New()
	r.Use(NewRateLimiter(2, 50*time.Millisecond))
	r.GET("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req3.RemoteAddr = "1.2.3.4:1234"
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusTooManyRequests, w3.Code)

	time.Sleep(60 * time.Millisecond)
	w4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req4.RemoteAddr = "1.2.3.4:1234"
	r.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusOK, w4.Code)
}
