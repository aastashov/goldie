package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tg "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"goldie/internal/model"
)

const settingsAlert2PageSize = 10

func (that *Interaction) handlerStart(ctx context.Context, bot *tg.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerStart", "user_id", update.Message.From.ID, "language", update.Message.From.LanguageCode)

	languageCode := that.getLanguageCode(ctx, update.Message.Chat, update.Message.From)

	startText, err := that.renderLocaledMessage(languageCode, "startWelcomeMessage")
	if err != nil {
		log.Error("failed to render start message", "error", err)
		return
	}

	replyMarkup := &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
		{Text: "🇷🇺 Русский", CallbackData: languageCallbackPrefix + "ru"},
		{Text: "🇬🇧 English", CallbackData: languageCallbackPrefix + "en"},
	}}}

	if _, err = bot.SendMessage(ctx, &tg.SendMessageParams{ChatID: update.Message.Chat.ID, Text: startText, ReplyMarkup: replyMarkup}); err != nil {
		log.Error("failed to send message", "error", err)
		return
	}
}

func (that *Interaction) handlerLanguageSelection(ctx context.Context, bot *tg.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerLanguageSelection")

	if update.CallbackQuery == nil || update.CallbackQuery.Message.Message == nil {
		return
	}

	data := update.CallbackQuery.Data
	if !strings.HasPrefix(data, languageCallbackPrefix) {
		return
	}

	languageCode := strings.TrimPrefix(data, languageCallbackPrefix)
	if languageCode == "" {
		return
	}

	if _, ok := that.supportedLangs[languageCode]; !ok {
		log.Warn("unsupported language selected", "language", languageCode)
		return
	}

	chatID := update.CallbackQuery.Message.Message.Chat.ID
	messageID := update.CallbackQuery.Message.Message.ID

	if err := that.chatsRepository.SetLanguage(ctx, chatID, languageCode); err != nil {
		log.Error("failed to set chat language", "error", err, "chat_id", chatID)
	}

	helpText, err := that.renderLocaledMessage(languageCode, "helpMessage")
	if err != nil {
		log.Error("failed to render start message", "error", err)
	} else if _, err = bot.EditMessageText(ctx, &tg.EditMessageTextParams{ChatID: chatID, MessageID: messageID, Text: helpText}); err != nil {
		log.Error("failed to edit language selection message", "error", err)
	}

	if _, err = bot.AnswerCallbackQuery(ctx, &tg.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID}); err != nil {
		log.Error("failed to answer callback query", "error", err)
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

	languageCode := that.getLanguageCode(ctx, update.Message.Chat, update.Message.From)

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

	languageCode := that.getLanguageCode(ctx, update.Message.Chat, update.Message.From)
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

	languageCode := that.getLanguageCode(ctx, update.CallbackQuery.Message.Message.Chat, update.CallbackQuery.Message.Message.From)
	if err = that.cal.HandleCallback(ctx, bot, languageCode, update, firstCalendarDate, time.Now()); err != nil {
		log.Error("failed to handle calendar callback", "error", err)
		return
	}
}

