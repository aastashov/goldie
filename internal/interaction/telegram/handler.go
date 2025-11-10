package telegram

import (
	"context"
	"fmt"
	"sort"
	"strings"

	telegramBot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (that *Interaction) handlerStart(ctx context.Context, bot *telegramBot.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerStart", "user_id", update.Message.From.ID, "language", update.Message.From.LanguageCode)

	if _, err := that.sendLocaledMessage(ctx, bot, update, "startWelcomeMessage"); err != nil {
		log.Error("failed to send message", "error", err)
		return
	}
}

func (that *Interaction) handlerPrice(ctx context.Context, bot *telegramBot.Bot, update *models.Update) {
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

	sort.SliceStable(prices, func(i, j int) bool {
		if prices[i].Date.Equal(prices[j].Date) {
			return prices[i].Weight < prices[j].Weight
		}
		return prices[i].Date.After(prices[j].Date)
	})

	currentDate := prices[0].Date
	title, _ := that.renderLocaledMessage(update, "goldPricesTitle", "Date", currentDate.Format("2006-01-02"))
	headerWeight, _ := that.renderLocaledMessage(update, "columnWeight")
	headerBuy, _ := that.renderLocaledMessage(update, "columnBuy")
	headerSell, _ := that.renderLocaledMessage(update, "columnSell")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>%s</b>\n<pre>\n", title))
	sb.WriteString(fmt.Sprintf("%-8s %-12s %-12s\n", headerWeight, headerBuy, headerSell))

	for _, p := range prices {
		if !p.Date.Equal(currentDate) {
			break
		}
		sb.WriteString(fmt.Sprintf("%-8.4g %-12.2f %-12.2f\n", p.Weight, p.PurchasePrice, p.SellPrice))
	}

	sb.WriteString("</pre>")
	text := sb.String()

	if _, err = bot.SendMessage(ctx, &telegramBot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: text, ParseMode: models.ParseModeHTML}); err != nil {
		log.Error("error sending message", "error", err)
		return
	}
}

func (that *Interaction) handlerHelp(ctx context.Context, bot *telegramBot.Bot, update *models.Update) {
	log := that.logger.With("method", "handlerHelp", "user_id", update.Message.From.ID)

	_, err := that.sendLocaledMessage(ctx, bot, update, "helpMessage")
	if err != nil {
		log.Error("error sending message", "error", err)
		return
	}
}
