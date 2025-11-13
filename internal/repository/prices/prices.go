package prices

import (
	"context"
	"fmt"
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

// SavePrices saves prices in the database using upsert.
func (that *Repository) SavePrices(ctx context.Context, prices []*model.GoldPrice) error {
	query := that.db.WithContext(ctx).Clauses(
		clause.OnConflict{
			Columns: []clause.Column{{Name: "date"}, {Name: "weight"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"purchase_price": gorm.Expr("EXCLUDED.purchase_price"),
				"sell_price":     gorm.Expr("EXCLUDED.sell_price"),
			}),
		},
		clause.Returning{
			// We should return the inserted fields from the upsert query to compare and count the number of objects inserted.
			Columns: []clause.Column{{Name: "created_at"}, {Name: "updated_at"}},
		},
	)

	if err := query.Create(prices).Error; err != nil {
		return fmt.Errorf("upsert prices in database: %w", err)
	}

	return nil
}

// GetLatestPrices returns the latest prices from the database.
func (that *Repository) GetLatestPrices(ctx context.Context) ([]*model.GoldPrice, error) {
	var prices []*model.GoldPrice

	query := that.db.WithContext(ctx).Where("date = (SELECT MAX(date) FROM gold_prices)").Order("weight asc")
	if err := query.Find(&prices).Error; err != nil {
		return nil, fmt.Errorf("get prices from database: %w", err)
	}

	return prices, nil
}

// ExistsFirstPrice checks if the first price exists in the database.
func (that *Repository) ExistsFirstPrice(ctx context.Context, date time.Time) (bool, error) {
	var prices []*model.GoldPrice

	query := that.db.WithContext(ctx).Where("date = ?", date)
	if err := query.Find(&prices).Error; err != nil {
		return false, fmt.Errorf("check if first price exists in database: %w", err)
	}

	return len(prices) > 0, nil
}

// GetFirstPriceDate returns the date of the first price.
func (that *Repository) GetFirstPriceDate(ctx context.Context) (time.Time, error) {
	var prices []*model.GoldPrice

	query := that.db.WithContext(ctx).Order("date asc")
	if err := query.Find(&prices).Error; err != nil {
		return time.Time{}, fmt.Errorf("get first price date from database: %w", err)
	}

	if len(prices) == 0 {
		return time.Time{}, fmt.Errorf("no prices found")
	}

	return prices[0].Date, nil
}
