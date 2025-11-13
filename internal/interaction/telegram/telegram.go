package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	tg "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/nicksnyder/go-i18n/v2/i18n"

	"goldie/internal/interaction/telegram/calendar"
	"goldie/internal/model"
)

var ErrWrongNumberOfArguments = fmt.Errorf("wrong number of arguments")

type PricesRepository interface {
	GetLatestPrices(ctx context.Context) ([]*model.GoldPrice, error)
	GetFirstPriceDate(ctx context.Context) (time.Time, error)
}

type ChatsRepository interface {
	EnableAlert1(ctx context.Context, chatID int64) error
	EnableAlert2(ctx context.Context, chatID int64, date time.Time) error
}

type Interaction struct {
	logger           *slog.Logger
	TgBot            *tg.Bot
	cal              *calendar.Calendar
	bundle           *i18n.Bundle
	pricesRepository PricesRepository
	chatsRepository  ChatsRepository
}

func NewInteraction(logger *slog.Logger, token string, client tg.HttpClient, bundle *i18n.Bundle, pricesRepository PricesRepository, chatsRepository ChatsRepository) *Interaction {
	cnt := &Interaction{
		logger:           logger.With("component", "telegram"),
		bundle:           bundle,
		pricesRepository: pricesRepository,
		chatsRepository:  chatsRepository,
	}

	opts := []tg.Option{
		tg.WithHTTPClient(time.Minute, client),
		tg.WithSkipGetMe(),
		tg.WithDefaultHandler(cnt.handler),
	}

	cal := calendar.New([]time.Weekday{time.Saturday, time.Sunday}, cnt.handlerAlert2SelectedDate, bundle)

	b, _ := tg.New(token, opts...)
	b.RegisterHandler(tg.HandlerTypeMessageText, "/start", tg.MatchTypeExact, cnt.handlerStart)
	b.RegisterHandler(tg.HandlerTypeMessageText, "/price", tg.MatchTypeExact, cnt.handlerPrice)
	b.RegisterHandler(tg.HandlerTypeMessageText, "/alert", tg.MatchTypeExact, cnt.handlerAlert)
	b.RegisterHandler(tg.HandlerTypeMessageText, "/alert1", tg.MatchTypeExact, cnt.handlerAlert1)
	b.RegisterHandler(tg.HandlerTypeMessageText, "/alert2", tg.MatchTypeExact, cnt.handlerAlert2)
	b.RegisterHandler(tg.HandlerTypeMessageText, "/help", tg.MatchTypeExact, cnt.handlerHelp)
	b.RegisterHandler(tg.HandlerTypeCallbackQueryData, calendar.Prefix, tg.MatchTypePrefix, cnt.handlerAlert2CalendarCallback)

	cnt.TgBot = b
	cnt.cal = cal
	return cnt
}

func (that *Interaction) Start(ctx context.Context) {
	that.TgBot.Start(ctx)
}

func (that *Interaction) handler(_ context.Context, _ *tg.Bot, update *models.Update) {
	log := that.logger.With("method", "handler")
	log.Info("handling message", "update", update)
}

// getLocalizer returns a localizer for the user.
func (that *Interaction) getLocalizer(languageCode string) *i18n.Localizer {
	if languageCode == "" {
		languageCode = "en"
	}

	return i18n.NewLocalizer(that.bundle, languageCode)
}

// renderLocaledMessage renders a localized message.
func (that *Interaction) renderLocaledMessage(languageCode string, messageID string, args ...string) (string, error) {
	if len(args)%2 != 0 {
		return "", ErrWrongNumberOfArguments
	}

	templateData := make(map[string]string, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		templateData[args[i]] = args[i+1]
	}

	text, err := that.getLocalizer(languageCode).Localize(&i18n.LocalizeConfig{MessageID: messageID, TemplateData: templateData})
	if err != nil {
		return "", fmt.Errorf("localize message: %w", err)
	}

	return text, nil
}

// sendLocaledMessage sends a localized message to the user.
func (that *Interaction) sendLocaledMessage(ctx context.Context, bot *tg.Bot, update *models.Update, messageID string, args ...string) (*models.Message, error) {
	languageCode := update.Message.From.LanguageCode

	text, err := that.renderLocaledMessage(languageCode, messageID, args...)
	if err != nil {
		return nil, fmt.Errorf("render localed message: %w", err)
	}

	msg, err := bot.SendMessage(ctx, &tg.SendMessageParams{ChatID: update.Message.Chat.ID, Text: text})
	if err != nil {
		return nil, fmt.Errorf("send message to telegram user: %w", err)
	}

	return msg, nil
}
