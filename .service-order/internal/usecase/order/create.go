package order

import (
	"context"

	"github.com/google/uuid"

	"github.com/nikolaev/service-order/internal/domain/entity"
)

func (s *service) Create(ctx context.Context, userID string, in CreateInput) (*entity.Order, error) {
	if userID == "" {
		return nil, entity.ErrUnauthorized
	}
	if in.RestaurantID == "" || len(in.Items) == 0 || in.TotalPrice < 0 {
		return nil, entity.ErrInvalidInput
	}
	now := s.clock.Now()
	o := &entity.Order{
		ID:              uuid.NewString(),
		UserID:          userID,
		OrderNumber:     in.OrderNumber,
		FIO:             in.FIO,
		RestaurantID:    in.RestaurantID,
		Items:           in.Items,
		TotalPrice:      in.TotalPrice,
		Address:         in.Address,
		Status:          entity.OrderStatusCreated,
		CreatedAt:       now,
		UpdatedAt:       now,
		StatusChangedAt: now,
	}
	advanceStatus(now, o)
	if err := s.repo.Create(ctx, o); err != nil {
		return nil, err
	}
	_ = s.producer.OrderCreated(ctx, o)

	return o, nil
}
