package calendar

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tg "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

const Prefix = "cal:"

var ErrWrongNumberOfLocalizedArguments = fmt.Errorf("wrong number of localized arguments")

type SelectedDateCallback func(ctx context.Context, b *tg.Bot, languageCode string, callBackQueryID string, chatID int64, messageID int, date time.Time)

type Calendar struct {
	disableDays          []time.Weekday
	selectedDateCallback SelectedDateCallback
	bundle               *i18n.Bundle
}

func New(disableDays []time.Weekday, selectedDateHandler SelectedDateCallback, bundle *i18n.Bundle) *Calendar {
	return &Calendar{
		disableDays:          disableDays,
		selectedDateCallback: selectedDateHandler,
		bundle:               bundle,
	}
}

func (c *Calendar) SendCalendar(ctx context.Context, b *tg.Bot, languageCode string, chatID int64, dateStart time.Time, dateEnd time.Time) error {
	return c.sendYearPicker(ctx, b, languageCode, chatID, 0, dateStart, dateEnd, true)
}

func (c *Calendar) HandleCallback(ctx context.Context, b *tg.Bot, languageCode string, update *models.Update, dateStart time.Time, dateEnd time.Time) error {
	var hideAnswerCallback bool

	defer func() {
		if hideAnswerCallback {
			_, _ = b.AnswerCallbackQuery(ctx, &tg.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
			})
		}
	}()

	data := update.CallbackQuery.Data
	if !strings.HasPrefix(data, Prefix) {
		return fmt.Errorf("invalid callback data")
	}

	parts := strings.Split(data, ":")
	if len(parts) < 3 {
		return nil
	}

	chatID := update.CallbackQuery.Message.Message.Chat.ID
	messageID := update.CallbackQuery.Message.Message.ID

	action := parts[1]
	arg := parts[2]

	switch action {
	case "back":
		if arg == "year" {
			return c.sendYearPicker(ctx, b, languageCode, chatID, messageID, dateStart, dateEnd, false)
		}

	case "year":
		selectedYear, _ := strconv.Atoi(arg)
		return c.sendMonthPicker(ctx, b, languageCode, chatID, messageID, dateStart, dateEnd, selectedYear)

	case "month":
		ym := strings.Split(arg, "-")
		if len(ym) != 2 {
			return fmt.Errorf("invalid month")
		}
		selectedYear, _ := strconv.Atoi(ym[0])
		selectedMonthInt, _ := strconv.Atoi(ym[1])
		return c.sendDayPicker(ctx, b, languageCode, chatID, messageID, dateStart, dateEnd, selectedYear, time.Month(selectedMonthInt))

	case "day":
		selected, err := time.Parse("2006-01-02", arg)
		if err != nil {
			return fmt.Errorf("parse selected date: %w", err)
		}

		hideAnswerCallback = true
		c.selectedDateCallback(ctx, b, languageCode, update.CallbackQuery.ID, chatID, messageID, selected)
	}

	return nil
}

