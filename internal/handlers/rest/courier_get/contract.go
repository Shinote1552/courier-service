//go:generate mockgen -source=contract.go -destination=./contract_mocks_test.go -package=courier_get_test
package courier_get

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
	GetCourier(ctx context.Context, id int64) (*entities.Courier, error)
}
