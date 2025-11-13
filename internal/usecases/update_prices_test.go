package usecases_test

import (
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"

	"goldie/internal/interaction/nbkr"
	"goldie/internal/model"
	"goldie/internal/repository/prices"
	"goldie/internal/usecases"
	"goldie/testing/suite"
)

func Test_UpdatePricesUseCase_FirstImport(t *testing.T) {
	t.Run("should import prices for the first year", func(t *testing.T) {
		ctx, st := suite.New(t, suite.WithPostgres())
		pricesRepository := prices.NewRepository(st.GetDB())

		r, err := recorder.New(filepath.Join("testdata", strings.ReplaceAll(t.Name(), "/", "_")))
		require.NoError(t, err)

		t.Cleanup(func() {
			// Make sure recorder is stopped once done with it.
			require.NoError(t, r.Stop())
		})

		interaction := nbkr.NewInteraction(slog.Default(), r.GetDefaultClient())
		updatePriceUC := usecases.NewUpdatePricesUseCase(st.Logger, pricesRepository, interaction, st.Loc)

		// When: We import the prices for the first year
		updatePriceUC.FirstImport(ctx)

		// Then: The prices should be created in the database
		var createdPrices []*model.GoldPrice
		require.NoError(t, st.GetDB().WithContext(ctx).Model(&createdPrices).Omit("ID", "CreatedAt", "UpdatedAt").Where("date = ?", suite.GetDateTime(t, usecases.FirstPriceDate)).Find(&createdPrices).Error)
		require.Len(t, createdPrices, 6)

		sort.SliceStable(createdPrices, func(i, j int) bool {
			if createdPrices[i].Date.Equal(createdPrices[j].Date) {
				return createdPrices[i].Weight < createdPrices[j].Weight
			}
			return createdPrices[i].Date.After(createdPrices[j].Date)
		})

		expectedPrices := []*model.GoldPrice{
			{Date: suite.GetDateTime(t, usecases.FirstPriceDate).In(time.Local), Weight: 1, PurchasePrice: 3822, SellPrice: 3841},
			{Date: suite.GetDateTime(t, usecases.FirstPriceDate).In(time.Local), Weight: 2, PurchasePrice: 6542.5, SellPrice: 6568.5},
			{Date: suite.GetDateTime(t, usecases.FirstPriceDate).In(time.Local), Weight: 5, PurchasePrice: 14969.5, SellPrice: 15014.5},
			{Date: suite.GetDateTime(t, usecases.FirstPriceDate).In(time.Local), Weight: 10, PurchasePrice: 29094, SellPrice: 29152},
			{Date: suite.GetDateTime(t, usecases.FirstPriceDate).In(time.Local), Weight: 31.1035, PurchasePrice: 87745, SellPrice: 87833},
			{Date: suite.GetDateTime(t, usecases.FirstPriceDate).In(time.Local), Weight: 100, PurchasePrice: 239802, SellPrice: 240041.5},
		}
		require.Equal(t, expectedPrices, createdPrices)
	})

	t.Run("shouldn't import prices for the first year if they already exist", func(t *testing.T) {
		ctx, st := suite.New(t, suite.WithPostgres())
		pricesRepository := prices.NewRepository(st.GetDB())

		r, err := recorder.New(filepath.Join("testdata", strings.ReplaceAll(t.Name(), "/", "_")))
		require.NoError(t, err)

		t.Cleanup(func() {
			// Make sure recorder is stopped once done with it.
			require.NoError(t, r.Stop())
		})

		// Given: Prepared prices for the first year and for some other years
		dbPrices := []*model.GoldPrice{
			{Date: suite.GetDateTime(t, usecases.FirstPriceDate), Weight: 3, PurchasePrice: 34567, SellPrice: 34901},
		}
		require.NoError(t, st.GetDB().WithContext(ctx).Create(&dbPrices).Error)

		interaction := nbkr.NewInteraction(slog.Default(), r.GetDefaultClient())
		updatePriceUC := usecases.NewUpdatePricesUseCase(st.Logger, pricesRepository, interaction, st.Loc)

		// When: We import the prices for the first year
		updatePriceUC.FirstImport(ctx)

		// Then: The prices shouldn't be created in the database
		var createdPrices []*model.GoldPrice
		require.NoError(t, st.GetDB().WithContext(ctx).Model(&createdPrices).Omit("ID", "CreatedAt", "UpdatedAt").Where("date != ?", suite.GetDateTime(t, usecases.FirstPriceDate)).Find(&createdPrices).Error)
		require.Len(t, createdPrices, 0)
	})
}
