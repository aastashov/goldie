package telegram_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"goldie/internal/interaction/telegram"
	"goldie/internal/model"
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

func Test_HandlerPrice(t *testing.T) {
	st, ctx := suite.New(t, suite.WithPostgres())

	pricesRepository := prices.NewRepository(st.Conn.DB)
	bundle, err := locales.GetBundle(st.BaseDir + "/")
	require.NoError(t, err)

	// Given: Prepared prices for current and previous day
	currentDate := suite.GetDateTime(t, "2024-10-01")
	dbPrices := []*model.GoldPrice{
		{Date: currentDate, Weight: 1, PurchasePrice: 12345, SellPrice: 12588},
		{Date: currentDate.Add(-24 * time.Hour), Weight: 2, PurchasePrice: 12345, SellPrice: 12588},
	}
	require.NoError(t, st.Conn.DB.WithContext(ctx).Create(&dbPrices).Error)

	newInteractionHandler := func() (*telegram.Interaction, *botMock.MockHttpClient) {
		mockedHTTPClient := botMock.NewMockHttpClient(t)
		return telegram.NewInteraction(st.Logger, "token", mockedHTTPClient, bundle, pricesRepository), mockedHTTPClient
	}

	t.Run("should return prices for the user on the last available date", func(t *testing.T) {
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
}
