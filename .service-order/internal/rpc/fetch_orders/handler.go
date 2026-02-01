package fetch_orders

import (
	"context"
	"github.com/nikolaev/service-order/internal/domain/entity"
	pb "github.com/nikolaev/service-order/internal/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log"
	"time"
)

type Handler struct {
	pb.UnimplementedOrdersServiceServer
	useCase usecase
}

func New(u usecase) *Handler {
	return &Handler{
		useCase: u,
	}
}

func (h *Handler) GetOrders(ctx context.Context, in *pb.GetOrdersRequest) (*pb.GetOrdersResponse, error) {
	from := time.Time{}
	if in != nil && in.From != nil {
		from = in.From.AsTime()
	}

	orders, err := h.useCase.ListFrom(ctx, from)
	if err != nil {
		log.Printf("from GetOrders gRPC: %v", err)
		return nil, err
	}

	return &pb.GetOrdersResponse{
		Orders: toOrderDtoResponse(orders),
	}, nil
}

func (h *Handler) GetOrderById(ctx context.Context, in *pb.GetOrderByIdRequest) (*pb.GetOrderByIdResponse, error) {
	order, err := h.useCase.GetByID(ctx, in.Id)
	if err != nil {
		log.Printf("from GetOrderById gRPC: %v", err)
		return nil, err
	}

	return &pb.GetOrderByIdResponse{
		Order: toOrderDtoResponse([]*entity.Order{order})[0],
	}, nil
}

func toOrderDtoResponse(orders []*entity.Order) []*pb.Order {
	resp := make([]*pb.Order, 0, len(orders))
	for _, order := range orders {
		resp = append(resp, &pb.Order{
			Id:                order.ID,
			UserId:            order.UserID,
			OrderNumber:       order.OrderNumber,
			Fio:               order.FIO,
			RestaurantId:      order.RestaurantID,
			Items:             toItemsDtoResponse(order.Items),
			TotalPrice:        order.TotalPrice,
			Address:           toAddressDtoResponse(order.Address),
			Status:            string(order.Status),
			CreatedAt:         timestamppb.New(order.CreatedAt),
			UpdatedAt:         timestamppb.New(order.UpdatedAt),
			EstimatedDelivery: timestamppb.New(order.EstimatedDelivery),
		})
	}

	return resp
}

func toItemsDtoResponse(items []entity.Item) []*pb.Item {
	resp := make([]*pb.Item, 0, len(items))
	for _, item := range items {
		resp = append(resp, &pb.Item{
			Name:     item.Name,
			Price:    int64(item.Price),
			Quantity: int64(item.Quantity),
		})
	}

	return resp
}

func toAddressDtoResponse(address entity.DeliveryAddress) *pb.DeliveryAddress {
	return &pb.DeliveryAddress{
		Street:    address.Street,
		House:     address.House,
		Apartment: address.Apartment,
		Floor:     address.Floor,
		Comment:   address.Comment,
	}
}
