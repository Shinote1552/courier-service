package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type (
	Tasks struct {
		CouriersStatusUpdateInterval time.Duration
		OrdersAssingProcessInterval  time.Duration
	}

	HTTPServer struct {
		Port             string
		RequestTimeout   time.Duration // middleware timeout
		RateLimiterQPS   int           // middleware  rate limiter capacity
		RateLimiterBurst int           // middlewarerate limiter burst/refill
		PprofEnabled     bool
		PprofPort        string
	}

	Database struct {
		Host     string
		Port     string
		User     string
		Password string
		DBName   string
		SSLMode  string
	}

	OrderService struct {
		GRPCHost string
	}

	Kafka struct {
		PortHealthcheck string
		Brokers         string
		Topic           string
		ConsumerGroup   string
		Sarama          Sarama
		Handlers        KafkaHandlers
	}

	Sarama struct {
		Version                   string
		ConsumerOffsetsAutocommit bool
	}

	KafkaHandlers struct {
		OrderStatusChanged OrderStatusChanged
	}

	OrderStatusChanged struct {
		ProcessTimeout time.Duration
	}

	Config struct {
		Tasks        Tasks
		Server       HTTPServer
		Database     Database
		OrderService OrderService
		Kafka        Kafka
	}
)

func Load() (*Config, error) {
	cfg, err := loadFromEnv()
	if err != nil {
		return nil, fmt.Errorf("environment loading: %w", err)
	}

	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}
	return cfg, nil
}

func loadFromEnv() (*Config, error) {
	courierInterval, err := osGetEnvDuration("BACKGROUND_COURIERS_STATUS_UPDATE_INTERVAL")
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	orderInterval, err := osGetEnvDuration("BACKGROUND_ORDERS_ASSIGN_PROCESS_INTERVAL")
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	saramaOffsetsAutocommit, err := osGetBool("KAFKA_SARAMA_OFFSETS_AUTOCOMMIT")
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	orderStatusChangedTimeout, err := osGetEnvDuration("KAFKA_HANDLER_ORDER_STATUS_CHANGED_PROCESS_TIMEOUT")
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	requestTimeout, err := osGetEnvDuration("MIDDLEWARE_REQUEST_TIMEOUT")
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	rateLimiterQPS, err := osGetInt("MIDDLEWARE_RATE_LIMIT_QPS")
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	rateLimiterBurst, err := osGetInt("MIDDLEWARE_RATE_LIMIT_BURST")
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	pprofEnabled, err := osGetBool("PPROF_ENABLED")
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	return &Config{
		Tasks: Tasks{
			CouriersStatusUpdateInterval: courierInterval,
			OrdersAssingProcessInterval:  orderInterval,
		},
		Server: HTTPServer{
			Port:             os.Getenv("PORT"),
			RequestTimeout:   requestTimeout,
			RateLimiterQPS:   rateLimiterQPS,
			RateLimiterBurst: rateLimiterBurst,
			PprofEnabled:     pprofEnabled,
			PprofPort:        os.Getenv("PPROF_PORT"),
		},
		Database: Database{
			Host:     os.Getenv("POSTGRES_HOST"),
			Port:     os.Getenv("POSTGRES_PORT"),
			User:     os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			DBName:   os.Getenv("POSTGRES_DB"),
			SSLMode:  os.Getenv("POSTGRES_SSLMODE"),
		},
		OrderService: OrderService{
			GRPCHost: os.Getenv("ORDER_SERVICE_GRPC_HOST"),
		},
		Kafka: Kafka{
			Brokers:         os.Getenv("KAFKA_BROKERS"),
			Topic:           os.Getenv("KAFKA_TOPIC"),
			ConsumerGroup:   os.Getenv("KAFKA_CONSUMER_GROUP"),
			PortHealthcheck: os.Getenv("KAFKA_HTTP_HEALTHCHECK_PORT"),
			Sarama: Sarama{
				Version:                   os.Getenv("KAFKA_SARAMA_VERSION"),
				ConsumerOffsetsAutocommit: saramaOffsetsAutocommit,
			},
			Handlers: KafkaHandlers{
				OrderStatusChanged: OrderStatusChanged{
					ProcessTimeout: orderStatusChangedTimeout,
				},
			},
		},
	}, nil
}

