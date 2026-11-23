package post

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal/config"
	"app/internal/database"

	"github.com/gin-gonic/gin"
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

func jsonInt(v uint) string {
	b, _ := json.Marshal(v)
	return string(b)
}

type mockRepo struct {
	listPublicPostsFn       func(PostFilter) ([]database.Post, error)
	findPublicPostBySlugFn  func(string) (*database.Post, error)
	slugExistsFn            func(string, uint) (bool, error)
	listPostsByUserFn       func(uint, PostFilter) ([]database.Post, error)
	findPostByIDAndUserFn   func(uint, uint) (*database.Post, error)
	createPostFn            func(*database.Post) error
	updatePostFn            func(*database.Post) error
	deletePostFn            func(*database.Post) error
	getStatsByUserFn        func(uint) (*PostStatsResponse, error)
	listAllUsersWithStatsFn func() ([]AdminUserSummary, error)
	listAllPostsFn          func() ([]database.Post, error)
	findUserByIDFn          func(uint) (*database.User, error)
	listAllPostsByUserFn    func(uint) ([]database.Post, error)
}

func (m *mockRepo) ListPublicPosts(f PostFilter) ([]database.Post, error) {
	return m.listPublicPostsFn(f)
}
func (m *mockRepo) FindPublicPostBySlug(s string) (*database.Post, error) {
	return m.findPublicPostBySlugFn(s)
}
func (m *mockRepo) SlugExists(s string, ex uint) (bool, error) { return m.slugExistsFn(s, ex) }
func (m *mockRepo) ListPostsByUser(u uint, f PostFilter) ([]database.Post, error) {
	return m.listPostsByUserFn(u, f)
}
func (m *mockRepo) FindPostByIDAndUser(id, u uint) (*database.Post, error) {
	return m.findPostByIDAndUserFn(id, u)
}
func (m *mockRepo) CreatePost(p *database.Post) error { return m.createPostFn(p) }
func (m *mockRepo) UpdatePost(p *database.Post) error { return m.updatePostFn(p) }
func (m *mockRepo) DeletePost(p *database.Post) error { return m.deletePostFn(p) }
func (m *mockRepo) GetStatsByUser(u uint) (*PostStatsResponse, error) {
	return m.getStatsByUserFn(u)
}
func (m *mockRepo) ListAllUsersWithStats() ([]AdminUserSummary, error) {
	return m.listAllUsersWithStatsFn()
}
func (m *mockRepo) ListAllPosts() ([]database.Post, error) { return m.listAllPostsFn() }
func (m *mockRepo) FindUserByID(u uint) (*database.User, error) {
	return m.findUserByIDFn(u)
}
func (m *mockRepo) ListAllPostsByUser(u uint) ([]database.Post, error) {
	return m.listAllPostsByUserFn(u)
}

func TestHelpersSlugAndReadingTime(t *testing.T) {
	assert.Equal(t, "hello-world-2026", Slugify("  Hello, World! 2026  "))
	assert.Equal(t, "artikel", Slugify("!!!"))
	assert.Equal(t, 1, CalculateReadingTime(""))
	assert.Equal(t, 1, CalculateReadingTime("short text"))
	assert.Equal(t, 3, CalculateReadingTime(strings.Repeat("word ", 401)))
	longContent := strings.Repeat("a", 200)
	assert.True(t, strings.HasSuffix(buildExcerpt("", longContent), "..."))
	assert.Equal(t, "short", buildExcerpt("", "short"))
}

