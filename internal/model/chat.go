package model

import (
	"time"

	"goldie/internal/config"
)

// TgChat - represents a Telegram chat.
type TgChat struct {
	ID            int64           `gorm:"column:id;primaryKey"`
	SourceID      int64           `gorm:"column:source_id;uniqueIndex"`
	Language      string          `gorm:"column:language"` // en, ru
	Alert1Enabled bool            `gorm:"column:alert1"`
	CreatedAt     time.Time       `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time       `gorm:"column:updated_at;autoUpdateTime"`
	Alerts2       []*TgChatAlert2 `gorm:"-"`
}

func (*TgChat) TableName() string {
	return "tg_chats"
}

func (that *TgChat) GetLanguageCode() string {
	if that.Language != "" {
		return that.Language
	}

	return config.DefaultLanguageCode
}
