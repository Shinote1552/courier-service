package zap_adapter

import (
	"go.uber.org/zap"
	"service/pkg/logger"
)

type ZapAdapter struct {
	logger *zap.Logger
}

func NewZapAdapter() (*ZapAdapter, error) {
	config := zap.NewProductionConfig()

	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}
	config.Encoding = "json"

	zapLogger, err := config.Build(
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	)
	if err != nil {
		return nil, err
	}
	return &ZapAdapter{logger: zapLogger}, nil
}

func (z *ZapAdapter) Debug(msg string, fields ...logger.Field) {
	z.logger.Debug(msg, convertFields(fields)...)
}

func (z *ZapAdapter) Info(msg string, fields ...logger.Field) {
	z.logger.Info(msg, convertFields(fields)...)
}

func (z *ZapAdapter) Warn(msg string, fields ...logger.Field) {
	z.logger.Warn(msg, convertFields(fields)...)
}

func (z *ZapAdapter) Error(msg string, fields ...logger.Field) {
	z.logger.Error(msg, convertFields(fields)...)
}

func (z *ZapAdapter) With(fields ...logger.Field) logger.Logger {
	zapFields := convertFields(fields)
	return &ZapAdapter{
		logger: z.logger.With(zapFields...),
	}
}

func (z *ZapAdapter) Sync() error {
	return z.logger.Sync()
}

func convertFields(fields []logger.Field) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, zap.Any(f.Key, f.Value))
	}
	return zapFields
}
