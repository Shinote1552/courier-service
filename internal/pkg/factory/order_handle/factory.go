package order_handle

import (
	"context"
	"fmt"

	"service/internal/entities"
	"service/internal/service/order"
)

type StatusHandlerFactory struct {
	deliveryService order.DeliveryService
}

func NewStatusHandlerFactory(deliveryService order.DeliveryService) *StatusHandlerFactory {
	return &StatusHandlerFactory{
		deliveryService: deliveryService,
	}
}

func (f *StatusHandlerFactory) GetHandler(status entities.OrderStatusType) (order.ExecuteFn, error) {
	switch status {
	case entities.OrderCreated:
		return f.createdHandler, nil
	case entities.OrderCancelled:
		return f.cancelledHandler, nil
	case entities.OrderCompleted:
		return f.completedHandler, nil
	default:
		return nil, fmt.Errorf("%w: %s", order.ErrUndefinedStatus, status)
	}
}

func (f *StatusHandlerFactory) createdHandler(ctx context.Context, orderID string) error {
	_, err := f.deliveryService.DeliveryAssign(ctx, orderID)
	if err != nil {
		return fmt.Errorf("assign courier for created order %s: %w", orderID, err)
	}
	return nil
}

func (f *StatusHandlerFactory) cancelledHandler(ctx context.Context, orderID string) error {
	_, err := f.deliveryService.DeliveryUnassign(ctx, orderID)
	if err != nil {
		return fmt.Errorf("unassign courier for cancelled order %s: %w", orderID, err)
	}
	return nil
}

func (f *StatusHandlerFactory) completedHandler(ctx context.Context, orderID string) error {
	err := f.deliveryService.FreeCourierByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("free courier for completed order %s: %w", orderID, err)
	}
	return nil
}