func (c *Calendar) sendYearPicker(ctx context.Context, b *tg.Bot, languageCode string, chatID int64, messageID int, dateStart time.Time, dateEnd time.Time, initial bool) error {
	text, err := c.getLocalizedText(languageCode, "chooseYear")
	if err != nil {
		return fmt.Errorf("get localized text: %w", err)
	}

	years := make([]int, 0, dateEnd.Year()-dateStart.Year()+1)
	for i := dateStart.Year(); i <= dateEnd.Year(); i++ {
		years = append(years, i)
	}

	const maxPerRow = 4
	var rows [][]models.InlineKeyboardButton
	var row []models.InlineKeyboardButton

	for _, y := range years {
		row = append(row, models.InlineKeyboardButton{
			Text:         strconv.Itoa(y),
			CallbackData: fmt.Sprintf("cal:year:%d", y),
		})

		if len(row) == maxPerRow {
			rows = append(rows, row)
			row = []models.InlineKeyboardButton{}
		}
	}

	if len(row) > 0 {
		rows = append(rows, row)
	}

	markup := &models.InlineKeyboardMarkup{InlineKeyboard: rows}

	if !initial {
		return c.editMessage(ctx, b, chatID, messageID, text, markup)
	}

	if _, err = b.SendMessage(ctx, &tg.SendMessageParams{ChatID: chatID, Text: text, ReplyMarkup: markup}); err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

func (c *Calendar) sendMonthPicker(ctx context.Context, b *tg.Bot, languageCode string, chatID int64, messageID int, dateStart time.Time, dateEnd time.Time, selectedYear int) error {
	text, err := c.getLocalizedText(languageCode, "chooseMonth", "Year", strconv.Itoa(selectedYear))
	if err != nil {
		return fmt.Errorf("get localized text: %w", err)
	}

	months := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

	// Determine the start and end months for the selected year
	startMonth := 1
	endMonth := 12
	if selectedYear == dateStart.Year() {
		startMonth = int(dateStart.Month())
	}
	if selectedYear == dateEnd.Year() {
		endMonth = int(dateEnd.Month())
	}

	var rows [][]models.InlineKeyboardButton
	var row []models.InlineKeyboardButton
	localizer := i18n.NewLocalizer(c.bundle, languageCode)

	for i, name := range months {
		btn := models.InlineKeyboardButton{Text: "⛔", CallbackData: "cal:noop"}

		if i+1 >= startMonth && i+1 <= endMonth {
			btn.Text, _ = localizer.Localize(&i18n.LocalizeConfig{MessageID: "month." + name})
			btn.CallbackData = fmt.Sprintf("cal:month:%04d-%02d", selectedYear, i+1)
		}

		row = append(row, btn)
		if len(row) == 3 {
			rows = append(rows, row)
			row = []models.InlineKeyboardButton{}
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}

	// add "back to years" button
	backTxt, _ := i18n.NewLocalizer(c.bundle, languageCode).Localize(&i18n.LocalizeConfig{MessageID: "chooseMonth.prev"})
	rows = append(rows, []models.InlineKeyboardButton{
		{Text: backTxt, CallbackData: fmt.Sprintf("cal:back:year")},
	})

	markup := &models.InlineKeyboardMarkup{InlineKeyboard: rows}

	if err = c.editMessage(ctx, b, chatID, messageID, text, markup); err != nil {
		return fmt.Errorf("send month picker: %w", err)
	}

	return nil
}

func (c *Calendar) sendDayPicker(ctx context.Context, b *tg.Bot, languageCode string, chatID int64, messageID int, dateStart time.Time, dateEnd time.Time, selectedYear int, selectedMonth time.Month) error {
	text, err := c.getLocalizedText(languageCode, "chooseDay", "Year", strconv.Itoa(selectedYear), "Month", selectedMonth.String())
	if err != nil {
		return fmt.Errorf("get localized text: %w", err)
	}

	first := time.Date(selectedYear, selectedMonth, 1, 0, 0, 0, 0, time.Local)
	last := first.AddDate(0, 1, -1)

	// header: day of week
	daysOfWeek := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	var rows [][]models.InlineKeyboardButton
	var header []models.InlineKeyboardButton
	for _, d := range daysOfWeek {
		txt, _ := i18n.NewLocalizer(c.bundle, languageCode).Localize(&i18n.LocalizeConfig{MessageID: "day." + d})
		header = append(header, models.InlineKeyboardButton{Text: txt, CallbackData: "cal:noop"})
	}
	rows = append(rows, header)

	// determine the offset for the first day
	// Weekday() в Go: Sunday=0, Monday=1, ..., Saturday=6
	startWeekday := int(first.Weekday())
	if startWeekday == 0 {
		startWeekday = 7 // Sunday -> 7
	}

	// fill empty slots until Monday
	var row []models.InlineKeyboardButton
	for i := 1; i < startWeekday; i++ {
		row = append(row, models.InlineKeyboardButton{Text: " ", CallbackData: "cal:noop"})
	}

	// add days of the month
	for d := 1; d <= last.Day(); d++ {
		date := time.Date(selectedYear, selectedMonth, d, 0, 0, 0, 0, time.Local)
		weekday := date.Weekday()

		btnText := fmt.Sprintf("%2d", d)
		callbackData := fmt.Sprintf("cal:day:%04d-%02d-%02d", selectedYear, selectedMonth, d)

		for _, disabledDay := range c.disableDays {
			if disabledDay == weekday {
				btnText = "🚫"
				callbackData = "cal:noop"
				break
			}
		}

		// disable days outside the date range
		if date.Before(dateStart) || date.After(dateEnd) {
			btnText = "⛔"
			callbackData = "cal:noop"
		}

		row = append(row, models.InlineKeyboardButton{Text: btnText, CallbackData: callbackData})
		if len(row) == 7 {
			rows = append(rows, row)
			row = []models.InlineKeyboardButton{}
		}
	}

	// if there are less than 7 days, fill empty slots
	if len(row) > 0 {
		for len(row) < 7 {
			row = append(row, models.InlineKeyboardButton{Text: " ", CallbackData: "cal:noop"})
		}
		rows = append(rows, row)
	}

	// add "back to months" button
	txt, _ := i18n.NewLocalizer(c.bundle, languageCode).Localize(&i18n.LocalizeConfig{MessageID: "chooseDay.prev"})
	rows = append(rows, []models.InlineKeyboardButton{
		{Text: txt, CallbackData: fmt.Sprintf("cal:year:%d", selectedYear)},
	})

	markup := &models.InlineKeyboardMarkup{InlineKeyboard: rows}
	if err = c.editMessage(ctx, b, chatID, messageID, text, markup); err != nil {
		return fmt.Errorf("send day picker: %w", err)
	}

	return nil
}

func (c *Calendar) editMessage(ctx context.Context, b *tg.Bot, chatID int64, messageID int, text string, markup *models.InlineKeyboardMarkup) error {
	_, err := b.EditMessageText(ctx, &tg.EditMessageTextParams{ChatID: chatID, MessageID: messageID, Text: text, ParseMode: models.ParseModeHTML, ReplyMarkup: markup})
	if err != nil {
		return fmt.Errorf("edit message: %w", err)
	}

	return nil
}

func (c *Calendar) getLocalizedText(languageCode string, messageID string, args ...string) (string, error) {
	if len(args)%2 != 0 {
		return "", ErrWrongNumberOfLocalizedArguments
	}

	templateData := make(map[string]string, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		templateData[args[i]] = args[i+1]
	}

	text, err := i18n.NewLocalizer(c.bundle, languageCode).Localize(&i18n.LocalizeConfig{MessageID: messageID, TemplateData: templateData})
	if err != nil {
		return "", fmt.Errorf("localize message: %w", err)
	}

	return text, nil
}
