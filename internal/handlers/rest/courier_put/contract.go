//go:generate mockgen -source=contract.go -destination=./contract_mocks_test.go -package=courier_put_test
package courier_put

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
	UpdateCourier(ctx context.Context, CourierModifyEntity entities.CourierModify) (*entities.Courier, error)
}
