package cmd

import (
	"github.com/spf13/cobra"

	"goldie/internal/interaction/telegram"
	"goldie/internal/storage"
)

var serveCmd = &cobra.Command{
	Use: "serve",
	Run: func(cmd *cobra.Command, _ []string) {
		log := logger.With("package", "cmd")
		ctx := cmd.Context()

		// Initialize database connection
		postgresConnection := storage.MustNewPostgresConnection(logger, cnf.Database.ConnString(), cnf.Logger.ParsedGORMLevel)
		_ = postgresConnection // TODO: need to use this connection

		// Initialize interactions
		telegramInteractor := telegram.NewInteraction(logger, cnf.Telegram.Token)

		log.Info("starting telegram bot")
		telegramInteractor.Start(ctx)
	},
}
