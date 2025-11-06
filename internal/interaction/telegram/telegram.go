package telegram

import (
	"context"
	"log/slog"
	"sync"

	telegramBot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Interaction struct {
	logger *slog.Logger
	tgBot  *telegramBot.Bot

	mu              sync.RWMutex
	waitingMessages map[int64]struct{}
}

func NewInteraction(logger *slog.Logger, token string) *Interaction {
	cnt := &Interaction{
		logger: logger.With("component", "telegram"),
	}

	opts := []telegramBot.Option{
		telegramBot.WithSkipGetMe(),
		telegramBot.WithDefaultHandler(cnt.handler),
	}

	b, _ := telegramBot.New(token, opts...)
	b.RegisterHandler(telegramBot.HandlerTypeMessageText, "/start", telegramBot.MatchTypeExact, cnt.handlerStart)
	b.RegisterHandler(telegramBot.HandlerTypeMessageText, "/about", telegramBot.MatchTypeExact, cnt.handlerAbout)
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
