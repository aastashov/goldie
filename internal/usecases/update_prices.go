package usecases

import (
	"context"
	"log/slog"
	"time"

	"goldie/internal/interaction/nbkr"
	"goldie/internal/model"
)

const FirstPriceDate = "2015-07-05"

type UpdatePricesRepository interface {
	SavePrices(ctx context.Context, prices []*model.GoldPrice) error
	ExistsFirstPrice(ctx context.Context, date time.Time) (bool, error)
}

type UpdatePricesInteraction interface {
	GetGoldPrices(ctx context.Context, beginDate time.Time, endDate time.Time) ([]nbkr.GoldPrice, error)
}

type UpdatePricesUseCase struct {
	logger      *slog.Logger
	repository  UpdatePricesRepository
	interaction UpdatePricesInteraction
	loc         *time.Location
}

func NewUpdatePricesUseCase(logger *slog.Logger, repository UpdatePricesRepository, interaction UpdatePricesInteraction, loc *time.Location) *UpdatePricesUseCase {
	return &UpdatePricesUseCase{logger: logger.With("component", "update_prices"), repository: repository, interaction: interaction, loc: loc}
}

func (that *UpdatePricesUseCase) UpdatePrices(ctx context.Context) {
	log := that.logger.With("method", "UpdatePrices")

	endDate := time.Now()
	startDate := time.Date(endDate.Year()-1, endDate.Month(), endDate.Day(), 0, 0, 0, 0, that.loc)

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

func (that *UpdatePricesUseCase) FirstImport(ctx context.Context) {
	log := that.logger.With("method", "UpdatePrices")

	// The start date is the first date of the first price from the NBKR
	startDate := time.Date(2015, 1, 1, 0, 0, 0, 0, that.loc)
	endDate := startDate.AddDate(1, 0, 0)

	// Need to check existing data from the first date
	firstPriceDate, _ := time.Parse("2006-01-02", FirstPriceDate)
	exists, err := that.repository.ExistsFirstPrice(ctx, firstPriceDate)
	if err != nil {
		log.Error("failed to check if first price exists", "error", err)
		return
	}

	if exists {
		// The first price already exists, we don't need to import it
		return
	}

	for {
		prices, err := that.interaction.GetGoldPrices(ctx, startDate, endDate)
		if err != nil {
			log.Error("failed to get gold prices", "error", err)
			return
		}

		if len(prices) == 0 {
			// Imported all the data
			break
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

		startDate = endDate
		endDate = startDate.AddDate(1, 0, 0)
	}
}
