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

	expectedPrices := []nbkr.GoldPrice{
		{Date: "06.11.2025", Weight: 1, PurchasePrice: 12526, SellPrice: 12588.5},
		{Date: "06.11.2025", Weight: 2, PurchasePrice: 23775, SellPrice: 23870},
		{Date: "06.11.2025", Weight: 5, PurchasePrice: 57656.5, SellPrice: 57829.5},
		{Date: "06.11.2025", Weight: 10, PurchasePrice: 113495.5, SellPrice: 113722.5},
		{Date: "06.11.2025", Weight: 31.1035, PurchasePrice: 349573, SellPrice: 354816.5},
		{Date: "06.11.2025", Weight: 100, PurchasePrice: 1.1194275e+06, SellPrice: 1.1530105e+06},
		{Date: "07.11.2025", Weight: 1, PurchasePrice: 12577, SellPrice: 12640},
		{Date: "07.11.2025", Weight: 2, PurchasePrice: 23877.5, SellPrice: 23973},
		{Date: "07.11.2025", Weight: 5, PurchasePrice: 57914.5, SellPrice: 58088},
		{Date: "07.11.2025", Weight: 10, PurchasePrice: 114010.5, SellPrice: 114238.5},
		{Date: "07.11.2025", Weight: 31.1035, PurchasePrice: 351173.5, SellPrice: 356441},
		{Date: "07.11.2025", Weight: 100, PurchasePrice: 1.124573e+06, SellPrice: 1.15831e+06},
	}

	require.Equal(t, expectedPrices, prices)
}
