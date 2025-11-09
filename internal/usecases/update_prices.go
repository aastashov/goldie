package usecases

import (
	"context"
	"log/slog"
	"time"

	"goldie/internal/interaction/nbkr"
	"goldie/internal/model"
)

type Repository interface {
	SavePrices(ctx context.Context, prices []*model.GoldPrice) error
}

type Interaction interface {
	GetGoldPrices(ctx context.Context, beginDate time.Time, endDate time.Time) ([]nbkr.GoldPrice, error)
}

type UpdatePricesUsecase struct {
	logger      *slog.Logger
	repository  Repository
	interaction Interaction
}

func NewUpdatePricesUsecase(logger *slog.Logger, repository Repository, interaction Interaction) *UpdatePricesUsecase {
	return &UpdatePricesUsecase{logger: logger.With("component", "update_prices"), repository: repository, interaction: interaction}
}

func (that *UpdatePricesUsecase) UpdatePrices(ctx context.Context) {
	log := that.logger.With("method", "UpdatePrices")

	endDate := time.Now()
	startDate := time.Date(endDate.Year()-1, endDate.Month(), endDate.Day(), 0, 0, 0, 0, time.UTC)

	prices, err := that.interaction.GetGoldPrices(ctx, startDate, endDate)
	if err != nil {
		log.Error("failed to get gold prices", "error", err)
		return
	}

	dbPrices := make([]*model.GoldPrice, len(prices))
	for i, price := range prices {
		dbPrices[i] = &model.GoldPrice{
			Date:          price.Date,
			Weight:        price.Weight,
			PurchasePrice: price.PurchasePrice,
			SellPrice:     price.SellPrice,
		}
	}

	if err = that.repository.SavePrices(ctx, dbPrices); err != nil {
		log.Error("failed to save prices", "error", err)
		return
	}
}
