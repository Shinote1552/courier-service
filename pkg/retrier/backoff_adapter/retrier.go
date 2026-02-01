package backoff_adapter

import (
	"context"

	"github.com/cenkalti/backoff/v4"
	"service/pkg/retrier"
)

type Retrier struct {
	config retrier.Config
}

func New(config retrier.Config) *Retrier {
	return &Retrier{config: config}
}

func (r *Retrier) ExecuteWithContext(ctx context.Context, fn func(context.Context) error) error {
	b := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(r.config.InitialInterval),
		backoff.WithMaxInterval(r.config.MaxInterval),
		backoff.WithMaxElapsedTime(r.config.MaxElapsedTime),
		backoff.WithRandomizationFactor(r.config.Randomization),
		backoff.WithMultiplier(r.config.Multiplier),
	)

	operation := func() error {
		err := fn(ctx)
		if err != nil && r.config.ShouldRetry != nil && !r.config.ShouldRetry(err) {
			return backoff.Permanent(err)
		}
		return err
	}

	return backoff.Retry(operation, backoff.WithContext(b, ctx))
}
