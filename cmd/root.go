package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"goldie/internal/config"
)

var (
	rootCmd = &cobra.Command{
		Use: "goldie",
	}

	cnf    *config.Config
	logger *slog.Logger
)

func Execute() {
	initConfig()
	initLogger()

	rootCmd.AddCommand(serveCmd)
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initConfig() {
	cnf = config.MustLoad("./config.yml")
}

func initLogger() {
	opts := &slog.HandlerOptions{Level: cnf.Logger.ParsedSlogLevel}
	logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
