package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	telegramBot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/nicksnyder/go-i18n/v2/i18n"

	"goldie/internal/model"
)

var ErrWrongNumberOfArguments = fmt.Errorf("wrong number of arguments")

type PricesRepository interface {
	GetLatestPrices(ctx context.Context) ([]*model.GoldPrice, error)
}

type Interaction struct {
	logger           *slog.Logger
	TgBot            *telegramBot.Bot
	bundle           *i18n.Bundle
	pricesRepository PricesRepository

	mu              sync.RWMutex
	waitingMessages map[int64]struct{}
}

func NewInteraction(logger *slog.Logger, token string, client telegramBot.HttpClient, bundle *i18n.Bundle, pricesRepository PricesRepository) *Interaction {
	cnt := &Interaction{
		logger:           logger.With("component", "telegram"),
		bundle:           bundle,
		pricesRepository: pricesRepository,
	}

	opts := []telegramBot.Option{
		telegramBot.WithHTTPClient(time.Minute, client),
		telegramBot.WithSkipGetMe(),
		telegramBot.WithDefaultHandler(cnt.handler),
	}

	b, _ := telegramBot.New(token, opts...)
	b.RegisterHandler(telegramBot.HandlerTypeMessageText, "/start", telegramBot.MatchTypeExact, cnt.handlerStart)
	b.RegisterHandler(telegramBot.HandlerTypeMessageText, "/price", telegramBot.MatchTypeExact, cnt.handlerPrice)
	b.RegisterHandler(telegramBot.HandlerTypeMessageText, "/help", telegramBot.MatchTypeExact, cnt.handlerHelp)
	b.RegisterHandler(telegramBot.HandlerTypeMessageText, "/delete", telegramBot.MatchTypeExact, cnt.handlerDelete)

	cnt.TgBot = b
	return cnt
}

func (that *Interaction) Start(ctx context.Context) {
	that.TgBot.Start(ctx)
}

func (that *Interaction) handler(ctx context.Context, bot *telegramBot.Bot, update *models.Update) {
	log := that.logger.With("method", "handler", "user_id", update.Message.From.ID)

	log.Info("handling message", "text", update.Message.Text)
	that.mu.RLock()
	_, ok := that.waitingMessages[update.Message.From.ID]
	that.mu.RUnlock()

	if ok {
		that.handleWaitingMessage(ctx, bot, update)
		return
	}
}

func (that *Interaction) handleWaitingMessage(ctx context.Context, bot *telegramBot.Bot, update *models.Update) {
	log := that.logger.With("method", "handleWaitingMessage", "user_id", update.Message.From.ID)

	that.mu.Lock()
	delete(that.waitingMessages, update.Message.From.ID)
	that.mu.Unlock()

	// TODO: Implement login
	log.Info("handling waiting for login")
}

// getUserLocalizer returns a localizer for the user.
func (that *Interaction) getUserLocalizer(update *models.Update) *i18n.Localizer {
	lang := update.Message.From.LanguageCode // "en", "ru", etc.
	if lang == "" {
		lang = "en"
	}

	return i18n.NewLocalizer(that.bundle, lang)
}

// renderLocaledMessage renders a localized message.
func (that *Interaction) renderLocaledMessage(update *models.Update, messageID string, args ...string) (string, error) {
	if len(args)%2 != 0 {
		return "", ErrWrongNumberOfArguments
	}

	templateData := make(map[string]string, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		templateData[args[i]] = args[i+1]
	}

	text, err := that.getUserLocalizer(update).Localize(&i18n.LocalizeConfig{MessageID: messageID, TemplateData: templateData})
	if err != nil {
		return "", fmt.Errorf("localize message: %w", err)
	}

	return text, nil
}

// sendLocaledMessage sends a localized message to the user.
func (that *Interaction) sendLocaledMessage(ctx context.Context, bot *telegramBot.Bot, update *models.Update, messageID string, args ...string) (*models.Message, error) {
	text, err := that.renderLocaledMessage(update, messageID, args...)
	if err != nil {
		return nil, fmt.Errorf("render localed message: %w", err)
	}

	msg, err := bot.SendMessage(ctx, &telegramBot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: text})
	if err != nil {
		return nil, fmt.Errorf("send message to telegram user: %w", err)
	}

	return msg, nil
}
