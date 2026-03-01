package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	customLogger "github.com/rockship/cosmo-agents-go/pkg/logger"
)

// PostgresConfig holds PostgreSQL configuration
type PostgresConfig struct {
	DSN         string
	MaxOpenConn int
	MaxIdleConn int
	MaxLifetime time.Duration
}

// NewPostgresDB creates a new PostgreSQL connection
// Equivalent to Python's core/db/session.py
func NewPostgresDB(cfg PostgresConfig) (*gorm.DB, error) {
	// GORM logger configuration
	gormLogger := logger.New(
		&gormLogWriter{},
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		Logger:                                   gormLogger,
		DisableForeignKeyConstraintWhenMigrating: false,
		PrepareStmt:                              true,
		SkipDefaultTransaction:                   true, // Disable auto-transaction for better performance
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Get generic database object sql.DB to set connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Connection pool settings (matching Python's asyncpg settings)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConn)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConn)
	sqlDB.SetConnMaxLifetime(cfg.MaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	customLogger.Info("PostgreSQL connected successfully")

	return db, nil
}

// gormLogWriter implements GORM's Writer interface
type gormLogWriter struct{}

func (w *gormLogWriter) Printf(format string, args ...interface{}) {
	customLogger.Logger.Debug().Msgf(format, args...)
}

// Close closes the database connection
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Transaction executes a function within a database transaction
// Equivalent to Python's @Transactional decorator
func Transaction(db *gorm.DB, fn func(*gorm.DB) error) error {
	return db.Transaction(fn)
}
