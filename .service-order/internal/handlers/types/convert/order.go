package convert

import (
	"time"

	"github.com/nikolaev/service-order/internal/domain/entity"
	"github.com/nikolaev/service-order/internal/handlers/types/transport"
	uc "github.com/nikolaev/service-order/internal/usecase/order"
)

func ToDomainCreate(in transport.CreateOrderRequest) uc.CreateInput {
	return uc.CreateInput{
		OrderNumber:  in.OrderNumber,
		FIO:          in.FIO,
		RestaurantID: in.RestaurantID,
		Items:        toDomainItems(in.Items),
		TotalPrice:   in.TotalPrice,
		Address:      toDomainAddress(in.Address),
	}
}

func ToDomainUpdate(in transport.UpdateOrderRequest) uc.UpdateInput {
	var items *[]entity.Item
	if in.Items != nil {
		v := toDomainItems(*in.Items)
		items = &v
	}
	var addr *entity.DeliveryAddress
	if in.Address != nil {
		v := toDomainAddress(*in.Address)
		addr = &v
	}

	return uc.UpdateInput{
		OrderNumber: in.OrderNumber,
		FIO:         in.FIO,
		Items:       items,
		TotalPrice:  in.TotalPrice,
		Address:     addr,
	}
}

func ToTransport(o *entity.Order) transport.OrderResponse {
	return transport.OrderResponse{
		ID:                o.ID,
		UserID:            o.UserID,
		OrderNumber:       o.OrderNumber,
		FIO:               o.FIO,
		RestaurantID:      o.RestaurantID,
		Items:             toTransportItems(o.Items),
		TotalPrice:        o.TotalPrice,
		Address:           toTransportAddress(o.Address),
		Status:            string(o.Status),
		CreatedAt:         o.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         o.UpdatedAt.Format(time.RFC3339),
		EstimatedDelivery: o.EstimatedDelivery.Format(time.RFC3339),
	}
}

func toDomainItems(items []transport.Item) []entity.Item {
	out := make([]entity.Item, 0, len(items))
	for _, it := range items {
		out = append(out, entity.Item{
			FoodID:   it.FoodID,
			Name:     it.Name,
			Quantity: it.Quantity,
			Price:    it.Price,
		})
	}

	return out
}

func toDomainAddress(a transport.DeliveryAddress) entity.DeliveryAddress {
	return entity.DeliveryAddress{
		Street:    a.Street,
		House:     a.House,
		Apartment: a.Apartment,
		Floor:     a.Floor,
		Comment:   a.Comment,
	}
}

func toTransportItems(items []entity.Item) []transport.Item {
	out := make([]transport.Item, 0, len(items))
	for _, it := range items {
		out = append(out, transport.Item{
			FoodID:   it.FoodID,
			Name:     it.Name,
			Quantity: it.Quantity,
			Price:    it.Price,
		})
	}

	return out
}

func toTransportAddress(a entity.DeliveryAddress) transport.DeliveryAddress {
	return transport.DeliveryAddress{
		Street:    a.Street,
		House:     a.House,
		Apartment: a.Apartment,
		Floor:     a.Floor,
		Comment:   a.Comment,
	}
}
