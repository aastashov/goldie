package telegram

import (
	"bytes"
	"context"
	"image/color"
	"image/png"

	"github.com/fogleman/gg"
	telegramBot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (that *Interaction) handlerStart(ctx context.Context, bot *telegramBot.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerStart", "user_id", update.Message.From.ID, "language", update.Message.From.LanguageCode)

	msg, err := that.renderLocaledMessage(update, "startWelcomeMessage")
	if err != nil {
		log.Error("failed to render message", "error", err)
		return
	}

	_, err = bot.SendMessage(ctx, &telegramBot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   msg,
	})

	if err != nil {
		log.Error("failed to send message", "error", err)
		return
	}
}

func (that *Interaction) handlerPrice(ctx context.Context, bot *telegramBot.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerPrice", "user_id", update.Message.From.ID)

	// TODO: Change the data to the real data
	data := [][]string{
		{"Гр.", "Покупка", "Продажа"},
		{"1.00", "12 526.00", "12 588.50"},
		{"2.00", "23 775.00", "23 870.00"},
		{"5.00", "57 656.50", "57 829.50"},
		{"10.00", "113 495.50", "113 722.50"},
		{"31.1035", "349 573.00", "354 816.50"},
		{"100.00", "1 119 427.50", "1 153 010.50"},
	}

	const (
		rowHeight = 45
		colWidth  = 220
		padding   = 40
		fontSize  = 18
	)

	width := len(data[0])*colWidth + padding*2
	height := len(data)*rowHeight + padding*2

	dc := gg.NewContext(width, height)
	dc.SetColor(color.White)
	dc.Clear()

	if err := dc.LoadFontFace("/System/Library/Fonts/SFNSMono.ttf", fontSize); err != nil {
		log.Error("failed to load font", "error", err)
		return
	}

	dc.SetColor(color.Black)
	y := float64(padding)
	for i, row := range data {
		x := float64(padding)
		for _, col := range row {
			dc.DrawStringAnchored(col, x+colWidth/2, y+rowHeight/2, 0.5, 0.5)
			x += colWidth
		}
		// Horizontal line
		if i == 0 {
			dc.SetLineWidth(2)
		} else {
			dc.SetLineWidth(1)
		}
		dc.DrawLine(float64(padding), y+rowHeight, float64(width-padding), y+rowHeight)
		dc.Stroke()
		y += rowHeight
	}

	buf := new(bytes.Buffer)
	if err := png.Encode(buf, dc.Image()); err != nil {
		log.Error("failed to encode image", "error", err)
		return
	}

	_, err := bot.SendPhoto(ctx, &telegramBot.SendPhotoParams{
		ChatID: update.Message.Chat.ID,
		Photo: &models.InputFileUpload{
			Filename: "gold_table.png",
			Data:     bytes.NewReader(buf.Bytes()),
		},
	})

	if err != nil {
		log.Error("failed to send message", "error", err)
		return
	}
}

func (that *Interaction) handlerHelp(ctx context.Context, bot *telegramBot.Bot, update *models.Update) {
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
