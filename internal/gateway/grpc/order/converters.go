package order

import (
	"service/internal/entities"
	proto "service/internal/generated/proto/clients"
)

func toDomainList(resp *proto.GetOrdersResponse) []entities.Order {
	if resp == nil || len(resp.Orders) == 0 {
		return []entities.Order{}
	}

	orders := make([]entities.Order, 0, len(resp.Orders))
	for _, protoOrder := range resp.Orders {
		order := toDomain(protoOrder)
		if order != nil {
			orders = append(orders, *order)
		}
	}
	return orders
}

func toDomain(protoOrder *proto.Order) *entities.Order {
	if protoOrder == nil {
		return nil
	}

	return &entities.Order{
		ID:        protoOrder.Id,
		Status:    entities.OrderStatusType(protoOrder.Status),
		CreatedAt: protoOrder.CreatedAt.AsTime(),
	}
}
