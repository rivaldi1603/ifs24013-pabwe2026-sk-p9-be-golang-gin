package post

import (
	"math"
	"regexp"
	"strings"
	"time"

	"app/internal/database"
)

// ValidCategories lists the supported Delcom Blog article categories.
var ValidCategories = []string{
	"Teknologi",
	"Pendidikan",
	"Gaya Hidup",
	"Bisnis",
	"Opini",
	"Umum",
}

// CreatePostRequest represents the payload to create a blog post.
type CreatePostRequest struct {
	Title          string `json:"title" binding:"required"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt"`
	Content        string `json:"content" binding:"required"`
	CoverImagePath string `json:"cover_image_path"`
	IsPublished    bool   `json:"is_published"`
}

// UpdatePostRequest represents the payload to update a blog post.
type UpdatePostRequest struct {
	Title          string `json:"title"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt"`
	Content        string `json:"content"`
	CoverImagePath string `json:"cover_image_path"`
	IsPublished    *bool  `json:"is_published"`
}

// PostFilter holds optional search and filter parameters.
type PostFilter struct {
	Query    string
	Category string
	Status   string
}

// AuthorSummary holds author metadata attached to internal/admin post responses.
type AuthorSummary struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// PublicPostResponse represents a public-safe blog article without internal user_id or email.
type PublicPostResponse struct {
	ID             uint       `json:"id"`
	Title          string     `json:"title"`
	Slug           string     `json:"slug"`
	Category       string     `json:"category"`
	Excerpt        string     `json:"excerpt"`
	Content        string     `json:"content"`
	CoverImagePath string     `json:"cover_image_path"`
	AuthorName     string     `json:"author_name"`
	ReadingTime    int        `json:"reading_time"`
	PublishedAt    *time.Time `json:"published_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// PostResponse represents a serialized blog article for author workspace and admin monitoring.
type PostResponse struct {
	ID             uint           `json:"id"`
	UserID         uint           `json:"user_id"`
	Author         *AuthorSummary `json:"author,omitempty"`
	AuthorName     string         `json:"author_name"`
	Title          string         `json:"title"`
	Slug           string         `json:"slug"`
	Category       string         `json:"category"`
	Excerpt        string         `json:"excerpt"`
	Content        string         `json:"content"`
	CoverImagePath string         `json:"cover_image_path"`
	IsPublished    bool           `json:"is_published"`
	PublishedAt    *time.Time     `json:"published_at"`
	ReadingTime    int            `json:"reading_time"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// PostStatsResponse represents the 3 author summary stat cards.
type PostStatsResponse struct {
	TotalArticles int64 `json:"total_articles"`
	TotalSelesai  int64 `json:"total_selesai"`
	BelumSelesai  int64 `json:"belum_selesai"`
}

// AdminUserSummary represents user metrics for the admin dashboard.
type AdminUserSummary struct {
	ID             uint      `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	TotalArticles  int64     `json:"total_articles"`
	TotalPublished int64     `json:"total_published"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

var nonAlnumRegex = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts an article title into a clean URL-friendly slug.
func Slugify(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = nonAlnumRegex.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "artikel"
	}
	return s
}

// CalculateReadingTime estimates reading duration in minutes at 200 words per minute.
func CalculateReadingTime(content string) int {
	words := len(strings.Fields(content))
	if words == 0 {
		return 1
	}
	return int(math.Ceil(float64(words) / 200.0))
}

func isValidCategory(category string) bool {
	for _, c := range ValidCategories {
		if c == category {
			return true
		}
	}
	return false
}

func buildExcerpt(excerpt, content string) string {
	trimmed := strings.TrimSpace(excerpt)
	if trimmed != "" {
		return trimmed
	}
	plain := strings.TrimSpace(strings.ReplaceAll(content, "\n", " "))
	if len(plain) > 160 {
		return strings.TrimSpace(plain[:160]) + "..."
	}
	return plain
}

func resolveAuthorName(u *database.User) string {
	if u != nil && u.Username != "" {
		return u.Username
	}
	return "Delcom Author"
}

func toPublicPostResponse(p database.Post) PublicPostResponse {
	return PublicPostResponse{
		ID:             p.ID,
		Title:          p.Title,
		Slug:           p.Slug,
		Category:       p.Category,
		Excerpt:        p.Excerpt,
		Content:        p.Content,
		CoverImagePath: p.CoverImagePath,
		AuthorName:     resolveAuthorName(p.User),
		ReadingTime:    CalculateReadingTime(p.Content),
		PublishedAt:    p.PublishedAt,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

func toPublicPostResponses(posts []database.Post) []PublicPostResponse {
	out := make([]PublicPostResponse, 0, len(posts))
	for _, p := range posts {
		out = append(out, toPublicPostResponse(p))
	}
	return out
}

func toPostResponse(p database.Post) PostResponse {
	res := PostResponse{
		ID:             p.ID,
		UserID:         p.UserID,
		AuthorName:     resolveAuthorName(p.User),
		Title:          p.Title,
		Slug:           p.Slug,
		Category:       p.Category,
		Excerpt:        p.Excerpt,
		Content:        p.Content,
		CoverImagePath: p.CoverImagePath,
		IsPublished:    p.IsPublished,
		PublishedAt:    p.PublishedAt,
		ReadingTime:    CalculateReadingTime(p.Content),
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
	if p.User != nil {
		res.Author = &AuthorSummary{
			ID:       p.User.ID,
			Username: p.User.Username,
			Email:    p.User.Email,
			Role:     p.User.Role,
		}
	}
	return res
}

func toPostResponses(posts []database.Post) []PostResponse {
	out := make([]PostResponse, 0, len(posts))
	for _, p := range posts {
		out = append(out, toPostResponse(p))
	}
	return out
}
