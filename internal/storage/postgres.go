package storage

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"goldie/internal/model"
)

type PostgresConnection struct {
	DB *gorm.DB
}

func NewPostgresConnection(logger *slog.Logger, connectionString string, logLevel slog.Level) (*PostgresConnection, error) {
	gormLogger := slogGorm.New(
		slogGorm.WithHandler(logger.Handler()),
		slogGorm.WithTraceAll(),
		slogGorm.SetLogLevel(slogGorm.ErrorLogType, slog.LevelError),
		slogGorm.SetLogLevel(slogGorm.SlowQueryLogType, slog.LevelWarn),
		slogGorm.SetLogLevel(slogGorm.DefaultLogType, logLevel),
	)

	db, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{Logger: gormLogger})
	if err != nil {
		return nil, fmt.Errorf("open connection: %w", err)
	}

	return &PostgresConnection{DB: db}, nil
}

func MustNewPostgresConnection(logger *slog.Logger, connectionString string, logLevel slog.Level) *PostgresConnection {
	conn, err := NewPostgresConnection(logger, connectionString, logLevel)
	if err != nil {
		panic(err)
	}

	return conn
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
		model.TgChat{},
		model.TgChatAlert2{},
	)

	if err != nil {
		panic(fmt.Errorf("migrate models: %w", err))
	}

	migrateAlert2Data(s.DB)
}

func migrateAlert2Data(db *gorm.DB) {
	migrator := db.Migrator()

	hasAlert2Column := migrator.HasColumn(&model.TgChat{}, "alert2")
	hasAlert2DateColumn := migrator.HasColumn(&model.TgChat{}, "alert2_date")

	if !hasAlert2Column && !hasAlert2DateColumn {
		return
	}

	type legacyChat struct {
		SourceID   int64
		Alert2Date time.Time
		Alert2     bool
	}

	var rows []legacyChat
	if err := db.Model(&model.TgChat{}).Where("alert2 = ? AND alert2_date IS NOT NULL", true).Find(&rows).Error; err != nil {
		panic(fmt.Errorf("fetch legacy alert2 data: %w", err))
	}

	if len(rows) > 0 {
		for _, row := range rows {
			if row.Alert2Date.IsZero() {
				continue
			}

			var chat model.TgChat
			if err := db.Where("source_id = ?", row.SourceID).First(&chat).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				panic(fmt.Errorf("fetch chat for migration: %w", err))
			}

			alert := &model.TgChatAlert2{ChatID: chat.ID, PurchaseDate: row.Alert2Date}
			if err := db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "chat_id"}, {Name: "purchase_date"}},
				DoNothing: true,
			}).Create(alert).Error; err != nil {
				panic(fmt.Errorf("migrate alert2 record: %w", err))
			}
		}
	}

	if hasAlert2Column {
		if err := migrator.DropColumn(&model.TgChat{}, "alert2"); err != nil {
			panic(fmt.Errorf("drop alert2 column: %w", err))
		}
	}

	if hasAlert2DateColumn {
		if err := migrator.DropColumn(&model.TgChat{}, "alert2_date"); err != nil {
			panic(fmt.Errorf("drop alert2_date column: %w", err))
		}
	}

	if migrator.HasColumn(&model.TgChatAlert2{}, "chat_source_id") {
		if err := migrator.DropColumn(&model.TgChatAlert2{}, "chat_source_id"); err != nil {
			panic(fmt.Errorf("drop chat_source_id column: %w", err))
		}
	}
}
