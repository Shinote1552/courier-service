//go:generate mockgen -source=contract.go -destination=./contract_mocks_test.go -package=delivery_test
package delivery

import (
	"context"
	"time"

	"service/internal/entities"
)

type Repository interface {
	Create(ctx context.Context, DeliveryAssignmentEntity entities.DeliveryModify) (*entities.Delivery, error)
	Delete(ctx context.Context, orderID string) error

	GetCourierIDAndDeliveryCountByOrderIDForAssing(ctx context.Context, orderID string) (int64, int64, error)
	GetCourierForAssignment(ctx context.Context) (*entities.Courier, error)
	UpdateCouriersAvailableWhereDeadlineExpired(ctx context.Context) (int64, error)

	GetLastAssignedDeliveryTime(ctx context.Context) (time.Time, error)
	GetCourierIDByOrderID(ctx context.Context, orderID string) (int64, error)
}

type CourierService interface {
	UpdateCourier(ctx context.Context, courierModify entities.CourierModify) (*entities.Courier, error)
}

type DeliveryTimeFactory interface {
	CalculateDeadline(transportType entities.CourierTransportType, baseTime time.Time) time.Time
}

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}
