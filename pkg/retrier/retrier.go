package retrier

import (
	"context"
	"time"
)

// Пример интерфейса
type Retrier interface {
	ExecuteWithContext(ctx context.Context, fn func(context.Context) error) error
}

type ShouldRetryFunc func(error) bool

type Config struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	MaxElapsedTime  time.Duration
	Randomization   float64
	Multiplier      float64

	// Если nil - ретраятся все ошибки, если не nil - только те где функция вернула true
	ShouldRetry ShouldRetryFunc
}
