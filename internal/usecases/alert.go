package usecases

import (
	"context"
	"log/slog"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/sync/errgroup"

	"goldie/internal/model"
)

const ParallelSendLimit = 100

type AlertPricesRepository interface {
	GetLatestPrices(ctx context.Context) ([]*model.GoldPrice, error)
}

type AlertChatsRepository interface {
	FetchChatsWithBuyingPrices(ctx context.Context) ([]*model.TgChat, error)
}

type AlertTGIntegration interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
	PricesToString(languageCode string, prices []*model.GoldPrice) string
	PricesWithGainToString(languageCode string, prices []*model.GoldPrice, buyingPrices []*model.GoldPrice) string
}

type AlertUseCase struct {
	logger           *slog.Logger
	bundle           *i18n.Bundle
	loc              *time.Location
	pricesRepository AlertPricesRepository
	chatsRepository  AlertChatsRepository
	tgIntegration    AlertTGIntegration
}

func NewAlertUseCase(logger *slog.Logger, bundle *i18n.Bundle, loc *time.Location, pricesRepository AlertPricesRepository, chatsRepository AlertChatsRepository, tgIntegration AlertTGIntegration) *AlertUseCase {
	return &AlertUseCase{logger: logger.With("component", "alert1"), bundle: bundle, loc: loc, pricesRepository: pricesRepository, chatsRepository: chatsRepository, tgIntegration: tgIntegration}
}

func (that *AlertUseCase) Run(ctx context.Context) {
	log := that.logger.With("method", "Run")

	prices, err := that.pricesRepository.GetLatestPrices(ctx)
	if err != nil {
		log.Error("failed to get prices", "error", err)
		return
	}

	if len(prices) == 0 {
		log.Info("no prices found")
		return
	}

	chats, err := that.chatsRepository.FetchChatsWithBuyingPrices(ctx)
	if err != nil {
		log.Error("failed to get chats", "error", err)
		return
	}

	// Prepare localized texts for all languages
	localizedAlert1Lookup := make(map[string]string)
	for _, chat := range chats {
		if !chat.Alert1Enabled {
			continue
		}

		languageCode := chat.Language
		if languageCode == "" {
			languageCode = "en"
		}

		_, exists := localizedAlert1Lookup[languageCode]
		if !exists {
			localizedAlert1Lookup[languageCode] = that.tgIntegration.PricesToString(languageCode, prices)
		}
	}

	parallelSend, parallelSendCtx := errgroup.WithContext(ctx)
	parallelSend.SetLimit(ParallelSendLimit)

	for _, chat := range chats {
		parallelSend.Go(func() error {
			if chat.Alert1Enabled {
				textForAlert1 := localizedAlert1Lookup[chat.Language]
				if textForAlert1 == "" {
					textForAlert1 = localizedAlert1Lookup["en"]
				}

				if err = that.tgIntegration.SendMessage(parallelSendCtx, chat.SourceID, textForAlert1); err != nil {
					log.Error("failed to send alert1", "error", err, "chat_id", chat.SourceID)
					return nil
				}
			}

			if chat.Alert2Enabled && len(chat.BuyingPrices) > 0 {
				languageCode := chat.Language
				if languageCode == "" {
					languageCode = "en"
				}

				textForAlert2 := that.tgIntegration.PricesWithGainToString(languageCode, prices, chat.BuyingPrices)
				if err = that.tgIntegration.SendMessage(parallelSendCtx, chat.SourceID, textForAlert2); err != nil {
					log.Error("failed to send alert2", "error", err, "chat_id", chat.SourceID)
					return nil
				}
			}
			return nil
		})
	}

	// Wait for all parallel sends to finish
	_ = parallelSend.Wait()
}
