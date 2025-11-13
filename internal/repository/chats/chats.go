package chats

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"goldie/internal/model"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (that *Repository) EnableAlert1(ctx context.Context, chatID int64) error {
	query := that.db.WithContext(ctx).Model(&model.TgChat{}).Where("source_id = ?", chatID)

	result := query.Update("alert1", "true")
	if err := result.Error; err != nil {
		return fmt.Errorf("update existing chat: %w", err)
	}

	if result.RowsAffected == 0 {
		if err := query.Create(&model.TgChat{SourceID: chatID, Alert1Enabled: true}).Error; err != nil {
			return fmt.Errorf("create new chat: %w", err)
		}
	}

	return nil
}

func (that *Repository) EnableAlert2(ctx context.Context, chatID int64, date time.Time) error {
	query := that.db.WithContext(ctx).Model(&model.TgChat{}).Where("source_id = ?", chatID)

	result := query.Updates(map[string]interface{}{"alert2": true, "alert2_date": date, "updated_at": time.Now()})
	if err := result.Error; err != nil {
		return fmt.Errorf("update existing chat: %w", err)
	}

	if result.RowsAffected == 0 {
		if err := query.Create(&model.TgChat{SourceID: chatID, Alert2Enabled: true, Alert2Date: date}).Error; err != nil {
			return fmt.Errorf("create new chat: %w", err)
		}
	}

	return nil
}
