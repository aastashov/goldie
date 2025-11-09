package prices

import (
	"context"
	"fmt"

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
