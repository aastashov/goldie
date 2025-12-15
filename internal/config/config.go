package config

import (
	"fmt"
	"log/slog"

	"github.com/ilyakaznacheev/cleanenv"
)

const DefaultLanguageCode = "en"

type Config struct {
	Database Database `yaml:"database"`
	Telegram Telegram `yaml:"telegram"`
	Logger   Logger   `yaml:"logger"`
}

type Database struct {
	Host     string `env-default:"localhost" yaml:"host"`
	Port     int    `env-default:"5432" yaml:"port"`
	User     string `env-default:"postgres" yaml:"user"`
	Password string `env-default:"postgres" yaml:"password"`
	Name     string `env-default:"postgres" yaml:"name"`
	SSLMode  string `env-default:"disable" yaml:"ssl-mode"`
}

func (d *Database) ConnString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode)
}

func (d *Database) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode) //nolint:nosprintfhostport // it's ok
}

type Telegram struct {
	Token string `env-default:"" yaml:"token"`
}

type Logger struct {
	Level           string     `env-default:"info" yaml:"level"`
	ParsedSlogLevel slog.Level `yaml:"-"`
	GORMLevel       string     `env-default:"info" yaml:"gorm_level"`
	ParsedGORMLevel slog.Level `yaml:"-"`
}

// MustLoad loads config from a file.
func MustLoad(configPath string) *Config {
	cnf := &Config{}

	if err := cleanenv.ReadConfig(configPath, cnf); err != nil {
		panic(fmt.Errorf("cannot read config: %w", err))
	}

	switch cnf.Logger.GORMLevel {
	case "silent":
		cnf.Logger.ParsedGORMLevel = slog.LevelDebug
	case "info":
		cnf.Logger.ParsedGORMLevel = slog.LevelInfo
	case "warn":
		cnf.Logger.ParsedGORMLevel = slog.LevelWarn
	case "error":
		cnf.Logger.ParsedGORMLevel = slog.LevelError
	default:
		cnf.Logger.ParsedGORMLevel = slog.LevelInfo
	}

	switch cnf.Logger.Level {
	case "debug":
		cnf.Logger.ParsedSlogLevel = slog.LevelDebug
	case "info":
		cnf.Logger.ParsedSlogLevel = slog.LevelInfo
	case "warn":
		cnf.Logger.ParsedSlogLevel = slog.LevelWarn
	case "error":
		cnf.Logger.ParsedSlogLevel = slog.LevelError
	default:
		cnf.Logger.ParsedSlogLevel = slog.LevelInfo
	}

	return cnf
}
