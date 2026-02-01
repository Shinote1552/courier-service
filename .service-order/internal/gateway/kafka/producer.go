package kafka

import (
	"context"
	"log"

	"github.com/nikolaev/service-order/internal/domain/entity"
)

type Producer interface {
	OrderCreated(ctx context.Context, o *entity.Order) error
	OrderUpdated(ctx context.Context, o *entity.Order) error
	OrderDeleted(ctx context.Context, id string, userID string) error
}

type NoopProducer struct{}

func (NoopProducer) OrderCreated(_ context.Context, o *entity.Order) error {
	log.Printf("kafka noop: order created %s", o.ID)
	return nil
}

func (NoopProducer) OrderUpdated(_ context.Context, o *entity.Order) error {
	log.Printf("kafka noop: order updated %s", o.ID)
	return nil
}

func (NoopProducer) OrderDeleted(_ context.Context, id string, userID string) error {
	log.Printf("kafka noop: order deleted %s", id)
	return nil
}
