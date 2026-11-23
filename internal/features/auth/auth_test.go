package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/config"
	"app/internal/database"
	"app/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Connect(&config.Config{DBConnection: "sqlite", DBDatabase: ":memory:"})
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	return db
}

type mockRepo struct {
	createFn         func(*database.User) error
	findByUsernameFn func(string) (*database.User, error)
	findByEmailFn    func(string) (*database.User, error)
	findByIDFn       func(uint) (*database.User, error)
	updateFn         func(*database.User) error
}

func (m *mockRepo) Create(u *database.User) error { return m.createFn(u) }
func (m *mockRepo) FindByUsername(u string) (*database.User, error) {
	return m.findByUsernameFn(u)
}
func (m *mockRepo) FindByEmail(e string) (*database.User, error) { return m.findByEmailFn(e) }
func (m *mockRepo) FindByID(id uint) (*database.User, error)     { return m.findByIDFn(id) }
func (m *mockRepo) Update(u *database.User) error                { return m.updateFn(u) }

func TestAuthEndToEnd(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo, "secret")
	h := NewHandler(svc)

	r := gin.New()
	api := r.Group("/api")
	h.RegisterRoutes(api, middleware.Auth("secret"))

	// Validation fail
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{}`)))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Register default member (invalid role forced to member)
	regBody := `{"username":"penulis1","email":"penulis1@example.com","password":"Password123!","role":"super"}`
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(regBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var regResp struct {
		Status string       `json:"status"`
		Data   AuthResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &regResp))
	token := regResp.Data.Token
	assert.NotEmpty(t, token)
	assert.Equal(t, "member", regResp.Data.User.Role)

	// Register explicit admin role
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"username":"admin2","email":"admin2@example.com","password":"Password123!","role":"admin"}`)))
	assert.Equal(t, http.StatusCreated, w.Code)
	var adminResp struct {
		Data AuthResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &adminResp))
	assert.Equal(t, "admin", adminResp.Data.User.Role)

	// Duplicate username & duplicate email
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(regBody)))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	dupEmail := `{"username":"penulis2","email":"penulis1@example.com","password":"Password123!"}`
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(dupEmail)))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Login tests (missing fields, unknown, wrong password, by username, by email)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{}`)))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"nope","password":"123"}`)))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"penulis1","password":"wrong"}`)))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"penulis1","password":"Password123!"}`)))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"penulis1@example.com","password":"Password123!"}`)))
	assert.Equal(t, http.StatusOK, w.Code)

	// FindByID miss + Profile & Update (invalid email 400 + success) & Logout
	_, err := repo.FindByID(9999)
	assert.Error(t, err)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/auth/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/auth/profile", bytes.NewBufferString(`{"email":"bukan-email"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/auth/profile", bytes.NewBufferString(`{"username":"penulis_new","email":"new@example.com","password":"NewPassword123!"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServiceAndHandlerErrors(t *testing.T) {
	dbErr := errors.New("db failure")
	notFound := func(string) (*database.User, error) { return nil, gorm.ErrRecordNotFound }

	t.Run("Register errors", func(t *testing.T) {
		svc := NewService(&mockRepo{findByUsernameFn: func(string) (*database.User, error) { return nil, dbErr }}, "secret")
		_, err := svc.Register(RegisterRequest{Username: "u", Email: "e@e.com", Password: "password"})
		assert.ErrorIs(t, err, dbErr)

		svc = NewService(&mockRepo{
			findByUsernameFn: notFound,
			findByEmailFn:    func(string) (*database.User, error) { return nil, dbErr },
		}, "secret")
		_, err = svc.Register(RegisterRequest{Username: "u", Email: "e@e.com", Password: "password"})
		assert.ErrorIs(t, err, dbErr)

		origHash := generateHashFn
		generateHashFn = func([]byte, int) ([]byte, error) { return nil, errors.New("hash err") }
		svc = NewService(&mockRepo{findByUsernameFn: notFound, findByEmailFn: notFound}, "secret")
		_, err = svc.Register(RegisterRequest{Username: "u", Email: "e@e.com", Password: "password"})
		assert.Error(t, err)
		generateHashFn = origHash

		svc = NewService(&mockRepo{
			findByUsernameFn: notFound,
			findByEmailFn:    notFound,
			createFn:         func(*database.User) error { return dbErr },
		}, "secret")
		_, err = svc.Register(RegisterRequest{Username: "u", Email: "e@e.com", Password: "password"})
		assert.ErrorIs(t, err, dbErr)

		origSign := signTokenFn
		signTokenFn = func(*jwt.Token, []byte) (string, error) { return "", errors.New("sign err") }
		svc = NewService(&mockRepo{
			findByUsernameFn: notFound,
			findByEmailFn:    notFound,
			createFn:         func(*database.User) error { return nil },
		}, "secret")
		_, err = svc.Register(RegisterRequest{Username: "u", Email: "e@e.com", Password: "password"})
		assert.Error(t, err)
		signTokenFn = origSign
	})

	t.Run("Login errors", func(t *testing.T) {
		svc := NewService(&mockRepo{findByUsernameFn: func(string) (*database.User, error) { return nil, dbErr }}, "secret")
		_, err := svc.Login(LoginRequest{Username: "u", Password: "p"})
		assert.ErrorIs(t, err, dbErr)

		svc = NewService(&mockRepo{
			findByUsernameFn: notFound,
			findByEmailFn:    func(string) (*database.User, error) { return nil, dbErr },
		}, "secret")
		_, err = svc.Login(LoginRequest{Username: "e@e.com", Password: "p"})
		assert.ErrorIs(t, err, dbErr)

		hashed, err := generateHashFn([]byte("pass"), 4)
		require.NoError(t, err)
		origSign := signTokenFn
		signTokenFn = func(*jwt.Token, []byte) (string, error) { return "", errors.New("sign err") }
		svc = NewService(&mockRepo{
			findByUsernameFn: func(string) (*database.User, error) {
				return &database.User{ID: 1, Username: "u", Password: string(hashed), Role: "member"}, nil
			},
		}, "secret")
		_, err = svc.Login(LoginRequest{Username: "u", Password: "pass"})
		assert.Error(t, err)
		signTokenFn = origSign
	})

	t.Run("Profile errors", func(t *testing.T) {
		svc := NewService(&mockRepo{findByIDFn: func(uint) (*database.User, error) { return nil, gorm.ErrRecordNotFound }}, "secret")
		_, err := svc.GetProfile(1)
		assert.ErrorIs(t, err, ErrUserNotFound)
		_, err = svc.UpdateProfile(1, UpdateProfileRequest{})
		assert.ErrorIs(t, err, ErrUserNotFound)

		svc = NewService(&mockRepo{findByIDFn: func(uint) (*database.User, error) { return nil, dbErr }}, "secret")
		_, err = svc.GetProfile(1)
		assert.ErrorIs(t, err, dbErr)
		_, err = svc.UpdateProfile(1, UpdateProfileRequest{})
		assert.ErrorIs(t, err, dbErr)

		origHash := generateHashFn
		generateHashFn = func([]byte, int) ([]byte, error) { return nil, errors.New("hash err") }
		svc = NewService(&mockRepo{findByIDFn: func(uint) (*database.User, error) { return &database.User{ID: 1}, nil }}, "secret")
		_, err = svc.UpdateProfile(1, UpdateProfileRequest{Password: "newpass"})
		assert.Error(t, err)
		generateHashFn = origHash

		svc = NewService(&mockRepo{
			findByIDFn: func(uint) (*database.User, error) { return &database.User{ID: 1}, nil },
			updateFn:   func(*database.User) error { return dbErr },
		}, "secret")
		_, err = svc.UpdateProfile(1, UpdateProfileRequest{Email: "a@b.com"})
		assert.ErrorIs(t, err, dbErr)
	})

	t.Run("Handler 500 and 404", func(t *testing.T) {
		svc := NewService(&mockRepo{
			findByUsernameFn: func(string) (*database.User, error) { return nil, dbErr },
			findByIDFn:       func(uint) (*database.User, error) { return nil, gorm.ErrRecordNotFound },
		}, "secret")
		h := NewHandler(svc)
		r := gin.New()
		r.Use(func(c *gin.Context) { c.Set("user_id", uint(99)); c.Next() })
		h.RegisterRoutes(r.Group("/api"), func(c *gin.Context) { c.Next() })

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"username":"abc","email":"a@b.com","password":"password"}`)))
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		w = httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"abc","password":"password"}`)))
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		w = httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/auth/profile", nil))
		assert.Equal(t, http.StatusNotFound, w.Code)

		w = httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/auth/profile", bytes.NewBufferString(`{"email":"x@y.com"}`)))
		assert.Equal(t, http.StatusNotFound, w.Code)

		svcErr := NewService(&mockRepo{findByIDFn: func(uint) (*database.User, error) { return nil, dbErr }}, "secret")
		hErr := NewHandler(svcErr)
		rErr := gin.New()
		rErr.Use(func(c *gin.Context) { c.Set("user_id", uint(1)); c.Next() })
		hErr.RegisterRoutes(rErr.Group("/api"), func(c *gin.Context) { c.Next() })

		w = httptest.NewRecorder()
		rErr.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/auth/profile", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		w = httptest.NewRecorder()
		rErr.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/auth/profile", bytes.NewBufferString(`{"email":"x@y.com"}`)))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
