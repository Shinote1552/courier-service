package metrics

import "service/pkg/logger"

type handlerLogger interface {
	Info(msg string, fields ...logger.Field)
	Warn(msg string, fields ...logger.Field)
	Error(msg string, fields ...logger.Field)
	With(fields ...logger.Field) logger.Logger
}
