package order

import (
	"context"
	"errors"
	"fmt"

	"service/internal/entities"
)

type Service struct {
	orderGateway    OrderGateway
	deliveryService DeliveryService
	statusFactory   HandlerFactory
}

func New(orderGateway OrderGateway, deliveryService DeliveryService, statusFactory HandlerFactory) *Service {
	return &Service{
		orderGateway:    orderGateway,
		deliveryService: deliveryService,
		statusFactory:   statusFactory,
	}
}

func (s *Service) ProcessOrderStatusChange(ctx context.Context, orderModify entities.OrderModify) (*entities.Order, error) {
	if orderModify.ID == nil || orderModify.Status == nil {
		return nil, fmt.Errorf("order id and status are required")
	}
	// Верификация через order-service
	order, err := s.orderGateway.GetOrderByID(ctx, *orderModify.ID)
	if err != nil {
		return nil, fmt.Errorf("get order from order-service: %w", err)
	}

	executeFn, err := s.statusFactory.GetHandler(order.Status)
	if err != nil {
		// необрабатыываемые статусы просто пропускаем
		if errors.Is(err, ErrUndefinedStatus) {
			return order, nil
		}
		return order, err
	}

	// Выполняем функцию!
	if err := executeFn(ctx, order.ID); err != nil {
		return nil, err
	}

	return order, nil
}
