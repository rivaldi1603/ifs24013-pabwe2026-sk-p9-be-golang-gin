package post

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"app/internal/database"

	"gorm.io/gorm"
)

var (
	ErrPostNotFound    = errors.New("post not found")
	ErrForbidden       = errors.New("forbidden: you do not own this post")
	ErrInvalidCategory = errors.New("invalid category")
)

// Service defines business logic for Delcom Blog posts and admin monitoring.
type Service interface {
	GetCategories() []string
	ListPublicPosts(filter PostFilter) ([]PublicPostResponse, error)
	GetPublicPostBySlug(slug string) (*PublicPostResponse, error)

	ListMyPosts(userID uint, filter PostFilter) ([]PostResponse, error)
	CreatePost(userID uint, req CreatePostRequest) (*PostResponse, error)
	GetMyPost(id, userID uint) (*PostResponse, error)
	UpdatePost(id, userID uint, req UpdatePostRequest) (*PostResponse, error)
	TogglePublish(id, userID uint) (*PostResponse, error)
	DeletePost(id, userID uint) error
	GetMyStats(userID uint) (*PostStatsResponse, error)

	AdminListUsers() ([]AdminUserSummary, error)
	AdminListPosts() ([]PostResponse, error)
	AdminGetUserPosts(userID uint) ([]PostResponse, error)
}

type service struct {
	repo Repository
}

// NewService creates a new post Service.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetCategories() []string {
	out := make([]string, len(ValidCategories))
	copy(out, ValidCategories)
	return out
}

func (s *service) ListPublicPosts(filter PostFilter) ([]PublicPostResponse, error) {
	posts, err := s.repo.ListPublicPosts(filter)
	if err != nil {
		return nil, err
	}
	return toPublicPostResponses(posts), nil
}

func (s *service) GetPublicPostBySlug(slug string) (*PublicPostResponse, error) {
	p, err := s.repo.FindPublicPostBySlug(slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}
	res := toPublicPostResponse(*p)
	return &res, nil
}

func (s *service) ListMyPosts(userID uint, filter PostFilter) ([]PostResponse, error) {
	posts, err := s.repo.ListPostsByUser(userID, filter)
	if err != nil {
		return nil, err
	}
	return toPostResponses(posts), nil
}

func (s *service) generateUniqueSlug(title string, excludeID uint) (string, error) {
	base := Slugify(title)
	candidate := base
	for suffix := 2; ; suffix++ {
		exists, err := s.repo.SlugExists(candidate, excludeID)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, suffix)
	}
}

func (s *service) findOwnedPost(id, userID uint) (*database.Post, error) {
	p, err := s.repo.FindPostByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}
	if p.UserID != userID {
		return nil, ErrForbidden
	}
	return p, nil
}

func (s *service) CreatePost(userID uint, req CreatePostRequest) (*PostResponse, error) {
	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = "Umum"
	}
	if !isValidCategory(category) {
		return nil, ErrInvalidCategory
	}

	slug, err := s.generateUniqueSlug(req.Title, 0)
	if err != nil {
		return nil, err
	}

	var publishedAt *time.Time
	if req.IsPublished {
		now := time.Now()
		publishedAt = &now
	}

	p := &database.Post{
		UserID:         userID,
		Title:          strings.TrimSpace(req.Title),
		Slug:           slug,
		Category:       category,
		Excerpt:        buildExcerpt(req.Excerpt, req.Content),
		Content:        req.Content,
		CoverImagePath: strings.TrimSpace(req.CoverImagePath),
		IsPublished:    req.IsPublished,
		PublishedAt:    publishedAt,
	}

	if err := s.repo.CreatePost(p); err != nil {
		return nil, err
	}
	res := toPostResponse(*p)
	return &res, nil
}

func (s *service) GetMyPost(id, userID uint) (*PostResponse, error) {
	p, err := s.findOwnedPost(id, userID)
	if err != nil {
		return nil, err
	}
	res := toPostResponse(*p)
	return &res, nil
}

func (s *service) UpdatePost(id, userID uint, req UpdatePostRequest) (*PostResponse, error) {
	p, err := s.findOwnedPost(id, userID)
	if err != nil {
		return nil, err
	}

	if title := strings.TrimSpace(req.Title); title != "" && title != p.Title {
		slug, err := s.generateUniqueSlug(title, p.ID)
		if err != nil {
			return nil, err
		}
		p.Title = title
		p.Slug = slug
	}
	if cat := strings.TrimSpace(req.Category); cat != "" {
		if !isValidCategory(cat) {
			return nil, ErrInvalidCategory
		}
		p.Category = cat
	}
	if req.Excerpt != "" {
		p.Excerpt = strings.TrimSpace(req.Excerpt)
	}
	if req.Content != "" {
		p.Content = req.Content
	}
	if req.CoverImagePath != "" {
		p.CoverImagePath = strings.TrimSpace(req.CoverImagePath)
	}
	if req.IsPublished != nil {
		p.IsPublished = *req.IsPublished
		if p.IsPublished && p.PublishedAt == nil {
			now := time.Now()
			p.PublishedAt = &now
		} else if !p.IsPublished {
			p.PublishedAt = nil
		}
	}

	if err := s.repo.UpdatePost(p); err != nil {
		return nil, err
	}
	res := toPostResponse(*p)
	return &res, nil
}

func (s *service) TogglePublish(id, userID uint) (*PostResponse, error) {
	p, err := s.findOwnedPost(id, userID)
	if err != nil {
		return nil, err
	}

	p.IsPublished = !p.IsPublished
	if p.IsPublished {
		now := time.Now()
		p.PublishedAt = &now
	} else {
		p.PublishedAt = nil
	}

	if err := s.repo.UpdatePost(p); err != nil {
		return nil, err
	}
	res := toPostResponse(*p)
	return &res, nil
}

func (s *service) DeletePost(id, userID uint) error {
	p, err := s.findOwnedPost(id, userID)
	if err != nil {
		return err
	}
	return s.repo.DeletePost(p)
}

func (s *service) GetMyStats(userID uint) (*PostStatsResponse, error) {
	return s.repo.GetStatsByUser(userID)
}

func (s *service) AdminListUsers() ([]AdminUserSummary, error) {
	return s.repo.ListAllUsersWithStats()
}

func (s *service) AdminListPosts() ([]PostResponse, error) {
	posts, err := s.repo.ListAllPosts()
	if err != nil {
		return nil, err
	}
	return toPostResponses(posts), nil
}

func (s *service) AdminGetUserPosts(userID uint) ([]PostResponse, error) {
	posts, err := s.repo.ListAllPostsByUser(userID)
	if err != nil {
		return nil, err
	}
	return toPostResponses(posts), nil
}
