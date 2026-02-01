package integration_test

import (
	"context"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/avito-tech/go-transaction-manager/pgxv5"
	"github.com/stretchr/testify/require"
	"service/internal/pkg/config"
	"service/internal/pkg/postgres"
	"service/pkg/logger/zap_adapter"
	"service/pkg/querier"
)

var (
	querierInstance *querier.Querier
	querierOnce     sync.Once
)

func GetQuerier() *querier.Querier {
	querierOnce.Do(func() {
		// godotenv.Load(.env.test) не вызываем так как Makefile подгружает их
		cfg := &config.Database{
			Host:     os.Getenv("POSTGRES_HOST"),
			Port:     os.Getenv("POSTGRES_PORT"),
			User:     os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			DBName:   os.Getenv("POSTGRES_DB"),
			SSLMode:  os.Getenv("POSTGRES_SSLMODE"),
		}

		ctx := context.Background()

		zapLogger, err := zap_adapter.NewZapAdapter()
		if err != nil {
			log.Fatalf("failed to initialize logger: %v", err)
		}
		defer func() {
			if err := zapLogger.Sync(); err != nil {
				log.Printf("failed to sync logger: %v", err)
			}
		}()

		connPool, err := postgres.NewConnPool(ctx, zapLogger, cfg)
		if err != nil {
			panic(err)
		}

		querierInstance = querier.New(connPool, pgxv5.DefaultCtxGetter)
	})

	return querierInstance
}

func SetupDB(t *testing.T, setupSql string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := GetQuerier().Exec(ctx, setupSql)

	require.NoError(t, err)
}

func TeardownDB(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := GetQuerier().Exec(ctx, `
		TRUNCATE TABLE delivery, couriers RESTART IDENTITY CASCADE;
	`)
	require.NoError(t, err)
}
