package database

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"nexus/internal/config"
	"nexus/internal/model"
)

// New 打开数据库连接并执行自动迁移。
func New(cfg *config.Config) (*gorm.DB, error) {
	dialector, err := dialector(cfg)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.AutoMigrate(&model.User{}, &model.Session{}, &model.Agent{}, &model.Conversation{}, &model.Message{}); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	return db, nil
}

// dialector 根据配置的数据库类型选择 GORM 驱动。
func dialector(cfg *config.Config) (gorm.Dialector, error) {
	switch cfg.Database.Type {
	case "postgres", "postgresql":
		return postgres.Open(cfg.Database.DSN), nil
	case "sqlite":
		return sqlite.Open(cfg.Database.DSN), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Database.Type)
	}
}
