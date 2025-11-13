package model

import "time"

// TgChat - represents a Telegram chat.
type TgChat struct {
	ID            int64        `gorm:"column:id;primaryKey"`
	SourceID      int64        `gorm:"column:source_id;uniqueIndex"`
	Language      string       `gorm:"column:language"` // en, ru
	Alert1Enabled bool         `gorm:"column:alert1"`
	Alert2Enabled bool         `gorm:"column:alert2"`
	Alert2Date    time.Time    `gorm:"column:alert2_date"`
	CreatedAt     time.Time    `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time    `gorm:"column:updated_at;autoUpdateTime"`
	BuyingPrices  []*GoldPrice `gorm:"-"`
}

func (*TgChat) TableName() string {
	return "tg_chats"
}
