package post

import (
	"strings"

	"app/internal/database"

	"gorm.io/gorm"
)

// Repository defines database operations for public posts, author posts, and admin monitoring.
type Repository interface {
	ListPublicPosts(filter PostFilter) ([]database.Post, error)
	FindPublicPostBySlug(slug string) (*database.Post, error)
	SlugExists(slug string, excludeID uint) (bool, error)

	ListPostsByUser(userID uint, filter PostFilter) ([]database.Post, error)
	FindPostByIDAndUser(id, userID uint) (*database.Post, error)
	CreatePost(post *database.Post) error
	UpdatePost(post *database.Post) error
	DeletePost(post *database.Post) error
	GetStatsByUser(userID uint) (*PostStatsResponse, error)

	ListAllUsersWithStats() ([]AdminUserSummary, error)
	ListAllPosts() ([]database.Post, error)
	FindUserByID(userID uint) (*database.User, error)
	ListAllPostsByUser(userID uint) ([]database.Post, error)
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository creates a GORM-backed post Repository.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func applyPostFilter(q *gorm.DB, filter PostFilter) *gorm.DB {
	if cat := strings.TrimSpace(filter.Category); cat != "" && cat != "Semua" {
		q = q.Where("category = ?", cat)
	}
	if s := strings.TrimSpace(strings.ToLower(filter.Query)); s != "" {
		like := "%" + s + "%"
		q = q.Where("LOWER(title) LIKE ? OR LOWER(excerpt) LIKE ? OR LOWER(content) LIKE ?", like, like, like)
	}
	return q
}

func (r *gormRepository) ListPublicPosts(filter PostFilter) ([]database.Post, error) {
	var posts []database.Post
	q := r.db.Preload("User").Where("is_published = ?", true)
	q = applyPostFilter(q, filter)
	err := q.Order("published_at desc, id desc").Find(&posts).Error
	return posts, err
}

func (r *gormRepository) FindPublicPostBySlug(slug string) (*database.Post, error) {
	var p database.Post
	err := r.db.Preload("User").Where("slug = ? AND is_published = ?", slug, true).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *gormRepository) SlugExists(slug string, excludeID uint) (bool, error) {
	var count int64
	q := r.db.Model(&database.Post{}).Where("slug = ?", slug)
	if excludeID > 0 {
		q = q.Where("id != ?", excludeID)
	}
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *gormRepository) ListPostsByUser(userID uint, filter PostFilter) ([]database.Post, error) {
	var posts []database.Post
	q := r.db.Preload("User").Where("user_id = ?", userID)
	q = applyPostFilter(q, filter)
	if filter.Status == "published" {
		q = q.Where("is_published = ?", true)
	} else if filter.Status == "draft" {
		q = q.Where("is_published = ?", false)
	}
	err := q.Order("id desc").Find(&posts).Error
	return posts, err
}

func (r *gormRepository) FindPostByIDAndUser(id, userID uint) (*database.Post, error) {
	var p database.Post
	err := r.db.Preload("User").Where("id = ? AND user_id = ?", id, userID).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *gormRepository) CreatePost(post *database.Post) error {
	if err := r.db.Create(post).Error; err != nil {
		return err
	}
	return r.db.Preload("User").First(post, post.ID).Error
}

func (r *gormRepository) UpdatePost(post *database.Post) error {
	return r.db.Save(post).Error
}

func (r *gormRepository) DeletePost(post *database.Post) error {
	return r.db.Delete(post).Error
}

func (r *gormRepository) GetStatsByUser(userID uint) (*PostStatsResponse, error) {
	var total int64
	if err := r.db.Model(&database.Post{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, err
	}
	var published int64
	if err := r.db.Model(&database.Post{}).Where("user_id = ? AND is_published = ?", userID, true).Count(&published).Error; err != nil {
		return nil, err
	}
	return &PostStatsResponse{
		Total:        total,
		TotalTerbit:  published,
		BelumSelesai: total - published,
	}, nil
}

func (r *gormRepository) ListAllUsersWithStats() ([]AdminUserSummary, error) {
	var users []database.User
	if err := r.db.Order("id asc").Find(&users).Error; err != nil {
		return nil, err
	}
	summaries := make([]AdminUserSummary, 0, len(users))
	for _, u := range users {
		var total, published int64
		if err := r.db.Model(&database.Post{}).Where("user_id = ?", u.ID).Count(&total).Error; err != nil {
			return nil, err
		}
		if err := r.db.Model(&database.Post{}).Where("user_id = ? AND is_published = ?", u.ID, true).Count(&published).Error; err != nil {
			return nil, err
		}
		summaries = append(summaries, AdminUserSummary{
			ID:             u.ID,
			Username:       u.Username,
			Email:          u.Email,
			Role:           u.Role,
			TotalPosts:     total,
			PublishedPosts: published,
			CreatedAt:      u.CreatedAt,
			UpdatedAt:      u.UpdatedAt,
		})
	}
	return summaries, nil
}

func (r *gormRepository) ListAllPosts() ([]database.Post, error) {
	var posts []database.Post
	err := r.db.Preload("User").Order("id desc").Find(&posts).Error
	return posts, err
}

func (r *gormRepository) FindUserByID(userID uint) (*database.User, error) {
	var user database.User
	if err := r.db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *gormRepository) ListAllPostsByUser(userID uint) ([]database.Post, error) {
	var posts []database.Post
	err := r.db.Preload("User").Where("user_id = ?", userID).Order("id desc").Find(&posts).Error
	return posts, err
}
