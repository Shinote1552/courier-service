package main

import (
	"context"
	"errors"
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/avito-tech/go-transaction-manager/pgxv5"
	"service/internal/app"
	orderstatushandler "service/internal/handlers/kafka-consumer/order_status_changed"
	"service/internal/handlers/rest/healthcheck_head"
	"service/internal/pkg/config"
	"service/internal/pkg/dotenv"
	"service/internal/pkg/grpcclient"
	"service/internal/pkg/kafka"
	"service/internal/pkg/postgres"
	"service/pkg/logger"
	"service/pkg/logger/zap_adapter"
)

func main() {
	zapLogger, err := zap_adapter.NewZapAdapter()
	if err != nil {
		stdlog.Fatalf("failed to initialize logger: %v", err)
	}
	defer func() {
		if err := zapLogger.Sync(); err != nil {
			stdlog.Printf("failed to sync logger: %v", err)
		}
	}()

	var appLogger logger.Logger = zapLogger
	mainLog := appLogger.With()

	mainLog.Info("starting kafka-worker application")

	if _, err := os.Stat(".env"); err == nil {
		if err := dotenv.Load(); err != nil {
			mainLog.Error("failed to load .env file",
				logger.NewField("error", err),
			)
			return
		}
	} else {
		mainLog.Warn("No .env file found, using system environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		mainLog.Error("load config",
			logger.NewField("error", err),
		)
		return
	}

	err = run(context.Background(), appLogger, cfg)
	if err != nil {
		mainLog.Error("application failed",
			logger.NewField("error", err),
		)
		return
	}
}

//nolint:contextcheck // Получаю предупреждения от линтера в местах де наследуюсь от context.Background(), хотя это часть gracefull shutdown
func run(ctx context.Context, log logger.Logger, cfg *config.Config) error {
	const (
		shutdownPeriod      = 15 * time.Second
		shutdownHardPeriod  = 3 * time.Second
		readinessDrainDelay = 5 * time.Second
	)

	// https://victoriametrics.com/blog/go-graceful-shutdown/#b-use-basecontext-to-provide-a-global-context-to-all-connections
	var isShuttingDown atomic.Bool

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	runLog := log.With()

	pool, err := postgres.NewConnPool(ctx, log, &cfg.Database)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()

	conn, err := grpcclient.NewConnClient(ctx, log, &cfg.OrderService)
	if err != nil {
		return fmt.Errorf("gRPC client: %w", err)
	}

	defer func() {
		err := conn.Close()
		if err != nil {
			runLog.Error("failed to close gRPC connection",
				logger.NewField("error", err),
			)
		}
	}()

	businessApp, err := app.InitializeKafkaWorkerApp(ctx, log, pool, pgxv5.DefaultCtxGetter, conn, cfg)
	if err != nil {
		return fmt.Errorf("business logic: %w", err)
	}

	// ongoingCtx используется для BaseContext и не должен отменяться при SIGTERM.
	// Он отменяется только после server.Shutdown() для завершения in-flight запросов.
	// https://victoriametrics.com/blog/go-graceful-shutdown/#b-use-basecontext-to-provide-a-global-context-to-all-connections
	ongoingCtx, stopOngoingGracefully := context.WithCancel(context.Background())
	defer stopOngoingGracefully()

	healthServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Kafka.PortHealthcheck),
		Handler: initHealthcheckRouter(&isShuttingDown),
		BaseContext: func(_ net.Listener) context.Context {
			return ongoingCtx
		},

		ReadHeaderTimeout: 5 * time.Second, // Slowloris DoS gosec G112
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	healthServerErr := make(chan error, 1)
	go func() {
		defer close(healthServerErr)

		runLog.With(
			logger.NewField("port", cfg.Kafka.PortHealthcheck),
		).Info("Server starting")
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			healthServerErr <- err
		}
	}()

	kafkaHandler := orderstatushandler.New(log, businessApp.OrderService, cfg.Kafka.Handlers.OrderStatusChanged.ProcessTimeout)

	brokers := strings.Split(cfg.Kafka.Brokers, ",")
	for i := range brokers {
		brokers[i] = strings.TrimSpace(brokers[i])
	}

	consumer, err := kafka.NewConsumer(
		ctx,
		log,
		&cfg.Kafka,
		brokers,
		cfg.Kafka.ConsumerGroup,
		[]string{cfg.Kafka.Topic},
		kafkaHandler,
	)
	if err != nil {
		return fmt.Errorf("kafka consumer: %w", err)
	}

	consumerErr := make(chan error, 1)
	go func() {
		defer close(consumerErr)

		runLog.With(
			logger.NewField("brokers", brokers),
			logger.NewField("topic", cfg.Kafka.Topic),
			logger.NewField("group", cfg.Kafka.ConsumerGroup),
		).Info("Kafka consumer starting")

		if err := consumer.Start(ongoingCtx); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, sarama.ErrClosedConsumerGroup) {
				runLog.Info("Kafka consumer stopped gracefully")
			} else {
				consumerErr <- err
			}
		}
	}()

	select {
	case <-ctx.Done():
		runLog.Info("Shutdown signal received")
	case err := <-consumerErr:
		return fmt.Errorf("consumer: %w", err)
	case err := <-healthServerErr:
		return fmt.Errorf("healthcheck server: %w", err)
	}

	stop()
	isShuttingDown.Store(true)

	time.Sleep(readinessDrainDelay)
	runLog.Info("Draining Kafka messages")

	// shutdownCtx должен быть независим от ctx, который уже отменен на этом этапе.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownPeriod)
	defer cancel()

	err = healthServer.Shutdown(shutdownCtx)
	if err != nil {
		runLog.Info("Graceful shutdown timeout, forcing close")
		time.Sleep(shutdownHardPeriod)
	}

	stopOngoingGracefully()

	if err := consumer.Close(); err != nil {
		runLog.With(logger.NewField("error", err)).Error("Failed to close Kafka consumer")
	}

	runLog.Info("Worker stopped")
	return nil
}

func initHealthcheckRouter(isShuttingDown *atomic.Bool) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/healthcheck", healthcheck_head.New(isShuttingDown))
	return mux
}
