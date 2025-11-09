package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	telegramBot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

var ErrWrongNumberOfArguments = fmt.Errorf("wrong number of arguments")

type Interaction struct {
	logger *slog.Logger
	tgBot  *telegramBot.Bot
	bundle *i18n.Bundle

	mu              sync.RWMutex
	waitingMessages map[int64]struct{}
}

func NewInteraction(logger *slog.Logger, token string, client *http.Client, bundle *i18n.Bundle) *Interaction {
	cnt := &Interaction{
		logger: logger.With("component", "telegram"),
		bundle: bundle,
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

	cnt.tgBot = b
	return cnt
}

func (that *Interaction) Start(ctx context.Context) {
	that.tgBot.Start(ctx)
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

func (that *Interaction) renderLocaledMessage(update *models.Update, messageID string, args ...string) (string, error) {
	if len(args)%2 != 0 {
		return "", ErrWrongNumberOfArguments
	}

	templateData := make(map[string]string, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		templateData[args[i]] = args[i+1]
	}

	that.logger.With("lang", update.Message.From.LanguageCode).Info("render localed message")

	lang := update.Message.From.LanguageCode // "en", "ru", etc.
	if lang == "" {
		lang = "en"
	}

	localizer := i18n.NewLocalizer(that.bundle, lang)
	return localizer.Localize(&i18n.LocalizeConfig{MessageID: messageID, TemplateData: templateData})
}
