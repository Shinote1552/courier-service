package order

import (
	"context"

	"github.com/nikolaev/service-order/internal/domain/entity"
)

func (s *service) Update(ctx context.Context, userID string, id string, in UpdateInput) (*entity.Order, error) {
	if userID == "" {
		return nil, entity.ErrUnauthorized
	}

	if id == "" {
		return nil, entity.ErrInvalidID
	}

	o, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if o == nil || o.IsDeleted {
		return nil, entity.ErrNotFound
	}

	if o.UserID != userID {
		return nil, entity.ErrForeignOwnership
	}

	if in.OrderNumber != nil {
		o.OrderNumber = *in.OrderNumber
	}

	if in.FIO != nil {
		o.FIO = *in.FIO
	}

	if in.Items != nil {
		o.Items = *in.Items
	}

	if in.TotalPrice != nil {
		o.TotalPrice = *in.TotalPrice
	}

	if in.Address != nil {
		o.Address = *in.Address
	}

	now := s.clock.Now()
	o.UpdatedAt = now
	o.Status = entity.OrderStatusUpdated
	o.StatusChangedAt = now
	advanceStatus(now, o)

	if err := s.repo.Update(ctx, o); err != nil {
		return nil, err
	}
	_ = s.producer.OrderUpdated(ctx, o)

	return o, nil
}
