package storage

import (
	"fmt"
	"log/slog"

	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"goldie/internal/model"
)

type PostgresConnection struct {
	DB *gorm.DB
}

func MustNewPostgresConnection(logger *slog.Logger, connectionString string, logLevel slog.Level) *PostgresConnection {
	gormLogger := slogGorm.New(
		slogGorm.WithHandler(logger.Handler()),
		slogGorm.WithTraceAll(),
		slogGorm.SetLogLevel(slogGorm.ErrorLogType, slog.LevelError),
		slogGorm.SetLogLevel(slogGorm.SlowQueryLogType, slog.LevelWarn),
		slogGorm.SetLogLevel(slogGorm.DefaultLogType, logLevel),
	)

	db, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{Logger: gormLogger})
	if err != nil {
		panic(fmt.Errorf("open connection: %w", err))
	}

	return &PostgresConnection{DB: db}
}

func (s *PostgresConnection) MustClose() {
	connection, err := s.DB.DB()
	if err != nil {
		panic(fmt.Errorf("get db connection: %w", err))
	}

	if err = connection.Close(); err != nil {
		panic(fmt.Errorf("close connection: %w", err))
	}
}

func (s *PostgresConnection) MustMigration() {
	err := s.DB.AutoMigrate(
		model.GoldPrice{},
	)

	if err != nil {
		panic(fmt.Errorf("migrate models: %w", err))
	}
}
