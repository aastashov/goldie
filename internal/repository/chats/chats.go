package chats

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
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

// DisableAlerts disables all alerts for the chat.
func (that *Repository) DisableAlerts(ctx context.Context, chatID int64) error {
	query := that.db.WithContext(ctx).Model(&model.TgChat{}).Where("source_id = ?", chatID)

	result := query.Updates(map[string]interface{}{"alert1": false, "alert2": false, "alert2_date": time.Time{}, "updated_at": time.Now()})
	if err := result.Error; err != nil {
		return fmt.Errorf("update existing chat: %w", err)
	}

	if result.RowsAffected == 0 {
		if err := query.Create(&model.TgChat{SourceID: chatID}).Error; err != nil {
			return fmt.Errorf("create new chat: %w", err)
		}
	}

	return nil
}

// DeleteChat removes chat record and any stored data.
func (that *Repository) DeleteChat(ctx context.Context, chatID int64) error {
	query := that.db.WithContext(ctx).Model(&model.TgChat{}).Where("source_id = ?", chatID)

	if err := query.Delete(&model.TgChat{}).Error; err != nil {
		return fmt.Errorf("delete chat: %w", err)
	}

	return nil
}

// SetLanguage sets chat language.
func (that *Repository) SetLanguage(ctx context.Context, chatID int64, language string) error {
	query := that.db.WithContext(ctx).Model(&model.TgChat{}).Where("source_id = ?", chatID)

	result := query.Updates(map[string]interface{}{"language": language, "updated_at": time.Now()})
	if err := result.Error; err != nil {
		return fmt.Errorf("update existing chat language: %w", err)
	}

	if result.RowsAffected == 0 {
		if err := query.Create(&model.TgChat{SourceID: chatID, Language: language}).Error; err != nil {
			return fmt.Errorf("create new chat with language: %w", err)
		}
	}

	return nil
}

// GetLanguage returns chat language.
func (that *Repository) GetLanguage(ctx context.Context, chatID int64) (string, error) {
	var chat model.TgChat

	err := that.db.WithContext(ctx).Model(&model.TgChat{}).Select("language").Where("source_id = ?", chatID).First(&chat).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}

		return "", fmt.Errorf("get chat language: %w", err)
	}

	return chat.Language, nil
}

// GetChat returns chat entity if exists.
func (that *Repository) GetChat(ctx context.Context, chatID int64) (*model.TgChat, error) {
	var chat model.TgChat

	err := that.db.WithContext(ctx).Model(&model.TgChat{}).Where("source_id = ?", chatID).First(&chat).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("get chat: %w", err)
	}

	return &chat, nil
}

func (that *Repository) FetchChatsWithBuyingPrices(ctx context.Context) ([]*model.TgChat, error) {
	var chats []*model.TgChat

	query := that.db.WithContext(ctx).Model(&model.TgChat{}).Where("alert1 = true OR alert2 = true")
	if err := query.Find(&chats).Error; err != nil {
		return nil, fmt.Errorf("fetch chats with buying prices from database: %w", err)
	}

	datesToFetch := make(map[time.Time]struct{}, len(chats))
	for _, chat := range chats {
		if chat.Alert2Enabled && !chat.Alert2Date.IsZero() {
			datesToFetch[chat.Alert2Date] = struct{}{}
		}
	}

	var prices []*model.GoldPrice
	goldPricesQuery := that.db.WithContext(ctx).Model(&model.GoldPrice{}).Where("date IN (?)", slices.Collect(maps.Keys(datesToFetch)))
	if err := goldPricesQuery.Find(&prices).Error; err != nil {
		return nil, fmt.Errorf("fetch prices from database: %w", err)
	}

	pricesMap := make(map[time.Time][]*model.GoldPrice, len(prices))
	for _, price := range prices {
		pricesMap[price.Date] = append(pricesMap[price.Date], price)
	}

	for _, chat := range chats {
		if chat.Alert2Enabled {
			chat.BuyingPrices = pricesMap[chat.Alert2Date]
		}
	}

	return chats, nil
}
