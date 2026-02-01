//go:generate mockgen -source=contract.go -destination=./contract_mocks_test.go -package=delivery_unassign_post_test
package delivery_unassign_post

import (
	"context"

	"service/internal/entities"
	"service/pkg/logger"
)

type handlerLogger interface {
	Info(msg string, fields ...logger.Field)
	Warn(msg string, fields ...logger.Field)
	Error(msg string, fields ...logger.Field)
	With(fields ...logger.Field) logger.Logger
}

type Service interface {
	DeliveryUnassign(ctx context.Context, orderId string) (*entities.DeliveryUnassignment, error)
}
