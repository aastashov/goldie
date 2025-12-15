package chats

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

func (that *Repository) CreateAlert2Subscription(ctx context.Context, chatID int64, date time.Time) error {
	return that.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := dropLegacyAlert2ChatSourceColumn(tx); err != nil {
			return err
		}

		chat, err := ensureChatExists(tx, chatID)
		if err != nil {
			return err
		}

		alert2 := &model.TgChatAlert2{ChatID: chat.ID, PurchaseDate: date}
		query := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "chat_id"}, {Name: "purchase_date"}},
			DoNothing: true,
		})

		if err = query.Create(alert2).Error; err != nil {
			return fmt.Errorf("create alert2 subscription: %w", err)
		}

		return nil
	})
}

// DisableAlerts disables all alerts for the chat.
func (that *Repository) DisableAlerts(ctx context.Context, chatID int64) error {
	return that.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		chat, err := ensureChatExists(tx, chatID)
		if err != nil {
			return err
		}

		query := tx.Model(&model.TgChat{}).Where("id = ?", chat.ID)
		if err = query.Updates(map[string]interface{}{"alert1": false, "updated_at": time.Now()}).Error; err != nil {
			return fmt.Errorf("update existing chat: %w", err)
		}

		query = tx.Where("chat_id = ?", chat.ID)
		if err = query.Delete(&model.TgChatAlert2{}).Error; err != nil {
			return fmt.Errorf("delete alert2 subscriptions: %w", err)
		}

		return nil
	})
}

// DeleteChat removes chat record and any stored data.
func (that *Repository) DeleteChat(ctx context.Context, chatID int64) error {
	return that.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var chat model.TgChat
		if err := tx.Where("source_id = ?", chatID).First(&chat).Error; err == nil {
			if err = tx.Where("chat_id = ?", chat.ID).Delete(&model.TgChatAlert2{}).Error; err != nil {
				return fmt.Errorf("delete alert2 subscriptions: %w", err)
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("fetch chat: %w", err)
		}

		if err := tx.Where("source_id = ?", chatID).Delete(&model.TgChat{}).Error; err != nil {
			return fmt.Errorf("delete chat: %w", err)
		}

		return nil
	})
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

func (that *Repository) ListAlert2Subscriptions(ctx context.Context, chatID int64) ([]*model.TgChatAlert2, error) {
	var alerts []*model.TgChatAlert2

	query := that.db.WithContext(ctx).Model(&model.TgChatAlert2{}).
		Joins("JOIN tg_chats ON tg_chats.id = tg_chat_alert2.chat_id").
		Where("tg_chats.source_id = ?", chatID).
		Order("tg_chat_alert2.purchase_date ASC")
	if err := query.Find(&alerts).Error; err != nil {
		return nil, fmt.Errorf("list alert2 subscriptions: %w", err)
	}

	return alerts, nil
}

func (that *Repository) ListAlert2SubscriptionsPaged(ctx context.Context, chatID int64, limit, offset int) ([]*model.TgChatAlert2, int64, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	baseQuery := that.db.WithContext(ctx).Model(&model.TgChatAlert2{}).
		Joins("JOIN tg_chats ON tg_chats.id = tg_chat_alert2.chat_id").
		Where("tg_chats.source_id = ?", chatID)

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count alert2 subscriptions: %w", err)
	}

	if total == 0 {
		return nil, 0, nil
	}

	var alerts []*model.TgChatAlert2
	if err := that.db.WithContext(ctx).Model(&model.TgChatAlert2{}).
		Joins("JOIN tg_chats ON tg_chats.id = tg_chat_alert2.chat_id").
		Where("tg_chats.source_id = ?", chatID).
		Order("tg_chat_alert2.purchase_date DESC").
		Offset(offset).
		Limit(limit).
		Preload("Chat").
		Find(&alerts).Error; err != nil {
		return nil, 0, fmt.Errorf("list alert2 subscriptions paged: %w", err)
	}

	return alerts, total, nil
}

