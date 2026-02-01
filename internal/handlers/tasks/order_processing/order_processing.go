package order_processing

import (
	"context"
	"time"
)

type Service interface {
	OrdersAssignProcess(ctx context.Context, cursor time.Time) (time.Time, error)
	GetLastTimeCreateAtCursor(ctx context.Context) (time.Time, error)
}

type OrderProcessing struct {
	service  Service
	interval time.Duration
	cursor   time.Time
}

func NewOrderProcessing(ctx context.Context, service Service, interval time.Duration) (*OrderProcessing, error) {
	cursor, err := service.GetLastTimeCreateAtCursor(ctx)
	if err != nil {
		return nil, err
	}

	return &OrderProcessing{
		service:  service,
		interval: interval,
		cursor:   cursor,
	}, nil
}

// // TTL возвращает интервал между выполнениями задачи.
func (o *OrderProcessing) TTL() time.Duration {
	return o.interval
}

// // Do выполняет логику задачи.
func (o *OrderProcessing) Do(ctx context.Context) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, o.interval)
	defer cancel()

	newCursor, err := o.service.OrdersAssignProcess(ctxWithTimeout, o.cursor)
	if err != nil {
		return err
	}

	if !newCursor.IsZero() && newCursor.After(o.cursor) {
		o.cursor = newCursor
	}

	return nil
}

// // Info возвращает читаемое описание задачи для логгирования и отладки.
func (d *OrderProcessing) Info() string {
	return "order processing"
}
