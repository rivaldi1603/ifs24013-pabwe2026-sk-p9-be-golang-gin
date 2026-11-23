package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) { return 0, errors.New("read error") }
func (errReadCloser) Close() error             { return nil }

func newMockHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

type mockAIClient struct {
	fn func(ctx context.Context, sys, usr string) (string, error)
}

func (m *mockAIClient) Complete(ctx context.Context, sys, usr string) (string, error) {
	return m.fn(ctx, sys, usr)
}

type mockSearchClient struct {
	fn func(ctx context.Context, query string) ([]SearchSource, error)
}

func (m *mockSearchClient) Search(ctx context.Context, query string) ([]SearchSource, error) {
	return m.fn(ctx, query)
}

func TestSearchClient(t *testing.T) {
	ctx := context.Background()

	t.Run("empty endpoint and invalid URL", func(t *testing.T) {
		_, err := NewSearchClient("").Search(ctx, "golang")
		assert.Error(t, err)

		_, err = NewSearchClient("http://\x7f").Search(ctx, "golang")
		assert.Error(t, err)
	})

	t.Run("http error, read error, status error, bad json, results & data fallback", func(t *testing.T) {
		c := NewSearchClient("https://search.test").(*httpSearchClient)

		c.httpClient = newMockHTTPClient(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		})
		_, err := c.Search(ctx, "golang")
		assert.Error(t, err)

		c.httpClient = newMockHTTPClient(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: errReadCloser{}}, nil
		})
		_, err = c.Search(ctx, "golang")
		assert.Error(t, err)

		c.httpClient = newMockHTTPClient(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(bytes.NewBufferString("err"))}, nil
		})
		_, err = c.Search(ctx, "golang")
		assert.Error(t, err)

		c.httpClient = newMockHTTPClient(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString("bad-json"))}, nil
		})
		_, err = c.Search(ctx, "golang")
		assert.Error(t, err)

		c.httpClient = newMockHTTPClient(func(*http.Request) (*http.Response, error) {
			b, _ := json.Marshal(map[string]interface{}{
				"results": []SearchSource{{Title: "Go Docs", URL: "https://go.dev", Snippet: "Official Go docs"}},
			})
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(b))}, nil
		})
		res1, err := c.Search(ctx, "golang")
		require.NoError(t, err)
		assert.Len(t, res1, 1)

		c.httpClient = newMockHTTPClient(func(*http.Request) (*http.Response, error) {
			b, _ := json.Marshal(map[string]interface{}{
				"data": []SearchSource{{Title: "Gin", URL: "https://gin-gonic.com", Snippet: "Fast HTTP"}},
			})
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(b))}, nil
		})
		res2, err := c.Search(ctx, "gin")
		require.NoError(t, err)
		assert.Len(t, res2, 1)
	})
}

func TestOpenAIClient(t *testing.T) {
	ctx := context.Background()

	t.Run("empty endpoint and invalid URL", func(t *testing.T) {
		_, err := NewClient("", "k", "m").Complete(ctx, "s", "u")
		assert.Error(t, err)

		_, err = NewClient("http://\x7f", "k", "m").Complete(ctx, "s", "u")
		assert.Error(t, err)
	})

	t.Run("http errors and success", func(t *testing.T) {
		c := NewClient("https://ai.test", "my-key", "gpt-4o-mini").(*openAIClient)

		c.httpClient = newMockHTTPClient(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("connect failed")
		})
		_, err := c.Complete(ctx, "s", "u")
		assert.Error(t, err)

		c.httpClient = newMockHTTPClient(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: errReadCloser{}}, nil
		})
		_, err = c.Complete(ctx, "s", "u")
		assert.Error(t, err)

		c.httpClient = newMockHTTPClient(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(bytes.NewBufferString("upstream error"))}, nil
		})
		_, err = c.Complete(ctx, "s", "u")
		assert.Error(t, err)

		c.httpClient = newMockHTTPClient(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString("not-json"))}, nil
		})
		_, err = c.Complete(ctx, "s", "u")
		assert.Error(t, err)

		c.httpClient = newMockHTTPClient(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`{"choices":[]}`))}, nil
		})
		_, err = c.Complete(ctx, "s", "u")
		assert.Error(t, err)

		c.httpClient = newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "Bearer my-key", req.Header.Get("Authorization"))
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"choices":[{"message":{"role":"assistant","content":"hello"}}]}`)),
			}, nil
		})
		out, err := c.Complete(ctx, "s", "u")
		require.NoError(t, err)
		assert.Equal(t, "hello", out)
	})
}

func TestServiceAndHandler(t *testing.T) {
	t.Run("success with web search and fenced JSON", func(t *testing.T) {
		svc := NewService(
			&mockAIClient{fn: func(ctx context.Context, sys, usr string) (string, error) {
				assert.Contains(t, usr, "Referensi hasil penelusuran mesin pencari terkini")
				return "```json\n{\"title\":\"Tren AI 2026\",\"excerpt\":\"Ringkas\",\"content\":\"# Tren AI 2026\\nIsi lengkap.\"}\n```", nil
			}},
			&mockSearchClient{fn: func(ctx context.Context, q string) ([]SearchSource, error) {
				return []SearchSource{{Title: "Ref", URL: "https://example.com/ai", Snippet: "Snippet AI"}}, nil
			}},
		)
		h := NewHandler(svc)
		r := gin.New()
		h.RegisterRoutes(r.Group("/api"), func(c *gin.Context) { c.Next() })

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/posts/ai-generate", bytes.NewBufferString(`{"topic":"Tren AI 2026","include_web_search":true}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Data GeneratePostResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Tren AI 2026", resp.Data.Title)
		assert.Equal(t, "Umum", resp.Data.Category)
		require.Len(t, resp.Data.SearchSources, 1)
	})

	t.Run("validation, AI error, bad JSON, missing fields", func(t *testing.T) {
		svcErr := NewService(&mockAIClient{fn: func(ctx context.Context, sys, usr string) (string, error) {
			return "", errors.New("ai down")
		}}, nil)
		h := NewHandler(svcErr)
		r := gin.New()
		h.RegisterRoutes(r.Group("/api"), func(c *gin.Context) { c.Next() })

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/posts/ai-generate", bytes.NewBufferString(`{}`)))
		assert.Equal(t, http.StatusBadRequest, w.Code)

		w = httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/posts/ai-generate", bytes.NewBufferString(`{"topic":"Test"}`)))
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		svcBadJSON := NewService(&mockAIClient{fn: func(ctx context.Context, sys, usr string) (string, error) {
			return "not json", nil
		}}, nil)
		_, err := svcBadJSON.GeneratePost(context.Background(), GeneratePostRequest{Topic: "Test", Category: "Teknologi"})
		assert.Error(t, err)

		svcMissing := NewService(&mockAIClient{fn: func(ctx context.Context, sys, usr string) (string, error) {
			return `{"title":"","content":""}`, nil
		}}, nil)
		_, err = svcMissing.GeneratePost(context.Background(), GeneratePostRequest{Topic: "Test"})
		assert.Error(t, err)
	})
}
