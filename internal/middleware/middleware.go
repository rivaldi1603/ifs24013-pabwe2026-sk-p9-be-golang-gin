package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"app/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthUser represents authenticated user claims stored in the Gin context.
type AuthUser struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// CORS adds cross-origin headers and handles preflight OPTIONS requests.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// Auth validates Bearer JWT tokens and injects AuthUser into the Gin context.
func Auth(secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			response.AbortFail(c, http.StatusUnauthorized, "Unauthorized: missing or invalid token")
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secretKey), nil
		})
		if err != nil || !token.Valid {
			response.AbortFail(c, http.StatusUnauthorized, "Unauthorized: invalid or expired token")
			return
		}

		claims := token.Claims.(jwt.MapClaims)

		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			response.AbortFail(c, http.StatusUnauthorized, "Unauthorized: invalid user_id claim")
			return
		}

		username, _ := claims["username"].(string)
		email, _ := claims["email"].(string)
		role, _ := claims["role"].(string)

		user := AuthUser{
			UserID:   uint(userIDFloat),
			Username: username,
			Email:    email,
			Role:     role,
		}
		c.Set("user", user)
		c.Set("user_id", user.UserID)
		c.Set("role", user.Role)
		c.Next()
	}
}

// RequireRole ensures the authenticated user has one of the required roles.
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(c *gin.Context) {
		val, exists := c.Get("user")
		if !exists {
			response.AbortFail(c, http.StatusUnauthorized, "Unauthorized")
			return
		}
		user, ok := val.(AuthUser)
		if !ok || !allowed[user.Role] {
			response.AbortFail(c, http.StatusForbidden, "Forbidden: insufficient role permissions")
			return
		}
		c.Next()
	}
}

type rateEntry struct {
	count     int
	resetTime time.Time
}

// RateLimiter limits excessive requests per client IP with default thresholds.
func RateLimiter() gin.HandlerFunc {
	return NewRateLimiter(300, time.Minute)
}

// NewRateLimiter creates a per-IP fixed-window rate limiter middleware.
func NewRateLimiter(maxRequests int, window time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	clients := make(map[string]*rateEntry)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		mu.Lock()
		entry, exists := clients[ip]
		if !exists || now.After(entry.resetTime) {
			clients[ip] = &rateEntry{
				count:     1,
				resetTime: now.Add(window),
			}
			mu.Unlock()
			c.Next()
			return
		}

		if entry.count >= maxRequests {
			mu.Unlock()
			response.AbortFail(c, http.StatusTooManyRequests, "Too many requests, please try again later")
			return
		}

		entry.count++
		mu.Unlock()
		c.Next()
	}
}
