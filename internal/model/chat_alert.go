package model

import (
	"time"
)

// TgChatAlert2 represents alert2 subscription for a chat.
type TgChatAlert2 struct {
	ID           int64        `gorm:"column:id;primaryKey"`
	ChatID       int64        `gorm:"column:chat_id;not null;uniqueIndex:idx_alert2_chat_date"`
	Chat         TgChat       `gorm:"foreignKey:ChatID;references:ID;constraint:OnDelete:CASCADE"`
	PurchaseDate time.Time    `gorm:"column:purchase_date;not null;uniqueIndex:idx_alert2_chat_date"`
	CreatedAt    time.Time    `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time    `gorm:"column:updated_at;autoUpdateTime"`
	BuyingPrices []*GoldPrice `gorm:"-"`
}

func (*TgChatAlert2) TableName() string {
	return "tg_chat_alert2"
}
