package auth

import (
	"app/internal/database"

	"gorm.io/gorm"
)

// Repository defines database operations for user authentication and profiles.
type Repository interface {
	Create(user *database.User) error
	FindByUsername(username string) (*database.User, error)
	FindByEmail(email string) (*database.User, error)
	FindByID(id uint) (*database.User, error)
	Update(user *database.User) error
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository creates a GORM-backed auth Repository.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) Create(user *database.User) error {
	return r.db.Create(user).Error
}

func (r *gormRepository) FindByUsername(username string) (*database.User, error) {
	var user database.User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *gormRepository) FindByEmail(email string) (*database.User, error) {
	var user database.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *gormRepository) FindByID(id uint) (*database.User, error) {
	var user database.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *gormRepository) Update(user *database.User) error {
	return r.db.Save(user).Error
}
