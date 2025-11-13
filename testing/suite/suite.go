package suite

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"gorm.io/gorm"

	"goldie/internal/storage"
)

const maxRetries = 5

type Option func(s *Suite)

type Suite struct {
	T       *testing.T
	Logger  *slog.Logger
	BaseDir string
	Loc     *time.Location

	Conn *storage.PostgresConnection
}

func New(t *testing.T, opts ...Option) (context.Context, *Suite) {
	ctx := context.Background()

	baseDir, err := findProjectRoot()
	if err != nil {
		t.Fatalf("could not get current working directory: %v", err)
	}

	s := &Suite{T: t, Logger: slog.Default(), BaseDir: baseDir, Loc: time.FixedZone("Asia/Bishkek", 6*3600)}
	s.Logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	for _, opt := range opts {
		opt(s)
	}
	return ctx, s
}

func (s *Suite) GetDB() *gorm.DB {
	s.T.Helper()

	if s.Conn == nil {
		s.T.Fatal("Database connection is not initialized! Use suite.New(t, suite.WithPostgres()) option.")
		return nil
	}

	return s.Conn.DB
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if strings.HasSuffix(dir, "goldie") {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (goldie)")
		}
		dir = parent
	}
}

func WithPostgres() Option {
	return func(s *Suite) {
		pool, err := dockertest.NewPool("")
		if err != nil {
			s.T.Fatal(err)
		}

		resource, err := pool.RunWithOptions(&dockertest.RunOptions{
			Repository: "postgres",
			Tag:        "alpine",
			Env: []string{
				"POSTGRES_USER=user",
				"POSTGRES_PASSWORD=pass",
				"POSTGRES_DB=db",
				"listen_addresses = '*'",
			}}, func(config *docker.HostConfig) {
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		})
		if err != nil {
			s.T.Fatal(err)
		}

		_ = resource.Expire(120)

		var conn *storage.PostgresConnection
		var retries int

		err = pool.Retry(func() error {
			retries += 1
			if retries > maxRetries {
				s.T.Fatal("Could not connect to docker container")
			}

			dsn := "postgres://user:pass@localhost:" + resource.GetPort("5432/tcp") + "/db?sslmode=disable"
			c, err := storage.NewPostgresConnection(s.Logger, dsn, 4)
			if err != nil {
				return err
			}
			conn = c
			return nil
		})

		if err != nil {
			s.T.Fatal(err)
		}

		conn.MustMigration()

		s.Conn = conn
		s.T.Log("Postgres ready")

		s.T.Cleanup(func() {
			_ = pool.Purge(resource)
			conn.MustClose()
		})
	}
}
