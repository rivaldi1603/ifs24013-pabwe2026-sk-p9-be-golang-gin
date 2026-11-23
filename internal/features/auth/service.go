package auth

import (
	"errors"
	"strings"
	"time"

	"app/internal/database"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserExists         = errors.New("username or email already registered")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserNotFound       = errors.New("user not found")
)

var (
	generateHashFn = bcrypt.GenerateFromPassword
	signTokenFn    = func(tok *jwt.Token, key []byte) (string, error) {
		return tok.SignedString(key)
	}
)

// Service defines business logic for user authentication and profile management.
type Service interface {
	Register(req RegisterRequest) (*AuthResponse, error)
	Login(req LoginRequest) (*AuthResponse, error)
	GetProfile(userID uint) (*UserProfileResponse, error)
	UpdateProfile(userID uint, req UpdateProfileRequest) (*UserProfileResponse, error)
}

type service struct {
	repo      Repository
	secretKey string
}

// NewService creates a new auth Service.
func NewService(repo Repository, secretKey string) Service {
	return &service{
		repo:      repo,
		secretKey: secretKey,
	}
}

func (s *service) Register(req RegisterRequest) (*AuthResponse, error) {
	if _, err := s.repo.FindByUsername(strings.TrimSpace(req.Username)); err == nil {
		return nil, ErrUserExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if _, err := s.repo.FindByEmail(strings.TrimSpace(req.Email)); err == nil {
		return nil, ErrUserExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hashed, err := generateHashFn([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &database.User{
		Username: strings.TrimSpace(req.Username),
		Email:    strings.TrimSpace(req.Email),
		Password: string(hashed),
		Role:     "member",
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		User:  toUserProfile(user),
	}, nil
}

func (s *service) Login(req LoginRequest) (*AuthResponse, error) {
	user, err := s.repo.FindByUsername(strings.TrimSpace(req.Username))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		User:  toUserProfile(user),
	}, nil
}

func (s *service) GetProfile(userID uint) (*UserProfileResponse, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	res := toUserProfile(user)
	return &res, nil
}

func (s *service) UpdateProfile(userID uint, req UpdateProfileRequest) (*UserProfileResponse, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if u := strings.TrimSpace(req.Username); u != "" {
		user.Username = u
	}
	if e := strings.TrimSpace(req.Email); e != "" {
		user.Email = e
	}
	if req.Password != "" {
		hashed, err := generateHashFn([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.Password = string(hashed)
	}

	if err := s.repo.Update(user); err != nil {
		return nil, err
	}
	res := toUserProfile(user)
	return &res, nil
}

func (s *service) generateToken(user *database.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return signTokenFn(tok, []byte(s.secretKey))
}
