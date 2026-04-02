package bootstrap

import (
	"fmt"
	"os"

	"camopanel/server/internal/config"
	"camopanel/server/internal/modules/auth/domain"
	authrepo "camopanel/server/internal/modules/auth/repo"
	platformaudit "camopanel/server/internal/platform/audit"
	platformauth "camopanel/server/internal/platform/auth"
	platformdb "camopanel/server/internal/platform/db"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Application struct {
	cfg     config.Config
	router  *gin.Engine
	modules ModuleSet
	db      *gorm.DB
}

func New(cfg config.Config) (*Application, error) {
	normalized := NormalizeConfig(cfg)

	if err := os.MkdirAll(normalized.ProjectsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create projects dir: %w", err)
	}

	db, err := platformdb.OpenSQLite(normalized.DatabasePath)
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&authrepo.UserRecord{}, &platformaudit.Record{}); err != nil {
		return nil, fmt.Errorf("migrate bootstrap tables: %w", err)
	}
	if err := cleanupLegacyApprovalData(db); err != nil {
		return nil, fmt.Errorf("cleanup legacy approval data: %w", err)
	}
	if err := seedAdmin(db, normalized); err != nil {
		return nil, err
	}

	modules, err := NewModules(normalized, db)
	if err != nil {
		return nil, fmt.Errorf("init modules: %w", err)
	}

	return &Application{
		cfg:     normalized,
		router:  NewRouter(modules),
		modules: modules,
		db:      db,
	}, nil
}

func (a *Application) Run() error {
	return a.router.Run(a.cfg.HTTPAddr)
}

func (a *Application) Router() *gin.Engine {
	return a.router
}

func (a *Application) Modules() ModuleSet {
	return a.modules
}

func (a *Application) DB() *gorm.DB {
	return a.db
}

func seedAdmin(db *gorm.DB, cfg config.Config) error {
	var count int64
	if err := db.Model(&authrepo.UserRecord{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}

	hash, err := platformauth.HashPassword(cfg.AdminPassword)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	return db.Create(&authrepo.UserRecord{
		ID:           uuid.NewString(),
		Username:     cfg.AdminUsername,
		PasswordHash: hash,
		Role:         domain.RoleSuperAdmin,
	}).Error
}

func cleanupLegacyApprovalData(db *gorm.DB) error {
	if db.Migrator().HasTable(&platformaudit.Record{}) {
		if err := db.Where("action LIKE ?", "approval_%").Delete(&platformaudit.Record{}).Error; err != nil {
			return err
		}
	}
	if db.Migrator().HasTable("approval_requests") {
		return db.Migrator().DropTable("approval_requests")
	}
	return nil
}
