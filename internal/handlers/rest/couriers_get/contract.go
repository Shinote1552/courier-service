//go:generate mockgen -source=contract.go -destination=./contract_mocks_test.go -package=couriers_get_test
package couriers_get

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
	GetCouriers(ctx context.Context) ([]entities.Courier, error)
}
