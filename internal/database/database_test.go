package database

import (
	"database/sql"
	"errors"
	"testing"

	"app/internal/config"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestConnectSQLiteAndMigrateAndSeed(t *testing.T) {
	cfg := &config.Config{
		DBConnection:  "sqlite",
		DBDatabase:    "",
		AdminUsername: "admin",
		AdminPassword: "admin",
	}

	db, err := Connect(cfg)
	require.NoError(t, err)
	require.NotNil(t, db)

	require.NoError(t, Migrate(db))
	require.NoError(t, Seed(db, cfg))

	// Calling Seed a second time should be idempotent.
	require.NoError(t, Seed(db, cfg))

	var userCount int64
	require.NoError(t, db.Model(&User{}).Count(&userCount).Error)
	assert.Equal(t, int64(2), userCount)

	var postCount int64
	require.NoError(t, db.Model(&Post{}).Where("is_published = ?", true).Count(&postCount).Error)
	assert.Equal(t, int64(3), postCount)
}

func TestConnectPostgresAndErrors(t *testing.T) {
	cfg := &config.Config{
		DBConnection: "pgsql",
		DBHost:       "127.0.0.1",
		DBPort:       "59999",
		DBDatabase:   "nonexistent",
		DBUsername:   "postgres",
		DBPassword:   "postgres",
		DBSSLMode:    "disable",
	}

	db, err := Connect(cfg)
	assert.Error(t, err)
	assert.Nil(t, db)

	// Invalid sqlite path triggers open error
	_, err = Connect(&config.Config{
		DBConnection: "sqlite",
		DBDatabase:   "/nonexistent_dir_xyz_999/test.db",
	})
	assert.Error(t, err)

	origOpen := openDBFn
	defer func() { openDBFn = origOpen }()
	openDBFn = func(_ gorm.Dialector, opts ...gorm.Option) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open(":memory:"), opts...)
	}

	// Success using real default getSQLDBFn and pingDBFn
	dbOK, err := Connect(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, dbOK)

	// getSQLDBFn failure
	orig := getSQLDBFn
	defer func() { getSQLDBFn = orig }()
	getSQLDBFn = func(*gorm.DB) (*sql.DB, error) {
		return nil, errors.New("sqldb error")
	}
	_, err = Connect(cfg)
	assert.Error(t, err)
	getSQLDBFn = orig

	// pingDBFn failure
	origPing := pingDBFn
	defer func() { pingDBFn = origPing }()
	pingDBFn = func(*sql.DB) error { return errors.New("ping error") }
	_, err = Connect(cfg)
	assert.Error(t, err)
}

func TestSeedErrors(t *testing.T) {
	cfg := &config.Config{
		DBConnection:  "sqlite",
		DBDatabase:    ":memory:",
		AdminUsername: "admin",
		AdminPassword: "admin",
	}

	t.Run("unmigrated db fails admin seed", func(t *testing.T) {
		db, err := Connect(cfg)
		require.NoError(t, err)
		err = Seed(db, cfg)
		assert.Error(t, err)
	})

	t.Run("hash error", func(t *testing.T) {
		db, err := Connect(cfg)
		require.NoError(t, err)
		require.NoError(t, Migrate(db))

		orig := hashPasswordFn
		defer func() { hashPasswordFn = orig }()
		hashPasswordFn = func([]byte, int) ([]byte, error) {
			return nil, errors.New("hash failure")
		}

		err = Seed(db, cfg)
		assert.Error(t, err)
	})

	t.Run("member seed error", func(t *testing.T) {
		db, err := Connect(cfg)
		require.NoError(t, err)
		require.NoError(t, Migrate(db))

		_, err = seedUser(db, "admin", "admin@delcom.org", "admin", "admin")
		require.NoError(t, err)

		orig := hashPasswordFn
		defer func() { hashPasswordFn = orig }()
		hashPasswordFn = func([]byte, int) ([]byte, error) {
			return nil, errors.New("member hash failure")
		}

		err = Seed(db, cfg)
		assert.Error(t, err)
	})

	t.Run("posts table missing on count", func(t *testing.T) {
		db, err := Connect(cfg)
		require.NoError(t, err)
		require.NoError(t, db.AutoMigrate(&User{}))

		err = Seed(db, cfg)
		assert.Error(t, err)
	})

	t.Run("posts table invalid schema on create", func(t *testing.T) {
		db, err := Connect(cfg)
		require.NoError(t, err)
		require.NoError(t, db.AutoMigrate(&User{}))
		require.NoError(t, db.Exec("CREATE TABLE posts (id INTEGER PRIMARY KEY)").Error)

		err = Seed(db, cfg)
		assert.Error(t, err)
	})
}
