package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/spf13/cobra"
	"golang.org/x/text/language"

	"goldie/internal/interaction/nbkr"
	"goldie/internal/interaction/telegram"
	"goldie/internal/repository/prices"
	"goldie/internal/scheduler"
	"goldie/internal/storage"
	"goldie/internal/usecases"
)

var serveCmd = &cobra.Command{
	Use: "serve",
	Run: func(cmd *cobra.Command, _ []string) {
		log := logger.With("package", "cmd")
		ctx := cmd.Context()

		// Initialize database connection
		postgresConnection := storage.MustNewPostgresConnection(logger, cnf.Database.ConnString(), cnf.Logger.ParsedGORMLevel)
		defer postgresConnection.MustClose()

		postgresConnection.MustMigration()

		// Initialize repository
		pricesRepository := prices.NewRepository(postgresConnection.DB)

		bundle := i18n.NewBundle(language.English)
		bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

		_, err := bundle.LoadMessageFile("locales/active.en.json")
		cobra.CheckErr(err)
		_, err = bundle.LoadMessageFile("locales/active.ru.json")
		cobra.CheckErr(err)

		// Initialize HTTP clients
		telegramClient := &http.Client{Timeout: time.Minute}
		nbkrClient := &http.Client{Timeout: time.Minute}

		// Initialize interactions
		telegramInteractor := telegram.NewInteraction(logger, cnf.Telegram.Token, telegramClient, bundle)
		nbkrInteractor := nbkr.NewInteraction(logger, nbkrClient)

		// Initialize usecases
		updatePriceUC := usecases.NewUpdatePricesUsecase(logger, pricesRepository, nbkrInteractor)

		// Initialize scheduler
		loc := time.FixedZone("Asia/Bishkek", 6*3600)
		sched := scheduler.New(ctx, loc)

		sched.Add("15 9 * * 1-5", func(ctx context.Context) {
			log.Info("running NBKR update")
			updatePriceUC.UpdatePrices(ctx)
		})

		log.Info("starting telegram bot")
		telegramInteractor.Start(ctx)
	},
}
