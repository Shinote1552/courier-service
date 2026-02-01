package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"service/internal/pkg/config"
	"service/pkg/logger"
	retrierconfig "service/pkg/retrier"
	"service/pkg/retrier/backoff_adapter"
)

const (
	maxConns        = 10
	minConns        = 5
	maxConnLifetime = time.Hour

	initialInterval = 5 * time.Second
	maxInterval     = 30 * time.Second
	maxElapsedTime  = 2 * time.Minute
	randomization   = 0.5
	multiplier      = 2
)

func NewConnPool(ctx context.Context, log logger.Logger, cfg *config.Database) (*pgxpool.Pool, error) {
	connString := newDsn(cfg)

	poolCfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = maxConns
	poolCfg.MaxConnLifetime = maxConnLifetime
	poolCfg.MinConns = minConns

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("connection pool: %w", err)
	}

	dbLog := log.With(
		logger.NewField("host", cfg.Host),
		logger.NewField("port", cfg.Port),
		logger.NewField("db", cfg.DBName),
	)

	err = pingDatabase(ctx, dbLog, pool)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("database connection: %w", err)
	}

	return pool, nil
}

func newDsn(cfg *config.Database) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DBName,
		cfg.SSLMode,
	)
}

func pingDatabase(ctx context.Context, log logger.Logger, pool *pgxpool.Pool) error {
	retryConfig := retrierconfig.Config{
		InitialInterval: initialInterval,
		MaxInterval:     maxInterval,
		MaxElapsedTime:  maxElapsedTime,
		Randomization:   randomization,
		Multiplier:      multiplier,
		ShouldRetry:     nil, // все ошибки ретраим
	}

	retrier := backoff_adapter.New(retryConfig)

	var attempt uint64
	err := retrier.ExecuteWithContext(ctx, func(ctx context.Context) error {
		attempt++
		log.With(
			logger.NewField("attempt", attempt),
		).Info("attempting Database connection")

		return pool.Ping(ctx)
	})
	if err != nil {
		log.With(
			logger.NewField("error", err),
			logger.NewField("attempts", attempt),
		).Error("Database connection failed after retries")
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.With(
		logger.NewField("attempts", attempt),
	).Info("Database connection established")
	return nil
}
