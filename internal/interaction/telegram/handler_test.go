package telegram_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"goldie/internal/interaction/telegram"
	"goldie/internal/model"
	"goldie/internal/repository/chats"
	"goldie/internal/repository/prices"
	"goldie/locales"
	botMock "goldie/mocks/bot"
	"goldie/testing/suite"
)

const testSettingsCallbackPrefix = "settings:"

func newUpdate(userID int64, languageCode string, text string) *models.Update {
	return &models.Update{Message: &models.Message{
		From: &models.User{ID: userID, LanguageCode: languageCode},
		Chat: models.Chat{ID: userID},
		Text: text,
	}}
}

func newCallbackQuery(userID int64, languageCode string, data string) *models.Update {
	return &models.Update{CallbackQuery: &models.CallbackQuery{
		ID:   "callback-id",
		Data: data,
		Message: models.MaybeInaccessibleMessage{
			Message: &models.Message{
				ID:   1,
				From: &models.User{ID: userID, LanguageCode: languageCode},
				Chat: models.Chat{ID: userID},
			},
		},
	}}
}

func Test_HandlerPrice(t *testing.T) {
	ctx, st := suite.New(t, suite.WithPostgres())

	pricesRepository := prices.NewRepository(st.GetDB())
	chatRepository := chats.NewRepository(st.GetDB())

	bundle, err := locales.GetBundle(st.BaseDir + "/")
	require.NoError(t, err)

	// Given: Prepared prices for current and previous day
	currentDate := suite.GetDateTime(t, "2024-10-01")
	dbPrices := []*model.GoldPrice{
		{Date: currentDate, Weight: 1, PurchasePrice: 12345, SellPrice: 12588},
		{Date: currentDate.Add(-24 * time.Hour), Weight: 2, PurchasePrice: 12345, SellPrice: 12588},
	}
	require.NoError(t, st.GetDB().WithContext(ctx).Create(&dbPrices).Error)

	newInteractionHandler := func() (*telegram.Interaction, *botMock.MockHttpClient) {
		mockedHTTPClient := botMock.NewMockHttpClient(t)
		return telegram.NewInteraction(st.Logger, "token", mockedHTTPClient, bundle, pricesRepository, chatRepository), mockedHTTPClient
	}

	t.Run("should return prices for the user on the last available date - en", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the prices
			require.Equal(t, "1", formData["chat_id"])
			require.Equal(t, "<b>Gold prices on (2024-10-01)</b>\n<pre>\nGram     Purchase     Sell        \n1        12345.00     12588.00    \n</pre>", formData["text"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		// When: We send the /price command
		interaction.TgBot.ProcessUpdate(ctx, newUpdate(1, "en", "/price"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 100)
	})

	t.Run("should return prices for the user on the last available date - ru", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the prices
			require.Equal(t, "1", formData["chat_id"])
			require.Equal(t, "<b>Цена на золото на (2024-10-01)</b>\n<pre>\nГрамм    Обратный выкуп Продажа     \n1        12345.00     12588.00    \n</pre>", formData["text"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		// When: We send the /price command
		interaction.TgBot.ProcessUpdate(ctx, newUpdate(1, "ru", "/price"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 100)
	})
}

func Test_HandlerAlert1(t *testing.T) {
	ctx, st := suite.New(t, suite.WithPostgres())

	chatsRepository := chats.NewRepository(st.GetDB())
	bundle, err := locales.GetBundle(st.BaseDir + "/")
	require.NoError(t, err)

	newInteractionHandler := func() (*telegram.Interaction, *botMock.MockHttpClient) {
		mockedHTTPClient := botMock.NewMockHttpClient(t)
		return telegram.NewInteraction(st.Logger, "token", mockedHTTPClient, bundle, nil, chatsRepository), mockedHTTPClient
	}

	const (
		existingChatID = 1
		newChatID      = 2
	)

	t.Run("should create alert1 for new chat", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the alert1 message
			require.Equal(t, strconv.FormatInt(newChatID, 10), formData["chat_id"])
			require.Equal(t, "Done. I'll send you an alert about the gold selling price every day at 10:00 AM (UTC +6)", formData["text"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		// When: We send the /alert1 command
		interaction.TgBot.ProcessUpdate(ctx, newUpdate(newChatID, "en", "/alert1"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 100)

		// Then: The chat should be created and alert1 should be enabled
		var createdChat model.TgChat
		require.NoError(t, st.GetDB().WithContext(ctx).Model(&createdChat).Where("source_id = ?", newChatID).First(&createdChat).Error)

		require.True(t, createdChat.Alert1Enabled)
	})

	t.Run("should create alert1 for existing chat", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		dbChat := &model.TgChat{SourceID: existingChatID, Alert1Enabled: false}
		require.NoError(t, st.GetDB().WithContext(ctx).Create(dbChat).Error)

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the alert1 message
			require.Equal(t, strconv.FormatInt(dbChat.SourceID, 10), formData["chat_id"])
			require.Equal(t, "Done. I'll send you an alert about the gold selling price every day at 10:00 AM (UTC +6)", formData["text"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		// When: We send the /alert1 command
		interaction.TgBot.ProcessUpdate(ctx, newUpdate(dbChat.SourceID, "en", "/alert1"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 100)

		// Then: The chat should be updated and alert1 should be enabled
		var updatedChat model.TgChat
		require.NoError(t, st.GetDB().WithContext(ctx).Model(&updatedChat).Where("source_id = ?", dbChat.SourceID).First(&updatedChat).Error)

		require.True(t, updatedChat.Alert1Enabled)
	})
}

func Test_HandlerStop(t *testing.T) {
	ctx, st := suite.New(t, suite.WithPostgres())

	chatsRepository := chats.NewRepository(st.GetDB())
	bundle, err := locales.GetBundle(st.BaseDir + "/")
	require.NoError(t, err)

	newInteractionHandler := func() (*telegram.Interaction, *botMock.MockHttpClient) {
		mockedHTTPClient := botMock.NewMockHttpClient(t)
		return telegram.NewInteraction(st.Logger, "token", mockedHTTPClient, bundle, nil, chatsRepository), mockedHTTPClient
	}

	t.Run("should disable alerts and reply in english", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		dbChat := &model.TgChat{SourceID: 50, Alert1Enabled: true}
		require.NoError(t, st.GetDB().WithContext(ctx).Create(dbChat).Error)
		require.NoError(t, st.GetDB().WithContext(ctx).Create(&model.TgChatAlert2{ChatID: dbChat.ID, PurchaseDate: suite.GetDateTime(t, "2024-09-01")}).Error)

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the stop message
			require.Equal(t, strconv.FormatInt(dbChat.SourceID, 10), formData["chat_id"])
			require.Equal(t, "Bot stopped, including your notifications.\nTo enable them press /alert.", formData["text"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		// When: We send the /stop command
		interaction.TgBot.ProcessUpdate(ctx, newUpdate(dbChat.SourceID, "en", "/stop"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 100)

		// Then: The chat should be updated and alerts should be disabled
		var updatedChat model.TgChat
		require.NoError(t, st.GetDB().WithContext(ctx).Model(&updatedChat).Where("source_id = ?", dbChat.SourceID).First(&updatedChat).Error)
		require.False(t, updatedChat.Alert1Enabled)

		var alert2Count int64
		require.NoError(t, st.GetDB().WithContext(ctx).Model(&model.TgChatAlert2{}).Where("chat_id = ?", updatedChat.ID).Count(&alert2Count).Error)
		require.EqualValues(t, 0, alert2Count)
	})

	t.Run("should create chat if missing and reply in russian", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		const chatID int64 = 77

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the stop message
			require.Equal(t, strconv.FormatInt(chatID, 10), formData["chat_id"])
			require.Equal(t, "Бот остановлен, включая ваши уведомления.\nЧтобы включить их, нажмите /alert.", formData["text"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		// When: We send the /stop command
		interaction.TgBot.ProcessUpdate(ctx, newUpdate(chatID, "ru", "/stop"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 100)

		// Then: The chat should be created and alerts should be disabled
		var createdChat model.TgChat
		require.NoError(t, st.GetDB().WithContext(ctx).Model(&createdChat).Where("source_id = ?", chatID).First(&createdChat).Error)
		require.False(t, createdChat.Alert1Enabled)

		var alert2Count int64
		require.NoError(t, st.GetDB().WithContext(ctx).Model(&model.TgChatAlert2{}).
			Joins("JOIN tg_chats ON tg_chats.id = tg_chat_alert2.chat_id").
			Where("tg_chats.source_id = ?", chatID).
			Count(&alert2Count).Error)
		require.EqualValues(t, 0, alert2Count)
	})
}

func Test_HandlerInfo(t *testing.T) {
	ctx, st := suite.New(t, suite.WithPostgres())

	chatsRepository := chats.NewRepository(st.GetDB())
	bundle, err := locales.GetBundle(st.BaseDir + "/")
	require.NoError(t, err)

	newInteractionHandler := func() (*telegram.Interaction, *botMock.MockHttpClient) {
		mockedHTTPClient := botMock.NewMockHttpClient(t)
		return telegram.NewInteraction(st.Logger, "token", mockedHTTPClient, bundle, nil, chatsRepository), mockedHTTPClient
	}

	t.Run("should show stored data - en", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		// Given: Prepared chat
		dbChat := &model.TgChat{SourceID: 15, Language: "en"}
		require.NoError(t, st.GetDB().WithContext(ctx).Create(dbChat).Error)

		dbAlerts2 := []*model.TgChatAlert2{
			{ChatID: dbChat.ID, PurchaseDate: suite.GetDateTime(t, "2024-10-01")},
			{ChatID: dbChat.ID, PurchaseDate: suite.GetDateTime(t, "2024-09-01")},
		}
		require.NoError(t, st.GetDB().WithContext(ctx).Create(&dbAlerts2).Error)

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the info message
			require.Equal(t, "15", formData["chat_id"])
			require.Equal(t, "What I keep about you:\nTelegramID: 15\nLanguage: en\nPurchase date #1: 2024-09-01\nPurchase date #2: 2024-10-01\n\nTo delete all information about you, enter the /delete command and I will clean up all the data related to you.", formData["text"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{\"ok\":true}`))}, nil
		})

		// When: We send the /info command
		interaction.TgBot.ProcessUpdate(ctx, newUpdate(dbChat.SourceID, "en", "/info"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 100)
	})

	t.Run("should show default data when chat missing - ru", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		// Given: Prepared non-existing chat
		const chatID int64 = 77

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the info message
			require.Equal(t, strconv.FormatInt(chatID, 10), formData["chat_id"])
			require.Equal(t, "Что я храню о тебе:\nTelegramID: 77\nЯзык: ru\n\nЧтобы удалить всю информацию о себе, введи команду /delete, и я удалю все данные, связанные с тобой.", formData["text"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{\"ok\":true}`))}, nil
		})

		// When: We send the /info command
		interaction.TgBot.ProcessUpdate(ctx, newUpdate(chatID, "ru", "/info"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 100)

		// Then: The chat should be created
		var count int64
		require.NoError(t, st.GetDB().WithContext(ctx).Model(&model.TgChat{}).Where("source_id = ?", chatID).Count(&count).Error)
		require.EqualValues(t, 0, count)
	})
}

func Test_HandlerSettings(t *testing.T) {
	ctx, st := suite.New(t, suite.WithPostgres())

	chatsRepository := chats.NewRepository(st.GetDB())
	bundle, err := locales.GetBundle(st.BaseDir + "/")
	require.NoError(t, err)

	newInteractionHandler := func() (*telegram.Interaction, *botMock.MockHttpClient) {
		mockedHTTPClient := botMock.NewMockHttpClient(t)
		return telegram.NewInteraction(st.Logger, "token", mockedHTTPClient, bundle, nil, chatsRepository), mockedHTTPClient
	}

	t.Run("should show empty list when no alert2 subscriptions", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			require.Contains(t, request.URL.Path, "sendMessage")
			require.Equal(t, "70", formData["chat_id"])
			require.Equal(t, "You don't have alert2 subscriptions yet.", formData["text"])
			require.Empty(t, formData["reply_markup"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		interaction.TgBot.ProcessUpdate(ctx, newUpdate(70, "en", "/settings"))
		time.Sleep(time.Millisecond * 100)
	})

	t.Run("should paginate and allow deletion", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		dbChat := &model.TgChat{SourceID: 90, Language: "en"}
		require.NoError(t, st.GetDB().WithContext(ctx).Create(dbChat).Error)

		for i := 0; i < 11; i++ {
			date := suite.GetDateTime(t, fmt.Sprintf("2024-10-%02d", i+1))
			require.NoError(t, st.GetDB().WithContext(ctx).Create(&model.TgChatAlert2{ChatID: dbChat.ID, PurchaseDate: date}).Error)
		}

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			require.Contains(t, request.URL.Path, "sendMessage")
			require.Equal(t, strconv.FormatInt(dbChat.SourceID, 10), formData["chat_id"])
			require.Contains(t, formData["text"], "Your alert2 subscriptions (1/2):")
			require.Contains(t, formData["text"], "Purchase date: 2024-10-11")

			var markup models.InlineKeyboardMarkup
			require.NoError(t, json.Unmarshal([]byte(formData["reply_markup"]), &markup))
			require.Equal(t, 11, len(markup.InlineKeyboard))
			require.Equal(t, "Next »", markup.InlineKeyboard[len(markup.InlineKeyboard)-1][0].Text)
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true,"result":{"message_id":1}}`))}, nil
		})

		interaction.TgBot.ProcessUpdate(ctx, newUpdate(dbChat.SourceID, "en", "/settings"))
		time.Sleep(time.Millisecond * 100)

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			require.Contains(t, request.URL.Path, "editMessageText")
			require.Equal(t, "1", formData["message_id"])
			require.Contains(t, formData["text"], "Your alert2 subscriptions (2/2):")
			require.Contains(t, formData["text"], "Purchase date: 2024-10-01")

			var markup models.InlineKeyboardMarkup
			require.NoError(t, json.Unmarshal([]byte(formData["reply_markup"]), &markup))
			require.Equal(t, 2, len(markup.InlineKeyboard))
			require.Equal(t, "« Prev", markup.InlineKeyboard[len(markup.InlineKeyboard)-1][0].Text)
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			require.Contains(t, request.URL.Path, "answerCallbackQuery")
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		interaction.TgBot.ProcessUpdate(ctx, newCallbackQuery(dbChat.SourceID, "en", testSettingsCallbackPrefix+"page:2"))
		time.Sleep(time.Millisecond * 100)

		var oldestAlert model.TgChatAlert2
		require.NoError(t, st.GetDB().WithContext(ctx).Where("chat_id = ?", dbChat.ID).Order("purchase_date ASC").First(&oldestAlert).Error)

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			require.Contains(t, request.URL.Path, "editMessageText")
			require.Contains(t, formData["text"], "Your alert2 subscriptions (1/1):")

			var markup models.InlineKeyboardMarkup
			require.NoError(t, json.Unmarshal([]byte(formData["reply_markup"]), &markup))
			require.Equal(t, 10, len(markup.InlineKeyboard))
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			require.Contains(t, request.URL.Path, "answerCallbackQuery")
			require.Equal(t, "Subscription deleted.", formData["text"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		interaction.TgBot.ProcessUpdate(ctx, newCallbackQuery(dbChat.SourceID, "en", fmt.Sprintf("%sdel:%d:2", testSettingsCallbackPrefix, oldestAlert.ID)))
		time.Sleep(time.Millisecond * 100)

		var count int64
		require.NoError(t, st.GetDB().WithContext(ctx).Model(&model.TgChatAlert2{}).Where("chat_id = ?", dbChat.ID).Count(&count).Error)
		require.EqualValues(t, 10, count)
	})
}

func Test_HandlerDelete(t *testing.T) {
	ctx, st := suite.New(t, suite.WithPostgres())

	chatsRepository := chats.NewRepository(st.GetDB())
	bundle, err := locales.GetBundle(st.BaseDir + "/")
	require.NoError(t, err)

	newInteractionHandler := func() (*telegram.Interaction, *botMock.MockHttpClient) {
		mockedHTTPClient := botMock.NewMockHttpClient(t)
		return telegram.NewInteraction(st.Logger, "token", mockedHTTPClient, bundle, nil, chatsRepository), mockedHTTPClient
	}

	t.Run("should delete chat and respond - en", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		// Given: Prepared chat
		dbChat := &model.TgChat{SourceID: 22, Language: "en"}
		require.NoError(t, st.GetDB().WithContext(ctx).Create(dbChat).Error)
		require.NoError(t, st.GetDB().WithContext(ctx).Create(&model.TgChatAlert2{ChatID: dbChat.ID, PurchaseDate: suite.GetDateTime(t, "2023-09-01")}).Error)

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the delete message
			require.Equal(t, "22", formData["chat_id"])
			require.Equal(t, "We forgot about you.", formData["text"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{\"ok\":true}`))}, nil
		})

		// When: We send the /delete command
		interaction.TgBot.ProcessUpdate(ctx, newUpdate(dbChat.SourceID, "en", "/delete"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 100)

		// Then: The chat should be deleted
		var count int64
		require.NoError(t, st.GetDB().WithContext(ctx).Model(&model.TgChat{}).Where("source_id = ?", dbChat.SourceID).Count(&count).Error)
		require.EqualValues(t, 0, count)

		require.NoError(t, st.GetDB().WithContext(ctx).Model(&model.TgChatAlert2{}).Where("chat_id = ?", dbChat.ID).Count(&count).Error)
		require.EqualValues(t, 0, count)
	})

	t.Run("should respond even if chat absent - ru", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		// Given: Prepared non-existing chat
		const chatID int64 = 88

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the delete message
			require.Equal(t, strconv.FormatInt(chatID, 10), formData["chat_id"])
			require.Equal(t, "Я забыл все о тебе.", formData["text"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{\"ok\":true}`))}, nil
		})

		// When: We send the /delete command
		interaction.TgBot.ProcessUpdate(ctx, newUpdate(chatID, "ru", "/delete"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 100)

		// Then: The chat should be deleted
		var count int64
		require.NoError(t, st.GetDB().WithContext(ctx).Model(&model.TgChat{}).Where("source_id = ?", chatID).Count(&count).Error)
		require.EqualValues(t, 0, count)
		require.NoError(t, st.GetDB().WithContext(ctx).Model(&model.TgChatAlert2{}).
			Joins("JOIN tg_chats ON tg_chats.id = tg_chat_alert2.chat_id").
			Where("tg_chats.source_id = ?", chatID).
			Count(&count).Error)
		require.EqualValues(t, 0, count)
	})
}

func Test_HandlerAlert2(t *testing.T) {
	ctx, st := suite.New(t, suite.WithPostgres())

	pricesRepository := prices.NewRepository(st.GetDB())
	chatRepository := chats.NewRepository(st.GetDB())

	// Given: Prepared prices for the first year
	dbPrices := []*model.GoldPrice{
		{Date: suite.GetDateTime(t, "2024-10-01"), Weight: 1, PurchasePrice: 12345, SellPrice: 12588},
	}
	require.NoError(t, st.GetDB().WithContext(ctx).Create(&dbPrices).Error)

	bundle, err := locales.GetBundle(st.BaseDir + "/")
	require.NoError(t, err)

	newInteractionHandler := func() (*telegram.Interaction, *botMock.MockHttpClient) {
		mockedHTTPClient := botMock.NewMockHttpClient(t)
		return telegram.NewInteraction(st.Logger, "token", mockedHTTPClient, bundle, pricesRepository, chatRepository), mockedHTTPClient
	}

	const chatID = 2

	t.Run("should send calendar for any chat - en", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the alert2 message
			require.Equal(t, strconv.FormatInt(chatID, 10), formData["chat_id"])
			require.Equal(t, "📅 Choose year:", formData["text"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true, "result":{"message_id":12345}}`))}, nil
		})

		// When: We send the /alert2 command
		interaction.TgBot.ProcessUpdate(ctx, newUpdate(chatID, "en", "/alert2"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 200)
	})

	t.Run("should send calendar for any chat - ru", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the alert2 message
			require.Equal(t, strconv.FormatInt(chatID, 10), formData["chat_id"])
			require.Equal(t, "📅 Выберите год:", formData["text"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true, "result":{"message_id":12345}}`))}, nil
		})

		// When: We send the /alert2 command
		interaction.TgBot.ProcessUpdate(ctx, newUpdate(chatID, "ru", "/alert2"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 200)
	})
}

func Test_HandlerStart(t *testing.T) {
	ctx, st := suite.New(t, suite.WithPostgres())

	chatsRepository := chats.NewRepository(st.GetDB())
	bundle, err := locales.GetBundle(st.BaseDir + "/")
	require.NoError(t, err)

	newInteractionHandler := func() (*telegram.Interaction, *botMock.MockHttpClient) {
		mockedHTTPClient := botMock.NewMockHttpClient(t)
		return telegram.NewInteraction(st.Logger, "token", mockedHTTPClient, bundle, nil, chatsRepository), mockedHTTPClient
	}

	t.Run("should send language prompt with inline keyboard", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the language prompt with inline keyboard
			require.Equal(t, "1", formData["chat_id"])
			require.Equal(t, "Hello. This is Goldie, your assistant for buying and selling gold bars. Choose your language", formData["text"])

			var markup models.InlineKeyboardMarkup
			require.NoError(t, json.Unmarshal([]byte(formData["reply_markup"]), &markup))
			require.Len(t, markup.InlineKeyboard, 1)
			require.Len(t, markup.InlineKeyboard[0], 2)

			require.Equal(t, "🇷🇺 Русский", markup.InlineKeyboard[0][0].Text)
			require.Equal(t, "lang:ru", markup.InlineKeyboard[0][0].CallbackData)

			require.Equal(t, "🇬🇧 English", markup.InlineKeyboard[0][1].Text)
			require.Equal(t, "lang:en", markup.InlineKeyboard[0][1].CallbackData)

			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		// When: We send the /start command
		interaction.TgBot.ProcessUpdate(ctx, newUpdate(1, "en", "/start"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 100)
	})

	t.Run("should save language and send help after selection", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			switch {
			case strings.Contains(request.URL.Path, "editMessageText"):
				// Then: The message should be updated with the help text
				require.Equal(t, "1", formData["message_id"])
				require.Equal(t, "Привет! Это Goldie, ваш помощник по покупке и продаже золотых слитков. Основная функция бота - следить за ценами на золото и рассчитывать, сколько вы заработаете, если продадите его сегодня.\n\nДоступные команды:\n/help - Информация о командах\n/start — Выберите язык\n/price — Показать текущую цену на золото\n/alert — Настроить новое оповещение\n/info - Информация о хранении данных пользователя\n/settings - Настройка оповещений\n/stop - Остановить бота", formData["text"])
			case strings.Contains(request.URL.Path, "answerCallbackQuery"):
				// Then: The callback query should be answered
				require.Equal(t, "callback-id", formData["callback_query_id"])
			default:
				t.Fatalf("unexpected telegram method: %s", request.URL.Path)
			}

			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		// When: the user selects Russian language
		interaction.TgBot.ProcessUpdate(ctx, newCallbackQuery(1, "en", "lang:ru"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 200)

		// Then: the chat should be created and the language should be set
		var chat model.TgChat
		require.NoError(t, st.GetDB().WithContext(ctx).Model(&chat).Where("source_id = ?", 1).First(&chat).Error)
		require.Equal(t, "ru", chat.Language)
	})

	t.Run("should send localized start if chat language exists", func(t *testing.T) {
		const chatID int64 = 3
		require.NoError(t, st.GetDB().WithContext(ctx).Create(&model.TgChat{SourceID: chatID, Language: "ru"}).Error)

		interaction, mockedHTTPClient := newInteractionHandler()

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the language prompt with inline keyboard
			require.Equal(t, strconv.FormatInt(chatID, 10), formData["chat_id"])
			require.Equal(t, "Привет. Это Goldie, твой помощник по покупке и продаже золотых слитков. Выбери язык", formData["text"])

			var markup models.InlineKeyboardMarkup
			require.NoError(t, json.Unmarshal([]byte(formData["reply_markup"]), &markup))
			require.Len(t, markup.InlineKeyboard, 1)
			require.Len(t, markup.InlineKeyboard[0], 2)

			require.Equal(t, "🇷🇺 Русский", markup.InlineKeyboard[0][0].Text)
			require.Equal(t, "lang:ru", markup.InlineKeyboard[0][0].CallbackData)

			require.Equal(t, "🇬🇧 English", markup.InlineKeyboard[0][1].Text)
			require.Equal(t, "lang:en", markup.InlineKeyboard[0][1].CallbackData)

			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		// When: We send the /start command from the English chat but with the Russian language in database
		interaction.TgBot.ProcessUpdate(ctx, newUpdate(chatID, "en", "/start"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 200)
	})
}

func Test_handlerAlert2SelectedDate(t *testing.T) {
	ctx, st := suite.New(t, suite.WithPostgres())

	pricesRepository := prices.NewRepository(st.GetDB())
	chatsRepository := chats.NewRepository(st.GetDB())

	// Given: Prepared prices for the first year
	dbPrices := []*model.GoldPrice{
		{Date: suite.GetDateTime(t, "2024-10-01"), Weight: 1, PurchasePrice: 12345, SellPrice: 12588},
	}
	require.NoError(t, st.GetDB().WithContext(ctx).Create(&dbPrices).Error)

	bundle, err := locales.GetBundle(st.BaseDir + "/")
	require.NoError(t, err)

	newInteractionHandler := func() (*telegram.Interaction, *botMock.MockHttpClient) {
		mockedHTTPClient := botMock.NewMockHttpClient(t)
		return telegram.NewInteraction(st.Logger, "token", mockedHTTPClient, bundle, pricesRepository, chatsRepository), mockedHTTPClient
	}

	t.Run("should create alert2 for new chat - en", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			switch {
			case strings.Contains(request.URL.Path, "editMessageText"):
				// Then: The message should be updated with the alert2 text
				require.Equal(t, "1", formData["message_id"])
				require.Equal(t, "Done. I'll send you an alert about how much I'll earn if you sell today at 10:00 AM (UTC +6)", formData["text"])
			case strings.Contains(request.URL.Path, "answerCallbackQuery"):
				// Then: The callback query should be answered
				require.Equal(t, "callback-id", formData["callback_query_id"])
			default:
				t.Fatalf("unexpected telegram method: %s", request.URL.Path)
			}

			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		// When: We send the /alert2 command
		interaction.TgBot.ProcessUpdate(ctx, newCallbackQuery(1, "en", "cal:day:2024-10-01"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 200)

		// Then: The alert subscription should be created
		var chat model.TgChat
		require.NoError(t, st.GetDB().WithContext(ctx).Where("source_id = ?", 1).First(&chat).Error)
		var alerts []model.TgChatAlert2
		require.NoError(t, st.GetDB().WithContext(ctx).Where("chat_id = ?", chat.ID).Find(&alerts).Error)
		require.Len(t, alerts, 1)
		require.Equal(t, suite.GetDateTime(t, "2024-10-01").In(time.Local), alerts[0].PurchaseDate.In(time.Local))
	})

	t.Run("should create alert2 for new chat - ru", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			switch {
			case strings.Contains(request.URL.Path, "editMessageText"):
				// Then: The message should be updated with the alert2 text
				require.Equal(t, "1", formData["message_id"])
				require.Equal(t, "Готово. Буду слать уведомления о том, сколько ты заработаешь, если сегодня продашь в 10:00 AM (UTC +6)", formData["text"])
			case strings.Contains(request.URL.Path, "answerCallbackQuery"):
				// Then: The callback query should be answered
				require.Equal(t, "callback-id", formData["callback_query_id"])
			default:
				t.Fatalf("unexpected telegram method: %s", request.URL.Path)
			}

			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		// When: We send the /alert2 command
		interaction.TgBot.ProcessUpdate(ctx, newCallbackQuery(1, "ru", "cal:day:2024-10-01"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 200)

		// Then: The alert subscription should be created
		var chat model.TgChat
		require.NoError(t, st.GetDB().WithContext(ctx).Where("source_id = ?", 1).First(&chat).Error)
		var alerts []model.TgChatAlert2
		require.NoError(t, st.GetDB().WithContext(ctx).Where("chat_id = ?", chat.ID).Find(&alerts).Error)
		require.Len(t, alerts, 1)
		require.Equal(t, suite.GetDateTime(t, "2024-10-01").In(time.Local), alerts[0].PurchaseDate.In(time.Local))
	})

	t.Run("should create alert2 for existing chat - en", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		dbChat := &model.TgChat{SourceID: 100}
		require.NoError(t, st.GetDB().WithContext(ctx).Create(dbChat).Error)
		require.NoError(t, st.GetDB().WithContext(ctx).Create(&model.TgChatAlert2{ChatID: dbChat.ID, PurchaseDate: suite.GetDateTime(t, "2024-09-01")}).Error)

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			switch {
			case strings.Contains(request.URL.Path, "editMessageText"):
				// Then: The message should be updated with the alert2 text
				require.Equal(t, "1", formData["message_id"])
				require.Equal(t, "Done. I'll send you an alert about how much I'll earn if you sell today at 10:00 AM (UTC +6)", formData["text"])
			case strings.Contains(request.URL.Path, "answerCallbackQuery"):
				// Then: The callback query should be answered
				require.Equal(t, "callback-id", formData["callback_query_id"])
			default:
				t.Fatalf("unexpected telegram method: %s", request.URL.Path)
			}

			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		// When: We send the /alert2 command
		interaction.TgBot.ProcessUpdate(ctx, newCallbackQuery(dbChat.SourceID, "en", "cal:day:2024-10-01"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 200)

		// Then: A new alert subscription should be added without removing previous ones
		var alerts []model.TgChatAlert2
		require.NoError(t, st.GetDB().WithContext(ctx).Where("chat_id = ?", dbChat.ID).Order("purchase_date").Find(&alerts).Error)
		require.Len(t, alerts, 2)
		require.Equal(t, suite.GetDateTime(t, "2024-09-01").In(time.Local), alerts[0].PurchaseDate.In(time.Local))
		require.Equal(t, suite.GetDateTime(t, "2024-10-01").In(time.Local), alerts[1].PurchaseDate.In(time.Local))
	})

	t.Run("should create alert2 for existing chat - ru", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		dbChat := &model.TgChat{SourceID: 200}
		require.NoError(t, st.GetDB().WithContext(ctx).Create(dbChat).Error)
		require.NoError(t, st.GetDB().WithContext(ctx).Create(&model.TgChatAlert2{ChatID: dbChat.ID, PurchaseDate: suite.GetDateTime(t, "2024-08-01")}).Error)

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			switch {
			case strings.Contains(request.URL.Path, "editMessageText"):
				// Then: The message should be updated with the alert2 text
				require.Equal(t, "1", formData["message_id"])
				require.Equal(t, "Готово. Буду слать уведомления о том, сколько ты заработаешь, если сегодня продашь в 10:00 AM (UTC +6)", formData["text"])
			case strings.Contains(request.URL.Path, "answerCallbackQuery"):
				// Then: The callback query should be answered
				require.Equal(t, "callback-id", formData["callback_query_id"])
			default:
				t.Fatalf("unexpected telegram method: %s", request.URL.Path)
			}

			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		// When: We send the /alert2 command
		interaction.TgBot.ProcessUpdate(ctx, newCallbackQuery(dbChat.SourceID, "ru", "cal:day:2024-10-01"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 200)

		// Then: A new alert subscription should be added without removing previous ones
		var alerts []model.TgChatAlert2
		require.NoError(t, st.GetDB().WithContext(ctx).Where("chat_id = ?", dbChat.ID).Order("purchase_date").Find(&alerts).Error)
		require.Len(t, alerts, 2)
		require.Equal(t, suite.GetDateTime(t, "2024-08-01").In(time.Local), alerts[0].PurchaseDate.In(time.Local))
		require.Equal(t, suite.GetDateTime(t, "2024-10-01").In(time.Local), alerts[1].PurchaseDate.In(time.Local))
	})
}

func Test_HandlerHelp(t *testing.T) {
	ctx, st := suite.New(t, suite.WithPostgres())

	chatRepository := chats.NewRepository(st.GetDB())

	bundle, err := locales.GetBundle(st.BaseDir + "/")
	require.NoError(t, err)

	newInteractionHandler := func() (*telegram.Interaction, *botMock.MockHttpClient) {
		mockedHTTPClient := botMock.NewMockHttpClient(t)
		return telegram.NewInteraction(st.Logger, "token", mockedHTTPClient, bundle, nil, chatRepository), mockedHTTPClient
	}

	t.Run("should return help message - en", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the help message
			require.Equal(t, "1", formData["chat_id"])
			require.Equal(t, "Hello. This is Goldie, your assistant for buying and selling gold bars. The bot's main function is to monitor gold prices and calculate how much you'll earn if you sell it today.\n\nAvailable commands:\n/help - Commands information\n/start - Choose your language\n/price - Show current gold price\n/alert - Configure your alerts\n/info - Information on storing user data\n/settings - Alerts setting\n/stop - Stop bot", formData["text"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		// When: We send the /help command
		interaction.TgBot.ProcessUpdate(ctx, newUpdate(1, "en", "/help"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 100)
	})

	t.Run("should return help message - ru", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the help message
			require.Equal(t, "1", formData["chat_id"])
			require.Equal(t, "Привет! Это Goldie, ваш помощник по покупке и продаже золотых слитков. Основная функция бота - следить за ценами на золото и рассчитывать, сколько вы заработаете, если продадите его сегодня.\n\nДоступные команды:\n/help - Информация о командах\n/start — Выберите язык\n/price — Показать текущую цену на золото\n/alert — Настроить новое оповещение\n/info - Информация о хранении данных пользователя\n/settings - Настройка оповещений\n/stop - Остановить бота", formData["text"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		// When: We send the /help command
		interaction.TgBot.ProcessUpdate(ctx, newUpdate(1, "ru", "/help"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 100)
	})
}
