package nbkr_test

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"

	"goldie/internal/interaction/nbkr"
	"goldie/testing/suite"
)

func Test_GetGoldPrices(t *testing.T) {
	r, err := recorder.New(filepath.Join("testdata", strings.ReplaceAll(t.Name(), "/", "_")))
	require.NoError(t, err)

	t.Cleanup(func() {
		// Make sure recorder is stopped once done with it.
		require.NoError(t, r.Stop())
	})

	client := r.GetDefaultClient()

	interaction := nbkr.NewInteraction(slog.Default(), client)

	startDate := suite.GetDateTime(t, "2025-11-06")
	endDate := suite.GetDateTime(t, "2025-11-09")

	prices, err := interaction.GetGoldPrices(context.Background(), startDate, endDate)
	require.NoError(t, err)

	dateOne := suite.GetDateTime(t, "2025-11-06")
	dateTwo := suite.GetDateTime(t, "2025-11-07")
	expectedPrices := []nbkr.GoldPrice{
		{Date: dateOne, Weight: 1, PurchasePrice: 12526, SellPrice: 12588.5},
		{Date: dateOne, Weight: 2, PurchasePrice: 23775, SellPrice: 23870},
		{Date: dateOne, Weight: 5, PurchasePrice: 57656.5, SellPrice: 57829.5},
		{Date: dateOne, Weight: 10, PurchasePrice: 113495.5, SellPrice: 113722.5},
		{Date: dateOne, Weight: 31.1035, PurchasePrice: 349573, SellPrice: 354816.5},
		{Date: dateOne, Weight: 100, PurchasePrice: 1.1194275e+06, SellPrice: 1.1530105e+06},
		{Date: dateTwo, Weight: 1, PurchasePrice: 12577, SellPrice: 12640},
		{Date: dateTwo, Weight: 2, PurchasePrice: 23877.5, SellPrice: 23973},
		{Date: dateTwo, Weight: 5, PurchasePrice: 57914.5, SellPrice: 58088},
		{Date: dateTwo, Weight: 10, PurchasePrice: 114010.5, SellPrice: 114238.5},
		{Date: dateTwo, Weight: 31.1035, PurchasePrice: 351173.5, SellPrice: 356441},
		{Date: dateTwo, Weight: 100, PurchasePrice: 1.124573e+06, SellPrice: 1.15831e+06},
	}

	require.Equal(t, expectedPrices, prices)
}
