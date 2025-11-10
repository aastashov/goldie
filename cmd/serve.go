package cmd

import (
	"context"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/atomic"

	"goldie/internal/interaction/nbkr"
	"goldie/internal/interaction/telegram"
	"goldie/internal/repository/prices"
	"goldie/internal/scheduler"
	"goldie/internal/storage"
	"goldie/internal/usecases"
	"goldie/locales"
)

var isReady = atomic.NewBool(false)

var serveCmd = &cobra.Command{
	Use: "serve",
	Run: func(cmd *cobra.Command, _ []string) {
		log := logger.With("package", "cmd")
		ctx := cmd.Context()

		go func() {
			http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
				if !isReady.Load() {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}

				w.WriteHeader(http.StatusOK)
			})
			_ = http.ListenAndServe(":8080", nil)
		}()

		// Initialize database connection
		postgresConnection := storage.MustNewPostgresConnection(logger, cnf.Database.ConnString(), cnf.Logger.ParsedGORMLevel)
		defer postgresConnection.MustClose()

		postgresConnection.MustMigration()

		// Initialize repository
		pricesRepository := prices.NewRepository(postgresConnection.DB)

		bundle, err := locales.GetBundle("")
		cobra.CheckErr(err)

		// Initialize HTTP clients
		telegramClient := &http.Client{Timeout: time.Minute}
		nbkrClient := &http.Client{Timeout: time.Minute}

		// Initialize interactions
		telegramInteractor := telegram.NewInteraction(logger, cnf.Telegram.Token, telegramClient, bundle, pricesRepository)
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

		updatePriceUC.UpdatePrices(ctx) // TODO: Remove this line

		isReady.Store(true)
		log.Info("starting telegram bot")
		telegramInteractor.Start(ctx)
	},
}
