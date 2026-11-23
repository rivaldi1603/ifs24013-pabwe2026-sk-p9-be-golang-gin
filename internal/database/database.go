package database

import (
	"database/sql"
	"fmt"
	"time"

	"app/internal/config"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// User represents a blog user (admin or member/author).
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"size:255;uniqueIndex;not null" json:"username"`
	Email     string    `gorm:"size:255;uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"type:text;not null" json:"-"`
	Role      string    `gorm:"size:50;not null;default:member" json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Posts     []Post    `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"posts,omitempty"`
}

// Post represents a blog article written by an author.
type Post struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         uint       `gorm:"not null;index" json:"user_id"`
	User           *User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"author,omitempty"`
	Title          string     `gorm:"size:255;not null" json:"title"`
	Slug           string     `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Category       string     `gorm:"size:100;not null;default:Umum" json:"category"`
	Excerpt        string     `gorm:"type:text" json:"excerpt"`
	Content        string     `gorm:"type:text" json:"content"`
	CoverImagePath string     `gorm:"type:text" json:"cover_image_path"`
	IsPublished    bool       `gorm:"not null;default:false" json:"is_published"`
	PublishedAt    *time.Time `json:"published_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

var (
	hashPasswordFn = bcrypt.GenerateFromPassword
	openDBFn       = gorm.Open
	getSQLDBFn     = func(db *gorm.DB) (*sql.DB, error) { return db.DB() }
	pingDBFn       = func(s *sql.DB) error { return s.Ping() }
)

// Connect opens a GORM database connection based on Config (PostgreSQL or SQLite).
func Connect(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	if cfg.DBConnection == "sqlite" {
		dsn := cfg.DBDatabase
		if dsn == "" {
			dsn = ":memory:"
		}
		dialector = sqlite.Open(dsn)
	} else {
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Jakarta",
			cfg.DBHost, cfg.DBUsername, cfg.DBPassword, cfg.DBDatabase, cfg.DBPort, cfg.DBSSLMode,
		)
		dialector = postgres.Open(dsn)
	}

	db, err := openDBFn(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	if cfg.DBConnection != "sqlite" {
		sqlDB, err := getSQLDBFn(db)
		if err != nil {
			return nil, err
		}
		if err := pingDBFn(sqlDB); err != nil {
			return nil, err
		}
	}

	return db, nil
}

// Migrate runs GORM AutoMigrate for all core blog tables.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&User{}, &Post{})
}

// Seed creates default admin & member accounts and sample published blog articles if missing.
func Seed(db *gorm.DB, cfg *config.Config) error {
	adminUser, err := seedUser(db, cfg.AdminUsername, cfg.AdminUsername+"@delcom.org", cfg.AdminPassword, "admin")
	if err != nil {
		return err
	}

	memberUser, err := seedUser(db, "member", "member@delcom.org", "member", "member")
	if err != nil {
		return err
	}

	var count int64
	if err := db.Model(&Post{}).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		now := time.Now()
		posts := []Post{
			{
				UserID:      adminUser.ID,
				Title:       "Membangun REST API Skalabel dengan Golang Gin & Clean Architecture",
				Slug:        "membangun-rest-api-skalabel-dengan-golang-gin-clean-architecture",
				Category:    "Teknologi",
				Excerpt:     "Panduan praktis menyusun proyek backend Golang Gin berbasis Feature-First, GORM, JWT, dan 100% unit test coverage.",
				Content:     "# Membangun REST API Skalabel dengan Golang Gin\n\nArsitektur **Feature-First & Clean Layering** memudahkan tim memperluas fitur tanpa menciptakan dependensi melingkar.\n\n## Lapisan Utama\n- **DTO**: Kontrak payload request & response.\n- **Repository**: Akses data GORM ke PostgreSQL.\n- **Service**: Aturan bisnis, slug unik, dan kalkulasi *reading time*.\n- **Handler**: Routing HTTP Gin dengan respons terstandar `{status, message, data}`.\n\n## Kesimpulan\nDengan memisahkan setiap fitur secara mandiri, pengujian unit dapat berjalan cepat menggunakan SQLite in-memory.",
				IsPublished: true,
				PublishedAt: &now,
			},
			{
				UserID:      memberUser.ID,
				Title:       "Strategi Belajar Pengembangan Web Modern untuk Mahasiswa Informatika",
				Slug:        "strategi-belajar-pengembangan-web-modern-untuk-mahasiswa-informatika",
				Category:    "Pendidikan",
				Excerpt:     "Langkah terstruktur menguasai backend REST API, ReactJS SPA, dan otomatisasi pengujian untuk persiapan karier rekayasa perangkat lunak.",
				Content:     "# Strategi Belajar Pengembangan Web Modern\n\nMenguasai *full-stack web development* membutuhkan kombinasi pemahaman fundamental dan praktik proyek nyata.\n\n1. Pahami kontrak HTTP, status code, dan autentikasi berbasis token JWT.\n2. Bangun antarmuka reaktif dengan **ReactJS**, **Vite**, dan **Tailwind CSS**.\n3. Terapkan *Test-Driven Development (TDD)* untuk menjaga kualitas kode.",
				IsPublished: true,
				PublishedAt: &now,
			},
			{
				UserID:      memberUser.ID,
				Title:       "Transformasi Konten Digital Menggunakan AI Generator & Search Engine",
				Slug:        "transformasi-konten-digital-menggunakan-ai-generator-search-engine",
				Category:    "Bisnis",
				Excerpt:     "Bagaimana integrasi mesin pencari live dengan model bahasa meningkatkan akurasi dan kecepatan produksi artikel bisnis.",
				Content:     "# Transformasi Konten Digital dengan AI & Search Engine\n\nIntegrasi penelusuran web real-time membantu penulis menyusun draf artikel yang relevan dengan data mutakhir serta dilengkapi rujukan transparan.\n\n> Artikel berbasis sumber yang dapat diaudit membangun kepercayaan pembaca secara berkelanjutan.",
				IsPublished: true,
				PublishedAt: &now,
			},
		}

		for i := range posts {
			if err := db.Create(&posts[i]).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func seedUser(db *gorm.DB, username, email, password, role string) (*User, error) {
	var user User
	err := db.Where("username = ?", username).First(&user).Error
	if err == nil {
		return &user, nil
	}

	hashed, err := hashPasswordFn([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user = User{
		Username: username,
		Email:    email,
		Password: string(hashed),
		Role:     role,
	}
	if err := db.Create(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
