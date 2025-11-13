package telegram

import (
	"context"
	"time"

	tg "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (that *Interaction) handlerStart(ctx context.Context, bot *tg.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerStart", "user_id", update.Message.From.ID, "language", update.Message.From.LanguageCode)

	if _, err := that.sendLocaledMessage(ctx, bot, update, "startWelcomeMessage"); err != nil {
		log.Error("failed to send message", "error", err)
		return
	}
}

func (that *Interaction) handlerPrice(ctx context.Context, bot *tg.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerPrice", "user_id", update.Message.From.ID)

	prices, err := that.pricesRepository.GetLatestPrices(ctx)
	if err != nil {
		log.Error("failed to get prices", "error", err)
		return
	}

	if len(prices) == 0 {
		if _, err = that.sendLocaledMessage(ctx, bot, update, "noPricesMessage"); err != nil {
			log.Error("failed to send message", "error", err)
			return
		}

		return
	}

	languageCode := update.Message.From.LanguageCode

	text := that.PricesToString(languageCode, prices)
	if _, err = bot.SendMessage(ctx, &tg.SendMessageParams{ChatID: update.Message.Chat.ID, Text: text, ParseMode: models.ParseModeHTML}); err != nil {
		log.Error("error sending message", "error", err)
		return
	}
}

func (that *Interaction) handlerAlert(ctx context.Context, bot *tg.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerAlert", "user_id", update.Message.From.ID)

	_, err := that.sendLocaledMessage(ctx, bot, update, "alertMessage")
	if err != nil {
		log.Error("error sending message", "error", err)
		return
	}
}

func (that *Interaction) handlerAlert1(ctx context.Context, bot *tg.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerAlert1", "user_id", update.Message.From.ID)

	if err := that.chatsRepository.EnableAlert1(ctx, update.Message.Chat.ID); err != nil {
		log.Error("failed to create alert", "error", err)
		return
	}

	if _, err := that.sendLocaledMessage(ctx, bot, update, "createAlert1Message"); err != nil {
		log.Error("error sending message", "error", err)
		return
	}
}

func (that *Interaction) handlerAlert2(ctx context.Context, bot *tg.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerAlert2", "user_id", update.Message.From.ID)

	firstCalendarDate, err := that.pricesRepository.GetFirstPriceDate(ctx)
	if err != nil {
		log.Error("failed to get first price date", "error", err)
		return
	}

	languageCode := update.Message.From.LanguageCode
	if languageCode == "" {
		languageCode = "en"
	}

	if err = that.cal.SendCalendar(ctx, bot, languageCode, update.Message.Chat.ID, firstCalendarDate, time.Now()); err != nil {
		log.Error("failed to send calendar", "error", err)
		return
	}
}

func (that *Interaction) handlerAlert2CalendarCallback(ctx context.Context, bot *tg.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerAlert2CalendarCallback", "chat_id", update.CallbackQuery.Message.Message.Chat.ID)

	firstCalendarDate, err := that.pricesRepository.GetFirstPriceDate(ctx)
	if err != nil {
		log.Error("failed to get first price date", "error", err)
		return
	}

	languageCode := update.CallbackQuery.Message.Message.From.LanguageCode
	if languageCode == "" {
		languageCode = "en"
	}

	if err = that.cal.HandleCallback(ctx, bot, languageCode, update, firstCalendarDate, time.Now()); err != nil {
		log.Error("failed to handle calendar callback", "error", err)
		return
	}
}

func (that *Interaction) handlerAlert2SelectedDate(ctx context.Context, bot *tg.Bot, languageCode string, chatID int64, messageID int, selected time.Time) {
	log := that.logger.With("method", "handlerAlert2SelectedDate", "chat_id", chatID)

	if err := that.chatsRepository.EnableAlert2(ctx, chatID, selected); err != nil {
		log.Error("failed to create alert", "error", err)
		return
	}

	text, err := that.renderLocaledMessage(languageCode, "createAlert2Message", "Date", selected.Format("02.01.2006"))
	if err != nil {
		log.Error("failed to get localized text", "error", err)
		return
	}

	if _, err = bot.EditMessageText(ctx, &tg.EditMessageTextParams{ChatID: chatID, MessageID: messageID, Text: text}); err != nil {
		log.Error("failed to edit selected date message", "error", err)
		return
	}
}

func (that *Interaction) handlerHelp(ctx context.Context, bot *tg.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerHelp", "user_id", update.Message.From.ID)

	_, err := that.sendLocaledMessage(ctx, bot, update, "helpMessage")
	if err != nil {
		log.Error("error sending message", "error", err)
		return
	}
}