func TestPostEndToEnd(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, database.Seed(db, &config.Config{AdminUsername: "admin", AdminPassword: "admin"}))

	var member database.User
	require.NoError(t, db.Where("username = ?", "member").First(&member).Error)

	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	r := gin.New()
	authMW := func(c *gin.Context) { c.Set("user_id", member.ID); c.Next() }
	adminMW := func(c *gin.Context) { c.Next() }
	h.RegisterRoutes(r.Group("/api"), authMW, adminMW)

	// 1. Public categories & public posts
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/public/categories", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/public/posts?search=Golang&category=Teknologi", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/public/posts/membangun-rest-api-skalabel-dengan-golang-gin-clean-architecture", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/public/posts/tidak-ada", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)

	// 2. Author CreatePost (validation, invalid category, success with duplicate slug)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/posts", bytes.NewBufferString(`{}`)))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/posts", bytes.NewBufferString(`{"title":"Test","content":"Body","category":"Invalid"}`)))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/posts", bytes.NewBufferString(`{"title":"Artikel Unik","content":"Isi artikel markdown lengkap","is_published":true}`)))
	assert.Equal(t, http.StatusCreated, w.Code)

	var created1 struct {
		Data PostResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created1))
	assert.Equal(t, "artikel-unik", created1.Data.Slug)
	assert.Equal(t, "Umum", created1.Data.Category)
	assert.Equal(t, 1, created1.Data.ReadingTime)

	// Second post with identical title generates unique slug "-2"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/posts", bytes.NewBufferString(`{"title":"Artikel Unik","category":"Teknologi","excerpt":"Ringkas","content":"Body 2","is_published":false}`)))
	assert.Equal(t, http.StatusCreated, w.Code)

	var created2 struct {
		Data PostResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created2))
	assert.Equal(t, "artikel-unik-2", created2.Data.Slug)
	pid := created2.Data.ID

	// 3. List author posts (status published & draft) and stats
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/posts?status=published&q=Unik", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/posts?status=draft", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/posts/stats", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 4. GetMyPost
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/posts/abc", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/posts/9999", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/posts/"+jsonInt(pid), nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 5. UpdatePost (bad id, bad json, 404, bad category, success publish & unpublish)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/posts/abc", bytes.NewBufferString(`{}`)))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/posts/"+jsonInt(pid), bytes.NewBufferString(`{bad`)))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/posts/9999", bytes.NewBufferString(`{"title":"X"}`)))
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/posts/"+jsonInt(pid), bytes.NewBufferString(`{"category":"Wrong"}`)))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/posts/"+jsonInt(pid), bytes.NewBufferString(`{"title":"Judul Baru","category":"Opini","excerpt":"Ex","content":"Konten baru","cover_image_path":"/cover.jpg","is_published":true}`)))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/posts/"+jsonInt(pid), bytes.NewBufferString(`{"is_published":false}`)))
	assert.Equal(t, http.StatusOK, w.Code)

	// 6. TogglePublish (bad id, 404, draft -> published -> draft)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/api/posts/0/toggle-publish", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/api/posts/9999/toggle-publish", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/api/posts/"+jsonInt(pid)+"/toggle-publish", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/api/posts/"+jsonInt(pid)+"/toggle-publish", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 7. Admin routes
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/users", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/posts", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/users/abc/posts", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/users/9999/posts", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/users/"+jsonInt(member.ID)+"/posts", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 8. DeletePost
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/posts/abc", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/posts/9999", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/posts/"+jsonInt(pid), nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPostRepoServiceHandlerErrors(t *testing.T) {
	dbErr := errors.New("db error")

	t.Run("Repository DB errors", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewRepository(db)

		// Missing posts table
		require.NoError(t, db.Migrator().DropTable(&database.Post{}))
		_, err := repo.SlugExists("x", 0)
		assert.Error(t, err)
		err = repo.CreatePost(&database.Post{Title: "A"})
		assert.Error(t, err)
		_, err = repo.GetStatsByUser(1)
		assert.Error(t, err)

		require.NoError(t, db.Create(&database.User{Username: "u1", Email: "u1@e.com", Password: "p", Role: "member"}).Error)
		_, err = repo.ListAllUsersWithStats()
		assert.Error(t, err)

		// Posts table without is_published column fails second Count in GetStatsByUser and ListAllUsersWithStats
		require.NoError(t, db.Exec("CREATE TABLE posts (id INTEGER PRIMARY KEY, user_id INTEGER)").Error)
		_, err = repo.GetStatsByUser(1)
		assert.Error(t, err)
		_, err = repo.ListAllUsersWithStats()
		assert.Error(t, err)

		// Missing users table fails ListAllUsersWithStats on users query
		require.NoError(t, db.Migrator().DropTable(&database.User{}))
		_, err = repo.ListAllUsersWithStats()
		assert.Error(t, err)
	})

	t.Run("Service and Handler 500 error paths", func(t *testing.T) {
		mr := &mockRepo{
			listPublicPostsFn:       func(PostFilter) ([]database.Post, error) { return nil, dbErr },
			findPublicPostBySlugFn:  func(string) (*database.Post, error) { return nil, dbErr },
			slugExistsFn:            func(string, uint) (bool, error) { return false, dbErr },
			listPostsByUserFn:       func(uint, PostFilter) ([]database.Post, error) { return nil, dbErr },
			findPostByIDAndUserFn:   func(uint, uint) (*database.Post, error) { return nil, dbErr },
			createPostFn:            func(*database.Post) error { return dbErr },
			updatePostFn:            func(*database.Post) error { return dbErr },
			deletePostFn:            func(*database.Post) error { return dbErr },
			getStatsByUserFn:        func(uint) (*PostStatsResponse, error) { return nil, dbErr },
			listAllUsersWithStatsFn: func() ([]AdminUserSummary, error) { return nil, dbErr },
			listAllPostsFn:          func() ([]database.Post, error) { return nil, dbErr },
			findUserByIDFn:          func(uint) (*database.User, error) { return nil, dbErr },
			listAllPostsByUserFn:    func(uint) ([]database.Post, error) { return nil, dbErr },
		}
		svc := NewService(mr)
		h := NewHandler(svc)
		r := gin.New()
		pass := func(c *gin.Context) { c.Set("user_id", uint(1)); c.Next() }
		h.RegisterRoutes(r.Group("/api"), pass, pass)

		routes := []struct {
			method, path, body string
		}{
			{http.MethodGet, "/api/public/posts", ""},
			{http.MethodGet, "/api/public/posts/some-slug", ""},
			{http.MethodGet, "/api/posts", ""},
			{http.MethodPost, "/api/posts", `{"title":"A","content":"B"}`},
			{http.MethodGet, "/api/posts/stats", ""},
			{http.MethodGet, "/api/posts/1", ""},
			{http.MethodPut, "/api/posts/1", `{"title":"A"}`},
			{http.MethodPatch, "/api/posts/1/toggle-publish", ""},
			{http.MethodDelete, "/api/posts/1", ""},
			{http.MethodGet, "/api/admin/users", ""},
			{http.MethodGet, "/api/admin/posts", ""},
			{http.MethodGet, "/api/admin/users/1/posts", ""},
		}
		for _, rt := range routes {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(rt.method, rt.path, bytes.NewBufferString(rt.body)))
			assert.Equal(t, http.StatusInternalServerError, w.Code, rt.method+" "+rt.path)
		}

		// Additional service branches where lookup succeeds but write/slug fails
		mr.slugExistsFn = func(string, uint) (bool, error) { return false, nil }
		_, err := svc.CreatePost(1, CreatePostRequest{Title: "A", Content: "B"})
		assert.ErrorIs(t, err, dbErr)

		mr.findPostByIDAndUserFn = func(uint, uint) (*database.Post, error) {
			return &database.Post{ID: 1, Title: "Old"}, nil
		}
		mr.slugExistsFn = func(string, uint) (bool, error) { return false, dbErr }
		_, err = svc.UpdatePost(1, 1, UpdatePostRequest{Title: "New"})
		assert.ErrorIs(t, err, dbErr)

		mr.slugExistsFn = func(string, uint) (bool, error) { return false, nil }
		_, err = svc.UpdatePost(1, 1, UpdatePostRequest{Title: "New"})
		assert.ErrorIs(t, err, dbErr)

		_, err = svc.TogglePublish(1, 1)
		assert.ErrorIs(t, err, dbErr)

		mr.findUserByIDFn = func(uint) (*database.User, error) { return &database.User{ID: 1}, nil }
		_, err = svc.AdminGetUserPosts(1)
		assert.ErrorIs(t, err, dbErr)
	})
}