func validateConfig(cfg *Config) error {
	if cfg.Server.Port == "" {
		return errors.New("server port is required (set via PORT env variable)")
	}
	if cfg.Server.RequestTimeout == time.Duration(0) {
		return errors.New("MIDDLEWARE_REQUEST_TIMEOUT is required")
	}
	if cfg.Server.RateLimiterQPS == 0 {
		return errors.New("MIDDLEWARE_RATE_LIMIT_QPS is required")
	}
	if cfg.Server.RateLimiterBurst == 0 {
		return errors.New("MIDDLEWARE_RATE_LIMIT_BURST is required")
	}
	if cfg.Server.PprofPort == "" && cfg.Server.PprofEnabled {
		return errors.New("PprofPort is required (set via PPROF_PORT env variable)")
	}

	if cfg.Database.Host == "" {
		return errors.New("POSTGRES_HOST is required")
	}
	if cfg.Database.Port == "" {
		return errors.New("POSTGRES_PORT is required")
	}
	if cfg.Database.User == "" {
		return errors.New("POSTGRES_USER is required")
	}
	if cfg.Database.Password == "" {
		return errors.New("POSTGRES_PASSWORD is required")
	}
	if cfg.Database.DBName == "" {
		return errors.New("POSTGRES_DB is required")
	}
	if cfg.Database.SSLMode == "" {
		return errors.New("POSTGRES_SSLMODE is required")
	}

	if cfg.Tasks.CouriersStatusUpdateInterval == time.Duration(0) {
		return errors.New("BACKGROUND_COURIERS_STATUS_UPDATE_INTERVAL is required")
	}
	if cfg.Tasks.OrdersAssingProcessInterval == time.Duration(0) {
		return errors.New("BACKGROUND_ORDERS_ASSIGN_PROCESS_INTERVAL is required")
	}

	if cfg.OrderService.GRPCHost == "" {
		return errors.New("ORDER_SERVICE_GRPC_HOST is required")
	}

	if cfg.Kafka.Brokers == "" {
		return errors.New("KAFKA_BROKERS is required")
	}
	if cfg.Kafka.Topic == "" {
		return errors.New("KAFKA_TOPIC is required")
	}
	if cfg.Kafka.ConsumerGroup == "" {
		return errors.New("KAFKA_CONSUMER_GROUP is required")
	}
	if cfg.Kafka.PortHealthcheck == "" {
		return errors.New("KAFKA_HTTP_HEALTHCHECK_PORT is required")
	}

	if cfg.Kafka.Sarama.Version == "" {
		return errors.New("KAFKA_SARAMA_VERSION is required")
	}

	if cfg.Kafka.Handlers.OrderStatusChanged.ProcessTimeout == time.Duration(0) {
		return errors.New("KAFKA_HANDLER_ORDER_STATUS_CHANGED_PROCESS_TIMEOUT is required")
	}

	return nil
}

func osGetInt(s string) (int, error) {
	val := os.Getenv(s)
	if val == "" {
		return 0, nil
	}

	res, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("invalid int format for %s=%q: %w", s, val, err)
	}
	return res, nil
}

func osGetEnvDuration(s string) (time.Duration, error) {
	val := os.Getenv(s)
	if val == "" {
		return time.Duration(0), nil
	}

	res, err := time.ParseDuration(val)
	if err != nil {
		return time.Duration(0), fmt.Errorf("invalid duration format for %s=%q: %w", s, val, err)
	}
	return res, nil
}

func osGetBool(s string) (bool, error) {
	val := os.Getenv(s)
	if val == "" {
		return false, nil
	}

	res, err := strconv.ParseBool(val)
	if err != nil {
		return false, fmt.Errorf("invalid bool format for %s=%q: %w", s, val, err)
	}
	return res, nil
}
