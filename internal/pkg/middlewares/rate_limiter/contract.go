package rate_limiter

import "service/pkg/logger"

type Limiter interface {
	Allow() bool
}

type handlerLogger interface {
	Info(msg string, fields ...logger.Field)
	Warn(msg string, fields ...logger.Field)
	Error(msg string, fields ...logger.Field)
	With(fields ...logger.Field) logger.Logger
}
