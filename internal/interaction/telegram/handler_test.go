package telegram_test

import (
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

func newUpdate(userID int64, languageCode string, text string) *models.Update {
	return &models.Update{Message: &models.Message{
		From: &models.User{ID: userID, LanguageCode: languageCode},
		Chat: models.Chat{ID: userID},
		Text: text,
	}}
}

func newCallbackQuery(userID int64, languageCode string, data string) *models.Update {
	return &models.Update{CallbackQuery: &models.CallbackQuery{
		Data: data,
		Message: models.MaybeInaccessibleMessage{
			Message: &models.Message{
				From: &models.User{ID: userID, LanguageCode: languageCode},
				Chat: models.Chat{ID: userID},
			},
		},
	}}
}

func Test_HandlerPrice(t *testing.T) {
	ctx, st := suite.New(t, suite.WithPostgres())

	pricesRepository := prices.NewRepository(st.GetDB())
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
		return telegram.NewInteraction(st.Logger, "token", mockedHTTPClient, bundle, pricesRepository, nil), mockedHTTPClient
	}

	t.Run("should return prices for the user on the last available date - en", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the prices
			require.Equal(t, "1", formData["chat_id"])
			require.Equal(t, "<b>Gold prices on (2024-10-01)</b>\n<pre>\nGram     Buy          Sell        \n1        12345.00     12588.00    \n</pre>", formData["text"])
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
			require.Equal(t, "<b>Цены мерных слитков на (2024-10-01)</b>\n<pre>\nГрамм    Покупка      Продажа     \n1        12345.00     12588.00    \n</pre>", formData["text"])
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

func Test_HandlerAlert2(t *testing.T) {
	ctx, st := suite.New(t, suite.WithPostgres())

	pricesRepository := prices.NewRepository(st.GetDB())

	// Given: Prepared prices for the first year
	dbPrices := []*model.GoldPrice{
		{Date: suite.GetDateTime(t, "2024-10-01"), Weight: 1, PurchasePrice: 12345, SellPrice: 12588},
	}
	require.NoError(t, st.GetDB().WithContext(ctx).Create(&dbPrices).Error)

	bundle, err := locales.GetBundle(st.BaseDir + "/")
	require.NoError(t, err)

	newInteractionHandler := func() (*telegram.Interaction, *botMock.MockHttpClient) {
		mockedHTTPClient := botMock.NewMockHttpClient(t)
		return telegram.NewInteraction(st.Logger, "token", mockedHTTPClient, bundle, pricesRepository, nil), mockedHTTPClient
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

			// Then: The user should receive the alert2 message
			require.Equal(t, strconv.FormatInt(1, 10), formData["chat_id"])
			require.Equal(t, "Done. I'll send you an alert about how much I'll earn if you sell today at 10:00 AM (UTC +6)", formData["text"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		// When: We send the /alert2 command
		interaction.TgBot.ProcessUpdate(ctx, newCallbackQuery(1, "en", "cal:day:2024-10-01"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 200)

		// Then: The chat should be created and alert2 should be enabled
		var createdChat model.TgChat
		require.NoError(t, st.GetDB().WithContext(ctx).Model(&createdChat).Where("source_id = ?", 1).First(&createdChat).Error)

		require.True(t, createdChat.Alert2Enabled)
		require.Equal(t, suite.GetDateTime(t, "2024-10-01").In(time.Local), createdChat.Alert2Date)
	})

	t.Run("should create alert2 for new chat - ru", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the alert2 message
			require.Equal(t, strconv.FormatInt(1, 10), formData["chat_id"])
			require.Equal(t, "Готово. Буду слать уведомления о том, сколько ты заработаешь, если сегодня продашь в 10:00 AM (UTC +6)", formData["text"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		// When: We send the /alert2 command
		interaction.TgBot.ProcessUpdate(ctx, newCallbackQuery(1, "ru", "cal:day:2024-10-01"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 200)

		// Then: The chat should be created and alert2 should be enabled
		var createdChat model.TgChat
		require.NoError(t, st.GetDB().WithContext(ctx).Model(&createdChat).Where("source_id = ?", 1).First(&createdChat).Error)

		require.True(t, createdChat.Alert2Enabled)
		require.Equal(t, suite.GetDateTime(t, "2024-10-01").In(time.Local), createdChat.Alert2Date)
	})

	t.Run("should create alert2 for existing chat - en", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		dbChat := &model.TgChat{SourceID: 100, Alert2Enabled: false}
		require.NoError(t, st.GetDB().WithContext(ctx).Create(dbChat).Error)

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the alert2 message
			require.Equal(t, strconv.FormatInt(dbChat.SourceID, 10), formData["chat_id"])
			require.Equal(t, "Done. I'll send you an alert about how much I'll earn if you sell today at 10:00 AM (UTC +6)", formData["text"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		// When: We send the /alert2 command
		interaction.TgBot.ProcessUpdate(ctx, newCallbackQuery(dbChat.SourceID, "en", "cal:day:2024-10-01"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 200)

		// Then: The chat should be updated and alert2 should be enabled
		var updatedChat model.TgChat
		require.NoError(t, st.GetDB().WithContext(ctx).Model(&updatedChat).Where("source_id = ?", dbChat.SourceID).First(&updatedChat).Error)

		require.True(t, updatedChat.Alert2Enabled)
		require.Equal(t, suite.GetDateTime(t, "2024-10-01").In(time.Local), updatedChat.Alert2Date)
	})

	t.Run("should create alert2 for existing chat - ru", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		dbChat := &model.TgChat{SourceID: 200, Alert2Enabled: false}
		require.NoError(t, st.GetDB().WithContext(ctx).Create(dbChat).Error)

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the alert2 message
			require.Equal(t, strconv.FormatInt(dbChat.SourceID, 10), formData["chat_id"])
			require.Equal(t, "Готово. Буду слать уведомления о том, сколько ты заработаешь, если сегодня продашь в 10:00 AM (UTC +6)", formData["text"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		// When: We send the /alert2 command
		interaction.TgBot.ProcessUpdate(ctx, newCallbackQuery(dbChat.SourceID, "ru", "cal:day:2024-10-01"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 200)

		// Then: The chat should be updated and alert2 should be enabled
		var updatedChat model.TgChat
		require.NoError(t, st.GetDB().WithContext(ctx).Model(&updatedChat).Where("source_id = ?", dbChat.SourceID).First(&updatedChat).Error)

		require.True(t, updatedChat.Alert2Enabled)
		require.Equal(t, suite.GetDateTime(t, "2024-10-01").In(time.Local), updatedChat.Alert2Date)
	})
}

func Test_HandlerHelp(t *testing.T) {
	ctx, st := suite.New(t)

	bundle, err := locales.GetBundle(st.BaseDir + "/")
	require.NoError(t, err)

	newInteractionHandler := func() (*telegram.Interaction, *botMock.MockHttpClient) {
		mockedHTTPClient := botMock.NewMockHttpClient(t)
		return telegram.NewInteraction(st.Logger, "token", mockedHTTPClient, bundle, nil, nil), mockedHTTPClient
	}

	t.Run("should return help message - en", func(t *testing.T) {
		interaction, mockedHTTPClient := newInteractionHandler()

		mockedHTTPClient.EXPECT().Do(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			formData := suite.ParseRequestBody(t, request)

			// Then: The user should receive the help message
			require.Equal(t, "1", formData["chat_id"])
			require.Equal(t, "/start — Choose your language\n/price — Show current gold price\n/alert — Configure your alerts", formData["text"])
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
			require.Equal(t, "/start — Выберите язык\n/price — Показать текущую цену мерных слитков\n/alert — Настроить свои предупреждения", formData["text"])
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
		})

		// When: We send the /help command
		interaction.TgBot.ProcessUpdate(ctx, newUpdate(1, "ru", "/help"))

		// Wait for the handler to be executed
		time.Sleep(time.Millisecond * 100)
	})
}