func (that *Interaction) handlerAlert2SelectedDate(ctx context.Context, bot *tg.Bot, languageCode string, callBackQueryID string, chatID int64, messageID int, selected time.Time) {
	log := that.logger.With("method", "handlerAlert2SelectedDate", "chat_id", chatID)

	var callbackText string
	defer func() {
		_, _ = bot.AnswerCallbackQuery(ctx, &tg.AnswerCallbackQueryParams{
			CallbackQueryID: callBackQueryID,
			Text:            callbackText,
			ShowAlert:       false,
		})
	}()

	if err := that.chatsRepository.CreateAlert2Subscription(ctx, chatID, selected); err != nil {
		log.Error("failed to create alert", "error", err)
		return
	}

	text, err := that.renderLocaledMessage(languageCode, "createAlert2Message")
	if err != nil {
		log.Error("failed to get localized text", "error", err)
		return
	}

	if _, err = bot.EditMessageText(ctx, &tg.EditMessageTextParams{ChatID: chatID, MessageID: messageID, Text: text}); err != nil {
		log.Error("failed to edit selected date message", "error", err)
		return
	}

	callbackText, err = that.renderLocaledMessage(languageCode, "createAlert2CallbackMessage", "Date", selected.Format("02.01.2006"))
	if err != nil {
		log.Error("failed to get localized text", "error", err)
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

func (that *Interaction) handlerStop(ctx context.Context, bot *tg.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerStop", "user_id", update.Message.From.ID)

	if err := that.chatsRepository.DisableAlerts(ctx, update.Message.Chat.ID); err != nil {
		log.Error("failed to disable alerts", "error", err)
		return
	}

	if _, err := that.sendLocaledMessage(ctx, bot, update, "stopMessage"); err != nil {
		log.Error("error sending message", "error", err)
		return
	}
}

func (that *Interaction) handlerInfo(ctx context.Context, bot *tg.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerInfo", "user_id", update.Message.From.ID)

	chat, err := that.chatsRepository.GetChat(ctx, update.Message.Chat.ID)
	if err != nil {
		log.Error("failed to get chat", "error", err)
		return
	}

	alerts, err := that.chatsRepository.ListAlert2Subscriptions(ctx, update.Message.Chat.ID)
	if err != nil {
		log.Error("failed to list alert2 subscriptions", "error", err)
		return
	}

	languageCode := that.getLanguageCode(ctx, update.Message.Chat, update.Message.From)

	userDataLines, err := that.buildUserDataLines(languageCode, update, chat, alerts)
	if err != nil {
		log.Error("failed to build user data lines", "error", err)
		return
	}

	text, err := that.renderLocaledMessage(languageCode, "infoMessage", "UserData", strings.Join(userDataLines, "\n"))
	if err != nil {
		log.Error("failed to render info message", "error", err)
		return
	}

	if _, err = bot.SendMessage(ctx, &tg.SendMessageParams{ChatID: update.Message.Chat.ID, Text: text}); err != nil {
		log.Error("failed to send info message", "error", err)
		return
	}
}

func (that *Interaction) handlerDelete(ctx context.Context, bot *tg.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerDelete", "user_id", update.Message.From.ID)

	languageCode := that.getLanguageCode(ctx, update.Message.Chat, update.Message.From)

	if err := that.chatsRepository.DeleteChat(ctx, update.Message.Chat.ID); err != nil {
		log.Error("failed to delete chat data", "error", err)
		return
	}

	text, err := that.renderLocaledMessage(languageCode, "deleteMessage")
	if err != nil {
		log.Error("failed to render delete message", "error", err)
		return
	}

	if _, err = bot.SendMessage(ctx, &tg.SendMessageParams{ChatID: update.Message.Chat.ID, Text: text}); err != nil {
		log.Error("failed to send delete message", "error", err)
		return
	}
}

func (that *Interaction) buildUserDataLines(languageCode string, update *models.Update, chat *model.TgChat, alerts []*model.TgChatAlert2) ([]string, error) {
	lines := make([]string, 0, 3)

	telegramIDLabel, err := that.renderLocaledMessage(languageCode, "userData.telegramID")
	if err != nil {
		return nil, err
	}
	lines = append(lines, telegramIDLabel+": "+strconv.FormatInt(update.Message.Chat.ID, 10))

	languageLabel, err := that.renderLocaledMessage(languageCode, "userData.language")
	if err != nil {
		return nil, err
	}
	languageValue := languageCode
	if chat != nil && chat.Language != "" {
		languageValue = chat.Language
	} else if update.Message.From.LanguageCode != "" {
		languageValue = update.Message.From.LanguageCode
	}
	lines = append(lines, languageLabel+": "+languageValue)

	if len(alerts) > 0 {
		purchaseDateLabel, err := that.renderLocaledMessage(languageCode, "userData.purchaseDate")
		if err != nil {
			return nil, err
		}

		for i, alert := range alerts {
			label := purchaseDateLabel
			if len(alerts) > 1 {
				label = fmt.Sprintf("%s #%d", purchaseDateLabel, i+1)
			}
			lines = append(lines, label+": "+alert.PurchaseDate.Format("2006-01-02"))
		}
	}

	return lines, nil
}

func (that *Interaction) handlerSettings(ctx context.Context, bot *tg.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerSettings", "user_id", update.Message.From.ID)

	languageCode := that.getLanguageCode(ctx, update.Message.Chat, update.Message.From)

	if err := that.sendSettingsPage(ctx, bot, update.Message.Chat.ID, languageCode, 1, 0); err != nil {
		log.Error("failed to send settings page", "error", err)
	}
}

func (that *Interaction) handlerSettingsCallback(ctx context.Context, bot *tg.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerSettingsCallback")

	if update.CallbackQuery == nil || update.CallbackQuery.Message.Message == nil {
		return
	}

	data := strings.TrimPrefix(update.CallbackQuery.Data, settingsCallbackPrefix)
	if data == "" {
		return
	}

	chatID := update.CallbackQuery.Message.Message.Chat.ID
	messageID := update.CallbackQuery.Message.Message.ID
	languageCode := that.getLanguageCode(ctx, update.CallbackQuery.Message.Message.Chat, update.CallbackQuery.Message.Message.From)

	var callbackText string
	var err error

	switch {
	case strings.HasPrefix(data, "page:"):
		pageStr := strings.TrimPrefix(data, "page:")
		page, parseErr := strconv.Atoi(pageStr)
		if parseErr != nil || page < 1 {
			page = 1
		}
		err = that.sendSettingsPage(ctx, bot, chatID, languageCode, page, messageID)
	case strings.HasPrefix(data, "del:"):
		parts := strings.Split(strings.TrimPrefix(data, "del:"), ":")
		if len(parts) < 2 {
			return
		}
		subscriptionID, parseErr := strconv.ParseInt(parts[0], 10, 64)
		if parseErr != nil {
			return
		}
		page, parseErr := strconv.Atoi(parts[1])
		if parseErr != nil || page < 1 {
			page = 1
		}

		if err = that.chatsRepository.DeleteAlert2Subscription(ctx, chatID, subscriptionID); err == nil {
			if err = that.sendSettingsPage(ctx, bot, chatID, languageCode, page, messageID); err == nil {
				callbackText, _ = that.renderLocaledMessage(languageCode, "settingsAlert2DeleteSuccess")
			}
		}
	default:
		return
	}

	if err != nil {
		log.Error("failed to process settings callback", "error", err)
	}

	if _, answerErr := bot.AnswerCallbackQuery(ctx, &tg.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID, Text: callbackText}); answerErr != nil {
		log.Error("failed to answer settings callback", "error", answerErr)
	}
}

func (that *Interaction) sendSettingsPage(ctx context.Context, bot *tg.Bot, chatID int64, languageCode string, requestedPage int, messageID int) error {
	alerts, currentPage, totalPages, err := that.fetchSettingsPageData(ctx, chatID, requestedPage)
	if err != nil {
		return err
	}

	var text string
	var keyboard *models.InlineKeyboardMarkup

	if len(alerts) == 0 && totalPages == 0 {
		if text, err = that.renderLocaledMessage(languageCode, "settingsAlert2Empty"); err != nil {
			return err
		}
	} else {
		if text, err = that.renderSettingsAlert2Text(languageCode, alerts, currentPage, totalPages); err != nil {
			return err
		}

		if keyboard, err = that.buildSettingsAlert2Keyboard(languageCode, alerts, currentPage, totalPages); err != nil {
			return err
		}
	}

	if messageID == 0 {
		params := &tg.SendMessageParams{ChatID: chatID, Text: text}
		if keyboard != nil {
			params.ReplyMarkup = keyboard
		}

		if _, err = bot.SendMessage(ctx, params); err != nil {
			return fmt.Errorf("send settings message: %w", err)
		}
		return nil
	}

	editParams := &tg.EditMessageTextParams{ChatID: chatID, MessageID: messageID, Text: text}
	if keyboard != nil {
		editParams.ReplyMarkup = keyboard
	}

	if _, err = bot.EditMessageText(ctx, editParams); err != nil {
		return fmt.Errorf("edit settings message: %w", err)
	}

	return nil
}

func (that *Interaction) fetchSettingsPageData(ctx context.Context, chatID int64, requestedPage int) ([]*model.TgChatAlert2, int, int, error) {
	if requestedPage < 1 {
		requestedPage = 1
	}

	offset := (requestedPage - 1) * settingsAlert2PageSize
	alerts, total, err := that.chatsRepository.ListAlert2SubscriptionsPaged(ctx, chatID, settingsAlert2PageSize, offset)
	if err != nil {
		return nil, 1, 0, err
	}

	if total == 0 {
		return nil, 1, 0, nil
	}

	totalPages := int((total + settingsAlert2PageSize - 1) / settingsAlert2PageSize)
	currentPage := requestedPage
	if currentPage > totalPages {
		currentPage = totalPages
		offset = (currentPage - 1) * settingsAlert2PageSize
		alerts, _, err = that.chatsRepository.ListAlert2SubscriptionsPaged(ctx, chatID, settingsAlert2PageSize, offset)
		if err != nil {
			return nil, 1, 0, err
		}
	}

	return alerts, currentPage, totalPages, nil
}

func (that *Interaction) renderSettingsAlert2Text(languageCode string, alerts []*model.TgChatAlert2, currentPage, totalPages int) (string, error) {
	if len(alerts) == 0 {
		return that.renderLocaledMessage(languageCode, "settingsAlert2Empty")
	}

	title, err := that.renderLocaledMessage(languageCode, "settingsAlert2Title",
		"CurrentPage", strconv.Itoa(currentPage),
		"TotalPages", strconv.Itoa(totalPages))
	if err != nil {
		return "", err
	}

	lines := []string{title}
	startIndex := (currentPage-1)*settingsAlert2PageSize + 1
	for i, alert := range alerts {
		line, lineErr := that.renderLocaledMessage(languageCode, "settingsAlert2Item",
			"Index", strconv.Itoa(startIndex+i),
			"Date", alert.PurchaseDate.Format("2006-01-02"))
		if lineErr != nil {
			return "", lineErr
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n"), nil
}

func (that *Interaction) buildSettingsAlert2Keyboard(languageCode string, alerts []*model.TgChatAlert2, currentPage, totalPages int) (*models.InlineKeyboardMarkup, error) {
	if len(alerts) == 0 {
		return nil, nil
	}

	deleteLabel, err := that.renderLocaledMessage(languageCode, "settingsAlert2DeleteButton")
	if err != nil {
		return nil, err
	}

	rows := make([][]models.InlineKeyboardButton, 0, len(alerts)+1)
	for _, alert := range alerts {
		btn := models.InlineKeyboardButton{
			Text:         fmt.Sprintf("%s %s", deleteLabel, alert.PurchaseDate.Format("2006-01-02")),
			CallbackData: fmt.Sprintf("%sdel:%d:%d", settingsCallbackPrefix, alert.ID, currentPage),
		}
		rows = append(rows, []models.InlineKeyboardButton{btn})
	}

	if totalPages > 1 {
		paginationRow := make([]models.InlineKeyboardButton, 0, 2)

		if currentPage > 1 {
			prevLabel, labelErr := that.renderLocaledMessage(languageCode, "settingsAlert2PrevPage")
			if labelErr != nil {
				return nil, labelErr
			}
			paginationRow = append(paginationRow, models.InlineKeyboardButton{
				Text:         prevLabel,
				CallbackData: fmt.Sprintf("%spage:%d", settingsCallbackPrefix, currentPage-1),
			})
		}

		if currentPage < totalPages {
			nextLabel, labelErr := that.renderLocaledMessage(languageCode, "settingsAlert2NextPage")
			if labelErr != nil {
				return nil, labelErr
			}
			paginationRow = append(paginationRow, models.InlineKeyboardButton{
				Text:         nextLabel,
				CallbackData: fmt.Sprintf("%spage:%d", settingsCallbackPrefix, currentPage+1),
			})
		}

		if len(paginationRow) > 0 {
			rows = append(rows, paginationRow)
		}
	}

	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}, nil
}
