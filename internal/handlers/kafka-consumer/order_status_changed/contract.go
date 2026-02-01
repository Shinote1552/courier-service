package order_status_changed

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
	ProcessOrderStatusChange(ctx context.Context, orderModify entities.OrderModify) (*entities.Order, error)
}
