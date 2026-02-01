//go:generate mockgen -source=contract.go -destination=./contract_mocks_test.go -package=ping_get_test
package ping_get

import (
	"service/pkg/logger"
)

type handlerLogger interface {
	Info(msg string, fields ...logger.Field)
	Warn(msg string, fields ...logger.Field)
	Error(msg string, fields ...logger.Field)
	With(fields ...logger.Field) logger.Logger
}
