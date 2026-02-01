package main

import (
	"context"
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	_ "net/http/pprof" //nolint:gosec // localhost-only ${PPROF_PORT}
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/avito-tech/go-transaction-manager/pgxv5"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	application "service/internal/app"
	// _ "service/internal/gateway/grpc/order"
	"service/internal/handlers/rest/courier_get"
	"service/internal/handlers/rest/courier_post"
	"service/internal/handlers/rest/courier_put"
	"service/internal/handlers/rest/couriers_get"
	"service/internal/handlers/rest/delivery_assign_post"
	"service/internal/handlers/rest/delivery_unassign_post"
	"service/internal/handlers/rest/healthcheck_head"
	"service/internal/handlers/rest/ping_get"
	"service/internal/pkg/config"
	"service/internal/pkg/dotenv"
	"service/internal/pkg/grpcclient"
	metrics_system "service/internal/pkg/metrics"
	"service/internal/pkg/middlewares/graceful_shutdown"
	"service/internal/pkg/middlewares/metrics"
	"service/internal/pkg/middlewares/rate_limiter"
	"service/internal/pkg/middlewares/timeout"
	"service/internal/pkg/postgres"
	"service/pkg/logger"
	"service/pkg/logger/zap_adapter"
	"service/pkg/token_bucket"
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

	mainLog.Info("starting courier-service application")

	if _, err := os.Stat(".env"); err == nil {
		if err := dotenv.Load(); err != nil {
			mainLog.Error("failed to load .env file", logger.NewField("error", err))
			return
		}
	} else {
		mainLog.Warn("No .env file found, using system environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		mainLog.Error("load config", logger.NewField("error", err))
		return
	}

	err = run(context.Background(), cfg, appLogger)
	if err != nil {
		mainLog.Error("application failed", logger.NewField("error", err))
		return
	}
}

//nolint:contextcheck // Получаю предупреждения от линтера в местах де наследуюсь от context.Background(), хотя это часть gracefull shutdown
func run(ctx context.Context, cfg *config.Config, log logger.Logger) error {
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

	businessApp, err := application.InitializeApplication(ctx, log, pool, pgxv5.DefaultCtxGetter, conn, cfg)
	if err != nil {
		return fmt.Errorf("business logic: %w", err)
	}

	metrics_system.StartSystemMetricsCollector()

	// ongoingCtx используется для BaseContext и не должен отменяться при SIGTERM.
	// Он отменяется только после server.Shutdown() для завершения in-flight запросов.
	// https://victoriametrics.com/blog/go-graceful-shutdown/#b-use-basecontext-to-provide-a-global-context-to-all-connections
	ongoingCtx, stopOngoingGracefully := context.WithCancel(context.Background())
	defer stopOngoingGracefully()

	// основной http сервер
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Server.Port),
		Handler: initRouter(ongoingCtx, log, &isShuttingDown, businessApp, cfg.Server),
		BaseContext: func(_ net.Listener) context.Context {
			return ongoingCtx
		},

		ReadHeaderTimeout: 5 * time.Second, // Slowloris DoS gosec G112
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		defer close(serverErr)
		runLog.Info("server starting",
			logger.NewField("port", cfg.Server.Port),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()
	// основной http сервер

	// pprof http сервер
	var pprofServer *http.Server
	var pprofServerErr chan error
	if cfg.Server.PprofEnabled {
		pprofMux := http.NewServeMux()
		pprofMux.Handle("/debug/pprof/", http.DefaultServeMux)

		pprofServer = &http.Server{
			Addr:    fmt.Sprintf(":%s", cfg.Server.PprofPort),
			Handler: initPprofRouter(&isShuttingDown),
			BaseContext: func(_ net.Listener) context.Context {
				return ongoingCtx
			},

			ReadHeaderTimeout: 5 * time.Second, // Slowloris DoS gosec G112
			ReadTimeout:       60 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       60 * time.Second,
		}

		pprofServerErr = make(chan error, 1)
		go func() {
			defer close(pprofServerErr)
			runLog.Info("pprof server starting",
				logger.NewField("port", cfg.Server.PprofPort),
			)
			if err := pprofServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				pprofServerErr <- err
			}
		}()
	}
	// pprof http сервер

	select {
	case <-ctx.Done():
		runLog.Info("Shutdown signal received")
	case err := <-serverErr:
		return fmt.Errorf("server: %w", err)
	case err := <-pprofServerErr: // if !cfg.Server.PprofEnabled будет nil по умолчанию, и данный кейс будет проигнорирован
		return fmt.Errorf("pprof server: %w", err)
	}

	stop()
	isShuttingDown.Store(true)

	time.Sleep(readinessDrainDelay)
	runLog.Info("draining requests")

	// shutdownCtx должен быть независим от ctx, который уже отменен на этом этапе.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownPeriod)

	defer cancel()

	var shutdownErr error
	err = server.Shutdown(shutdownCtx)
	if pprofServer != nil {
		shutdownErr = pprofServer.Shutdown(shutdownCtx)
		if shutdownErr != nil {
			runLog.Error("pprof server shutdown error", logger.NewField("error", shutdownErr))
		} else {
			runLog.Info("pprof server stopped")
		}
	}

	stopOngoingGracefully()
	if err != nil || shutdownErr != nil {
		runLog.Info("Graceful shutdown timeout, forcing close")
		time.Sleep(shutdownHardPeriod)
	}

	runLog.Info("Server stopped")
	return nil
}

func initRouter(ongoingCtx context.Context, log logger.Logger, isShuttingDown *atomic.Bool, app *application.Application, cfg config.HTTPServer) http.Handler {
	router := mux.NewRouter()

	router.Use(graceful_shutdown.Middleware(isShuttingDown, ongoingCtx))

	router.Use(timeout.Middleware(cfg.RequestTimeout))
	router.Use(metrics.Middleware(log))
	router.Use(rate_limiter.Middleware(log, cfg.RateLimiterQPS, token_bucket.NewTokenBucket(cfg.RateLimiterQPS, float64(cfg.RateLimiterBurst))))
	router.Handle("/metrics", promhttp.Handler())

	router.Handle("/healthcheck", healthcheck_head.New(isShuttingDown)).Methods("HEAD")
	router.Handle("/ping", ping_get.New(log)).Methods("GET")

	router.Handle("/courier/{id}", courier_get.New(log, app.ServiceCourier)).Methods("GET")
	router.Handle("/couriers", couriers_get.New(log, app.ServiceCourier)).Methods("GET")
	router.Handle("/courier", courier_post.New(log, app.ServiceCourier)).Methods("POST")
	router.Handle("/courier", courier_put.New(log, app.ServiceCourier)).Methods("PUT")

	router.Handle("/delivery/assign", delivery_assign_post.New(log, app.ServiceDelivery)).Methods("POST")
	router.Handle("/delivery/unassign", delivery_unassign_post.New(log, app.ServiceDelivery)).Methods("POST")

	return router
}

func initPprofRouter(isShuttingDown *atomic.Bool) http.Handler {
	router := mux.NewRouter()

	router.Handle("/healthcheck", healthcheck_head.New(isShuttingDown)).Methods("HEAD")
	router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)

	return router
}
