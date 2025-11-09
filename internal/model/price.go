package model

import "time"

// GoldPrice describes a gold price.
// Unique index are Date and Weight together.
type GoldPrice struct {
	Date          string    `gorm:"column:date;uniqueIndex:date_weight"`
	Weight        float64   `gorm:"column:weight;uniqueIndex:date_weight"`
	PurchasePrice float64   `gorm:"column:purchase_price"`
	SellPrice     float64   `gorm:"column:sell_price"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (*GoldPrice) TableName() string {
	return "gold_prices"
}
