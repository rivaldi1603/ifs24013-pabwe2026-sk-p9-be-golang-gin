package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfigDefaults(t *testing.T) {
	keys := []string{
		"APP_ENV", "APP_PORT", "APP_KEY",
		"ADMIN_USERNAME", "ADMIN_PASSWORD",
		"DB_CONNECTION", "DB_HOST", "DB_PORT", "DB_DATABASE", "DB_USERNAME", "DB_PASSWORD", "DB_SSLMODE",
		"SEARCH_ENGINE_ENDPOINT", "OPENAI_ENDPOINT", "OPENAI_KEY", "OPENAI_MODEL",
	}
	for _, k := range keys {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}

	cfg := LoadConfig()
	assert.Equal(t, "development", cfg.AppEnv)
	assert.Equal(t, "8081", cfg.AppPort)
	assert.NotEmpty(t, cfg.AppKey)
	assert.Equal(t, "admin", cfg.AdminUsername)
	assert.Equal(t, "admin", cfg.AdminPassword)
	assert.Equal(t, "pgsql", cfg.DBConnection)
	assert.Equal(t, "127.0.0.1", cfg.DBHost)
	assert.Equal(t, "5432", cfg.DBPort)
	assert.Equal(t, "pabwe_p9_blog", cfg.DBDatabase)
	assert.Equal(t, "postgres", cfg.DBUsername)
	assert.Equal(t, "postgres", cfg.DBPassword)
	assert.Equal(t, "disable", cfg.DBSSLMode)
	assert.Equal(t, "https://delcom.org/api/v1/search", cfg.SearchEngineEndpoint)
	assert.Equal(t, "https://delcom.org/api/v1", cfg.OpenAIEndpoint)
	assert.Equal(t, "key", cfg.OpenAIKey)
	assert.Equal(t, "gpt-4o-mini", cfg.OpenAIModel)
}

func TestLoadConfigOverrides(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_PORT", "9091")
	t.Setenv("APP_KEY", "custom-blog-key")
	t.Setenv("ADMIN_USERNAME", "superadmin")
	t.Setenv("ADMIN_PASSWORD", "secret")
	t.Setenv("DB_CONNECTION", "sqlite")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_DATABASE", "custom_blog")
	t.Setenv("DB_USERNAME", "custom_user")
	t.Setenv("DB_PASSWORD", "custom_pass")
	t.Setenv("DB_SSLMODE", "require")
	t.Setenv("SEARCH_ENGINE_ENDPOINT", "https://search.example.com")
	t.Setenv("OPENAI_ENDPOINT", "https://api.openai.com/v1")
	t.Setenv("OPENAI_KEY", "sk-test")
	t.Setenv("OPENAI_MODEL", "gpt-4")

	cfg := LoadConfig()
	assert.Equal(t, "production", cfg.AppEnv)
	assert.Equal(t, "9091", cfg.AppPort)
	assert.Equal(t, "custom-blog-key", cfg.AppKey)
	assert.Equal(t, "superadmin", cfg.AdminUsername)
	assert.Equal(t, "secret", cfg.AdminPassword)
	assert.Equal(t, "sqlite", cfg.DBConnection)
	assert.Equal(t, "localhost", cfg.DBHost)
	assert.Equal(t, "5433", cfg.DBPort)
	assert.Equal(t, "custom_blog", cfg.DBDatabase)
	assert.Equal(t, "custom_user", cfg.DBUsername)
	assert.Equal(t, "custom_pass", cfg.DBPassword)
	assert.Equal(t, "require", cfg.DBSSLMode)
	assert.Equal(t, "https://search.example.com", cfg.SearchEngineEndpoint)
	assert.Equal(t, "https://api.openai.com/v1", cfg.OpenAIEndpoint)
	assert.Equal(t, "sk-test", cfg.OpenAIKey)
	assert.Equal(t, "gpt-4", cfg.OpenAIModel)
}