func (that *Repository) DeleteAlert2Subscription(ctx context.Context, chatID int64, subscriptionID int64) error {
	return that.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := dropLegacyAlert2ChatSourceColumn(tx); err != nil {
			return err
		}

		var alert model.TgChatAlert2
		if err := tx.Joins("JOIN tg_chats ON tg_chats.id = tg_chat_alert2.chat_id").
			Where("tg_chat_alert2.id = ? AND tg_chats.source_id = ?", subscriptionID, chatID).
			First(&alert).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return fmt.Errorf("fetch alert2 subscription: %w", err)
		}

		if err := tx.Delete(&model.TgChatAlert2{}, alert.ID).Error; err != nil {
			return fmt.Errorf("delete alert2 subscription: %w", err)
		}

		return nil
	})
}

func (that *Repository) FetchChatsWithBuyingPrices(ctx context.Context) ([]*model.TgChat, error) {
	var chats []*model.TgChat

	query := that.db.WithContext(ctx).Model(&model.TgChat{}).Where("alert1 = true OR EXISTS (SELECT 1 FROM tg_chat_alert2 WHERE tg_chat_alert2.chat_id = tg_chats.id)")
	if err := query.Find(&chats).Error; err != nil {
		return nil, fmt.Errorf("fetch chats: %w", err)
	}

	if len(chats) == 0 {
		return chats, nil
	}

	chatByID := make(map[int64]*model.TgChat, len(chats))
	chatIDs := make([]int64, 0, len(chats))
	for _, chat := range chats {
		chatByID[chat.ID] = chat
		chatIDs = append(chatIDs, chat.ID)
	}

	var alerts []*model.TgChatAlert2
	if err := that.db.WithContext(ctx).Model(&model.TgChatAlert2{}).Joins("Chat").Where("chat_id IN ?", chatIDs).Find(&alerts).Error; err != nil {
		return nil, fmt.Errorf("fetch alert2 subscriptions: %w", err)
	}

	datesToFetch := make(map[time.Time]struct{}, len(alerts))
	for _, alert := range alerts {
		datesToFetch[alert.PurchaseDate] = struct{}{}
	}

	var prices []*model.GoldPrice
	if len(datesToFetch) > 0 {
		goldPricesQuery := that.db.WithContext(ctx).Model(&model.GoldPrice{}).Where("date IN (?)", slices.Collect(maps.Keys(datesToFetch)))
		if err := goldPricesQuery.Find(&prices).Error; err != nil {
			return nil, fmt.Errorf("fetch prices from database: %w", err)
		}
	}

	pricesMap := make(map[time.Time][]*model.GoldPrice, len(prices))
	for _, price := range prices {
		pricesMap[price.Date] = append(pricesMap[price.Date], price)
	}

	for _, alert := range alerts {
		alert.BuyingPrices = pricesMap[alert.PurchaseDate]
		if chat := chatByID[alert.ChatID]; chat != nil {
			chat.Alerts2 = append(chat.Alerts2, alert)
		}
	}

	return chats, nil
}

func ensureChatExists(tx *gorm.DB, sourceID int64) (*model.TgChat, error) {
	var chat model.TgChat

	if err := tx.Where("source_id = ?", sourceID).First(&chat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			chat = model.TgChat{SourceID: sourceID}
			if err = tx.Create(&chat).Error; err != nil {
				return nil, fmt.Errorf("create new chat: %w", err)
			}

			return &chat, nil
		}

		return nil, fmt.Errorf("get chat: %w", err)
	}

	return &chat, nil
}

func dropLegacyAlert2ChatSourceColumn(tx *gorm.DB) error {
	migrator := tx.Migrator()
	if migrator.HasColumn(&model.TgChatAlert2{}, "chat_source_id") {
		if err := migrator.DropColumn(&model.TgChatAlert2{}, "chat_source_id"); err != nil {
			return fmt.Errorf("drop chat_source_id column: %w", err)
		}
	}
	return nil
}
