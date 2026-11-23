package post

import (
	"errors"
	"net/http"
	"strconv"

	"app/internal/response"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP endpoints for public blog, author workspace, and admin monitoring.
type Handler struct {
	service Service
}

// NewHandler creates a new post Handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes mounts public, author, and admin blog routes onto the router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware, adminMiddleware gin.HandlerFunc) {
	pub := rg.Group("/public")
	{
		pub.GET("/categories", h.GetCategories)
		pub.GET("/posts", h.ListPublicPosts)
		pub.GET("/posts/:slug", h.GetPublicPostBySlug)
	}

	posts := rg.Group("/posts", authMiddleware)
	{
		posts.GET("", h.ListMyPosts)
		posts.POST("", h.CreatePost)
		posts.GET("/stats", h.GetMyStats)
		posts.GET("/:id", h.GetMyPost)
		posts.PUT("/:id", h.UpdatePost)
		posts.PATCH("/:id/toggle-publish", h.TogglePublish)
		posts.DELETE("/:id", h.DeletePost)
	}

	admin := rg.Group("/admin", authMiddleware, adminMiddleware)
	{
		admin.GET("/users", h.AdminListUsers)
		admin.GET("/posts", h.AdminListPosts)
		admin.GET("/users/:id/posts", h.AdminGetUserPosts)
	}
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, http.StatusBadRequest, "Invalid ID parameter")
		return 0, false
	}
	return uint(id), true
}

func (h *Handler) GetCategories(c *gin.Context) {
	response.Success(c, http.StatusOK, "Categories retrieved", h.service.GetCategories())
}

func (h *Handler) ListPublicPosts(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		query = c.Query("search")
	}
	filter := PostFilter{
		Query:    query,
		Category: c.Query("category"),
	}
	posts, err := h.service.ListPublicPosts(filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to load public posts")
		return
	}
	response.Success(c, http.StatusOK, "Public posts retrieved", posts)
}

func (h *Handler) GetPublicPostBySlug(c *gin.Context) {
	slug := c.Param("slug")
	p, err := h.service.GetPublicPostBySlug(slug)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			response.Fail(c, http.StatusNotFound, "Post not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to load post")
		return
	}
	response.Success(c, http.StatusOK, "Post retrieved", p)
}

func (h *Handler) ListMyPosts(c *gin.Context) {
	userID := c.GetUint("user_id")
	filter := PostFilter{
		Query:    c.Query("q"),
		Category: c.Query("category"),
		Status:   c.Query("status"),
	}
	posts, err := h.service.ListMyPosts(userID, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to load author posts")
		return
	}
	response.Success(c, http.StatusOK, "Author posts retrieved", posts)
}

func (h *Handler) CreatePost(c *gin.Context) {
	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}
	userID := c.GetUint("user_id")
	created, err := h.service.CreatePost(userID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidCategory) {
			response.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to create post")
		return
	}
	response.Success(c, http.StatusCreated, "Post created successfully", created)
}

func (h *Handler) GetMyStats(c *gin.Context) {
	userID := c.GetUint("user_id")
	stats, err := h.service.GetMyStats(userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to load post stats")
		return
	}
	response.Success(c, http.StatusOK, "Post stats retrieved", stats)
}

func (h *Handler) GetMyPost(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	userID := c.GetUint("user_id")
	p, err := h.service.GetMyPost(id, userID)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			response.Fail(c, http.StatusNotFound, "Post not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to load post")
		return
	}
	response.Success(c, http.StatusOK, "Post retrieved", p)
}

func (h *Handler) UpdatePost(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	userID := c.GetUint("user_id")
	updated, err := h.service.UpdatePost(id, userID, req)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			response.Fail(c, http.StatusNotFound, "Post not found")
			return
		}
		if errors.Is(err, ErrInvalidCategory) {
			response.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to update post")
		return
	}
	response.Success(c, http.StatusOK, "Post updated successfully", updated)
}

func (h *Handler) TogglePublish(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	userID := c.GetUint("user_id")
	updated, err := h.service.TogglePublish(id, userID)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			response.Fail(c, http.StatusNotFound, "Post not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to toggle publish status")
		return
	}
	response.Success(c, http.StatusOK, "Post publish status updated", updated)
}

func (h *Handler) DeletePost(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	userID := c.GetUint("user_id")
	if err := h.service.DeletePost(id, userID); err != nil {
		if errors.Is(err, ErrPostNotFound) {
			response.Fail(c, http.StatusNotFound, "Post not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to delete post")
		return
	}
	response.Success(c, http.StatusOK, "Post deleted successfully", nil)
}

func (h *Handler) AdminListUsers(c *gin.Context) {
	users, err := h.service.AdminListUsers()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to load users")
		return
	}
	response.Success(c, http.StatusOK, "Users retrieved", users)
}

func (h *Handler) AdminListPosts(c *gin.Context) {
	posts, err := h.service.AdminListPosts()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to load all posts")
		return
	}
	response.Success(c, http.StatusOK, "All posts retrieved", posts)
}

func (h *Handler) AdminGetUserPosts(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	posts, err := h.service.AdminGetUserPosts(id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.Fail(c, http.StatusNotFound, "User not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to load user posts")
		return
	}
	response.Success(c, http.StatusOK, "User posts retrieved", posts)
}
