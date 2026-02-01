package order

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"service/internal/entities"
	proto "service/internal/generated/proto/clients"
	retrierconfig "service/pkg/retrier"
	"service/pkg/retrier/backoff_adapter"
)

const (
	serviceName = "order-service"
)

const (
	initialInterval = 100 * time.Millisecond
	maxInterval     = 2 * time.Second
	maxElapsedTime  = 1 * time.Second
	randomization   = 0.5
	multiplier      = 2.0
)

type OrderGateway struct {
	client  client
	retrier retrier
}

func New(client client) *OrderGateway {
	retryConfig := retrierconfig.Config{
		InitialInterval: initialInterval,
		MaxInterval:     maxInterval,
		MaxElapsedTime:  maxElapsedTime,
		Randomization:   randomization,
		Multiplier:      multiplier,
		ShouldRetry:     isRetryableCode,
	}

	return &OrderGateway{
		client:  client,
		retrier: backoff_adapter.New(retryConfig),
	}
}

func (o *OrderGateway) GetOrdersAfter(ctx context.Context, after time.Time) ([]entities.Order, error) {
	req := proto.GetOrdersRequest{
		From: timestamppb.New(after),
	}

	var resp *proto.GetOrdersResponse

	err := o.executeWithMetrics(ctx, "GetOrders", func(ctx context.Context) error {
		var err error
		resp, err = o.client.GetOrders(ctx, &req)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("gateway order, get orders: %w", err)
	}

	return toDomainList(resp), nil
}

func (o *OrderGateway) GetOrderByID(ctx context.Context, orderID string) (*entities.Order, error) {
	req := &proto.GetOrderByIdRequest{
		Id: orderID,
	}

	var resp *proto.GetOrderByIdResponse

	err := o.executeWithMetrics(ctx, "GetOrderById", func(ctx context.Context) error {
		var err error
		resp, err = o.client.GetOrderById(ctx, req)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("gateway order, get order: %s: %w", orderID, err)
	}

	if resp.Order == nil {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	return toDomain(resp.Order), nil
}

func isRetryableCode(err error) bool {
	if err == nil {
		return false
	}
	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	switch st.Code() {
	case codes.ResourceExhausted, // По ДЗ 429 из HTTP, для grpc это '8'
		// Добавил еще некоторые коды при которых будут происходить ретраи
		codes.Unavailable,
		codes.DeadlineExceeded:
		return true
	default:
		return false
	}
}

// Тут наверное декаратор как будто бы лучше подошел, особенно когда тут начанет все больше наслаиватся concerns, как думаете?
// Примерно~ latency metric -> attempts metric -> retrier -> gateway
func (o *OrderGateway) executeWithMetrics(ctx context.Context, method string, fn func(context.Context) error) error {
	var attempt uint64
	start := time.Now()

	err := o.retrier.ExecuteWithContext(ctx, func(ctx context.Context) error {
		attempt++
		return fn(ctx)
	})

	grpcCode := getGRPCCode(err)
	// Метрики Prometheus
	GatewayRequestDuration.WithLabelValues(serviceName, method, grpcCode).Observe(time.Since(start).Seconds())

	if attempt > 1 {
		// Метрики Prometheus
		GatewayRetriesTotal.WithLabelValues(serviceName, method, grpcCode).Inc()
	}

	return err
}

func getGRPCCode(err error) string {
	if err == nil {
		return "OK"
	}
	if st, ok := status.FromError(err); ok {
		return st.Code().String()
	}
	return "UNKNOWN"
}
