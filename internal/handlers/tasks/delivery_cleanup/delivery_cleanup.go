package delivery_cleanup

import (
	"context"
	"time"

	"service/pkg/logger"
)

type Service interface {
	CleanupExpiredDeliveries(ctx context.Context) (int64, error)
}

type DeliveryCleanup struct {
	log      logger.Logger
	service  Service
	interval time.Duration
}

func NewDeliveryCleanup(log logger.Logger, service Service, interval time.Duration) *DeliveryCleanup {
	return &DeliveryCleanup{
		log:      log,
		service:  service,
		interval: interval,
	}
}

func (d *DeliveryCleanup) TTL() time.Duration {
	return d.interval
}

func (d *DeliveryCleanup) Do(ctx context.Context) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, d.interval)
	defer cancel()

	rowsAffected, err := d.service.CleanupExpiredDeliveries(ctxWithTimeout)

	if rowsAffected > 0 {
		d.log.With(
			logger.NewField("expired_couriers", rowsAffected),
		).Info("delivery cleanup")
	}

	return err
}

func (d *DeliveryCleanup) Info() string {
	return "delivery cleanup"
}
