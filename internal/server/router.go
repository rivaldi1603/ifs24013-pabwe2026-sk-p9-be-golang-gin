package server

import (
	"net/http"

	"app/docs"
	"app/internal/config"
	"app/internal/features/ai"
	"app/internal/features/auth"
	"app/internal/features/post"
	"app/internal/middleware"
	"app/internal/response"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// SetupRouter initializes Gin with middlewares, Swagger UI, all Delcom Blog routes, and 404 handler.
func SetupRouter(cfg *config.Config, db *gorm.DB) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery(), middleware.CORS(), middleware.RateLimiter())

	r.GET("/api.json", func(c *gin.Context) {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", docs.GetSwaggerJSON(c.Request.Host, scheme))
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/api.json")))

	authMW := middleware.Auth(cfg.AppKey)
	adminMW := middleware.RequireRole("admin")

	api := r.Group("/api")

	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(authRepo, cfg.AppKey)
	auth.NewHandler(authSvc).RegisterRoutes(api, authMW)

	postRepo := post.NewRepository(db)
	postSvc := post.NewService(postRepo)
	post.NewHandler(postSvc).RegisterRoutes(api, authMW, adminMW)

	aiClient := ai.NewClient(cfg.OpenAIEndpoint, cfg.OpenAIKey, cfg.OpenAIModel)
	searchClient := ai.NewSearchClient(cfg.SearchEngineEndpoint)
	aiSvc := ai.NewService(aiClient, searchClient)
	ai.NewHandler(aiSvc).RegisterRoutes(api, authMW)

	r.NoRoute(func(c *gin.Context) {
		response.Fail(c, http.StatusNotFound, "Endpoint not found")
	})

	return r
}
