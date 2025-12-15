package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	tg "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	tgModels "github.com/go-telegram/bot/models"
	"github.com/nicksnyder/go-i18n/v2/i18n"

	"goldie/internal/config"
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
	DisableAlerts(ctx context.Context, chatID int64) error
	SetLanguage(ctx context.Context, chatID int64, language string) error
	GetLanguage(ctx context.Context, chatID int64) (string, error)
}

type Interaction struct {
	logger           *slog.Logger
	TgBot            *tg.Bot
	cal              *calendar.Calendar
	bundle           *i18n.Bundle
	pricesRepository PricesRepository
	chatsRepository  ChatsRepository
	supportedLangs   map[string]struct{}
}

const languageCallbackPrefix = "lang:"

var botCommandDefinitions = []struct {
	command           string
	descriptionLocale string
}{
	{command: "start", descriptionLocale: "command.start.description"},
	{command: "price", descriptionLocale: "command.price.description"},
	{command: "alert", descriptionLocale: "command.alert.description"},
	{command: "help", descriptionLocale: "command.help.description"},
	{command: "info", descriptionLocale: "command.info.description"},
	{command: "settings", descriptionLocale: "command.settings.description"},
	{command: "stop", descriptionLocale: "command.stop.description"},
}

func NewInteraction(logger *slog.Logger, token string, client tg.HttpClient, bundle *i18n.Bundle, pricesRepository PricesRepository, chatsRepository ChatsRepository) *Interaction {
	supportedLangs := make(map[string]struct{})
	for _, tag := range bundle.LanguageTags() {
		supportedLangs[tag.String()] = struct{}{}
	}

	cnt := &Interaction{
		logger:           logger.With("component", "telegram"),
		bundle:           bundle,
		pricesRepository: pricesRepository,
		chatsRepository:  chatsRepository,
		supportedLangs:   supportedLangs,
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
	b.RegisterHandler(tg.HandlerTypeMessageText, "/stop", tg.MatchTypeExact, cnt.handlerStop)
	b.RegisterHandler(tg.HandlerTypeCallbackQueryData, languageCallbackPrefix, tg.MatchTypePrefix, cnt.handlerLanguageSelection)
	b.RegisterHandler(tg.HandlerTypeCallbackQueryData, calendar.Prefix, tg.MatchTypePrefix, cnt.handlerAlert2CalendarCallback)

	cnt.TgBot = b
	cnt.cal = cal
	return cnt
}

func (that *Interaction) Start(ctx context.Context) {
	that.setMyCommands(ctx)
	that.TgBot.Start(ctx)
}

func (that *Interaction) SendMessage(ctx context.Context, chatID int64, text string) error {
	_, err := that.TgBot.SendMessage(ctx, &tg.SendMessageParams{ChatID: chatID, Text: text, ParseMode: models.ParseModeHTML})
	return err
}

func (that *Interaction) handler(_ context.Context, _ *tg.Bot, update *models.Update) {
	log := that.logger.With("method", "handler")
	log.Info("handling message", "update", update)
}

func (that *Interaction) getLanguageCode(ctx context.Context, chat tgModels.Chat, from *tgModels.User) string {
	language, _ := that.chatsRepository.GetLanguage(ctx, chat.ID)
	if language != "" {
		return language
	}

	if from.LanguageCode != "" {
		return from.LanguageCode
	}

	return config.DefaultLanguageCode
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

	text, err := i18n.NewLocalizer(that.bundle, languageCode).Localize(&i18n.LocalizeConfig{MessageID: messageID, TemplateData: templateData})
	if err != nil {
		return "", fmt.Errorf("localize message: %w", err)
	}

	return text, nil
}

// sendLocaledMessage sends a localized message to the user.
func (that *Interaction) sendLocaledMessage(ctx context.Context, bot *tg.Bot, update *models.Update, messageID string, args ...string) (*models.Message, error) {
	languageCode := that.getLanguageCode(ctx, update.Message.Chat, update.Message.From)

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

func (that *Interaction) setMyCommands(ctx context.Context) {
	log := that.logger.With("method", "setMyCommands")
	var defaultCommands []models.BotCommand

	for _, tag := range that.bundle.LanguageTags() {
		languageCode := tag.String()
		log = log.With("language", languageCode)

		localizedCommands := make([]models.BotCommand, 0, len(botCommandDefinitions))
		for _, definition := range botCommandDefinitions {
			log = log.With("tg.command", definition.command)

			description, err := that.renderLocaledMessage(languageCode, definition.descriptionLocale)
			if err != nil {
				log.Error("failed to render command description", "error", err)
				continue
			}

			localizedCommands = append(localizedCommands, models.BotCommand{Command: definition.command, Description: description})
		}

		if len(localizedCommands) == 0 {
			continue
		}

		if _, err := that.TgBot.SetMyCommands(ctx, &tg.SetMyCommandsParams{Commands: localizedCommands, LanguageCode: languageCode}); err != nil {
			log.Error("failed to set bot commands", "error", err)
			continue
		}

		if languageCode == config.DefaultLanguageCode {
			defaultCommands = localizedCommands
		}
	}

	if len(defaultCommands) == 0 {
		return
	}

	if _, err := that.TgBot.SetMyCommands(ctx, &tg.SetMyCommandsParams{Commands: defaultCommands}); err != nil {
		log.Error("failed to set default bot commands", "error", err)
	}
}
