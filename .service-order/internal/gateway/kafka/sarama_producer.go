package kafka

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/IBM/sarama"

	"github.com/nikolaev/service-order/internal/domain/entity"
)

// SaramaProducer implements producing order events to Kafka using sarama.
// Env:
//   - KAFKA_BROKERS: comma-separated list of brokers (default: localhost:9092)
//   - KAFKA_ORDER_TOPIC: topic for order status changes (default: order.status.changed)
type SaramaProducer struct {
	p     sarama.SyncProducer
	topic string
}

type createdEvent struct {
	OrderID   string `json:"order_id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func NewSaramaProducer() (*SaramaProducer, error) {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092"
	}

	topic := os.Getenv("KAFKA_ORDER_TOPIC")
	if topic == "" {
		topic = "order.status.changed"
	}

	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Retry.Max = 5

	prod, err := sarama.NewSyncProducer(strings.Split(brokers, ","), cfg)
	if err != nil {
		return nil, err
	}

	return &SaramaProducer{p: prod, topic: topic}, nil
}

func (s *SaramaProducer) Close() error {
	return s.p.Close()
}

func (s *SaramaProducer) OrderCreated(_ context.Context, o *entity.Order) error {
	payload := createdEvent{
		OrderID:   o.ID,
		Status:    string(entity.OrderStatusCreated),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	b, _ := json.Marshal(payload)
	msg := &sarama.ProducerMessage{
		Topic: s.topic,
		Value: sarama.ByteEncoder(b),
	}

	_, _, err := s.p.SendMessage(msg)
	return err
}

func (s *SaramaProducer) OrderUpdated(_ context.Context, o *entity.Order) error {
	payload := createdEvent{
		OrderID:   o.ID,
		Status:    string(o.Status),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	b, _ := json.Marshal(payload)
	msg := &sarama.ProducerMessage{
		Topic: s.topic,
		Value: sarama.ByteEncoder(b),
	}

	_, _, err := s.p.SendMessage(msg)
	return err
}

func (s *SaramaProducer) OrderDeleted(_ context.Context, id string, userID string) error {
	payload := createdEvent{
		OrderID:   id,
		Status:    string(entity.OrderStatusDeleted),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	b, _ := json.Marshal(payload)
	msg := &sarama.ProducerMessage{
		Topic: s.topic,
		Value: sarama.ByteEncoder(b),
	}

	_, _, err := s.p.SendMessage(msg)
	return err
}
