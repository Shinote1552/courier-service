package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"service/internal/pkg/config"
	"service/pkg/logger"
	retrierconfig "service/pkg/retrier"
	"service/pkg/retrier/backoff_adapter"
)

const (
	initialInterval = 1 * time.Second
	maxInterval     = 30 * time.Second
	maxElapsedTime  = 2 * time.Minute
	randomization   = 0.5
	multiplier      = 2
)

type Consumer struct {
	log     logger.Logger
	client  sarama.ConsumerGroup
	topics  []string
	handler sarama.ConsumerGroupHandler
}

func NewSaramaConfig(
	versionStr string,
	autoCommit bool,
	initialOffset int64,
	rebalanceStrategy sarama.BalanceStrategy,
) (*sarama.Config, error) {
	cfg := sarama.NewConfig()

	// Version из строки
	version, err := sarama.ParseKafkaVersion(versionStr)
	if err != nil {
		return nil, fmt.Errorf("parse kafka version %q: %w", versionStr, err)
	}
	cfg.Version = version

	// Остальные параметры
	cfg.Consumer.Offsets.Initial = initialOffset
	cfg.Consumer.Offsets.AutoCommit.Enable = autoCommit
	cfg.Consumer.Group.Rebalance.Strategy = rebalanceStrategy

	return cfg, nil
}

func NewConsumer(ctx context.Context, log logger.Logger, cfg *config.Kafka, brokers []string, groupID string, topics []string, handler sarama.ConsumerGroupHandler) (*Consumer, error) {
	saramaConfig, err := NewSaramaConfig(
		cfg.Sarama.Version,
		cfg.Sarama.ConsumerOffsetsAutocommit,
		sarama.OffsetOldest,
		sarama.NewBalanceStrategyRoundRobin(),
	)
	if err != nil {
		return nil, fmt.Errorf("build saramaConfig: %w", err)
	}

	client, err := sarama.NewConsumerGroup(brokers, groupID, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	kafkaLog := log.With(
		logger.NewField("brokers", brokers),
		logger.NewField("group", groupID),
		logger.NewField("topics", topics),
	)

	err = pingKafka(ctx, kafkaLog, brokers, saramaConfig)
	if err != nil {
		clientCloseErr := client.Close()
		if clientCloseErr != nil {
			return nil, fmt.Errorf("kafka client connection: %w (failed to close: %w)", err, clientCloseErr)
		}
		return nil, fmt.Errorf("kafka connection: %w", err)
	}

	return &Consumer{
		log:     kafkaLog,
		client:  client,
		topics:  topics,
		handler: handler,
	}, nil
}

// Start запускает consumer (блокирующий вызов)
func (c *Consumer) Start(ctx context.Context) error {
	c.log.Info("Kafka consumer starting")

	for {
		err := c.client.Consume(ctx, c.topics, c.handler)
		if err != nil {
			c.log.With(
				logger.NewField("error", err),
			).Error("Error from consumer")
			return fmt.Errorf("consumer error: %w", err)
		}

		if ctx.Err() != nil {
			c.log.Warn("Context cancelled, stopping consumer")
			return ctx.Err()
		}
	}
}

func (c *Consumer) Close() error {
	return c.client.Close()
}

func pingKafka(ctx context.Context, log logger.Logger, brokers []string, cfg *sarama.Config) error {
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
		).Info("attempting Kafka connection")

		client, err := sarama.NewClient(brokers, cfg)
		if err != nil {
			return err
		}

		defer func() {
			err := client.Close()
			if err != nil {
				log.Error("failed to close Kafka connection",
					logger.NewField("error", err),
				)
			}
		}()

		_, err = client.Topics()
		return err
	})
	if err != nil {
		log.With(
			logger.NewField("error", err),
			logger.NewField("attempts", attempt),
		).Error("Kafka connection failed after retries")
		return fmt.Errorf("failed to connect to Kafka: %w", err)
	}

	log.With(logger.NewField(
		"attempts", attempt),
	).Info("Kafka connection established")
	return nil
}
