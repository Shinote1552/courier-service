//go:generate mockgen -source=contract.go -destination=./contract_mocks_test.go -package=order_test
package order

import (
	"context"

	"service/internal/entities"
)

type OrderGateway interface {
	GetOrderByID(ctx context.Context, orderID string) (*entities.Order, error)
}

type DeliveryService interface {
	DeliveryAssign(ctx context.Context, orderID string) (*entities.DeliveryAssignment, error)
	DeliveryUnassign(ctx context.Context, orderID string) (*entities.DeliveryUnassignment, error)
	FreeCourierByOrderID(ctx context.Context, orderID string) error
}

type (
	ExecuteFn      func(ctx context.Context, orderID string) error
	HandlerFactory interface {
		GetHandler(status entities.OrderStatusType) (ExecuteFn, error)
	}
)
