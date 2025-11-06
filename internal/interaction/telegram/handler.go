package telegram

import (
	"context"
	"fmt"

	telegramBot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (that *Interaction) handlerStart(ctx context.Context, bot *telegramBot.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerStart", "user_id", update.Message.From.ID, "language", update.Message.From.LanguageCode)

	// TODO: Implement start handler
	log.Info("received start handler")
	_, err := bot.SendMessage(ctx, &telegramBot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("Hello, %s! Your language is %s", update.Message.From.FirstName, update.Message.From.LanguageCode),
	})

	if err != nil {
		log.Error("failed to send message", "error", err)
	}
}

func (that *Interaction) handlerAbout(ctx context.Context, bot *telegramBot.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerAbout", "user_id", update.Message.From.ID)

	// TODO: Change the about message
	const aboutMessage = `*MegaLineBalanceBot* \- ваш помощник для удобного отслеживания баланса в личном кабинете MegaLine\.

✨ Я уважаю вашу конфиденциальность и использую данные только для того, чтобы напоминать вам о балансе\.
🛡️ Храню только ту информацию, которая необходима для работы, и ничего лишнего\.
💻 Мой код открыт для всех и доступен на GitHub: [GitHub](https://github\.com/aastashov/megalinekg_bot)\.
🧹 Если захотите удалить свои данные, просто используйте команду \/delete — всё удалится полностью\.

📥 Чтобы сохранить логин и пароль от личного кабинета, используйте команду \/save\. Эти данные будут храниться только для получения актуального баланса и расчетного периода для напоминания\.

Спасибо, что доверяете мне\! 😊`

	disabled := true
	_, err := bot.SendMessage(ctx, &telegramBot.SendMessageParams{
		ChatID:             update.Message.Chat.ID,
		Text:               aboutMessage,
		ParseMode:          models.ParseModeMarkdown,
		LinkPreviewOptions: &models.LinkPreviewOptions{IsDisabled: &disabled},
	})

	if err != nil {
		log.Error("error sending message", "error", err)
		return
	}
}

func (that *Interaction) handlerDelete(ctx context.Context, bot *telegramBot.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerDelete", "user_id", update.Message.From.ID)

	responseText := "Ваши данные удалены. Для начала работы заново, напишите /start."

	// TODO: Implement delete user
	_, err := bot.SendMessage(ctx, &telegramBot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   responseText,
	})

	if err != nil {
		log.Error("error sending message", "error", err, "response_text", responseText)
		return
	}
}
