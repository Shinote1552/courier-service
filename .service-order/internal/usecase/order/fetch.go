package order

import (
	"context"
	"time"

	"github.com/nikolaev/service-order/internal/domain/entity"
)

func (s *service) Get(ctx context.Context, userID string, id string) (*entity.Order, error) {
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

	if userID != "" && o.UserID != userID {
		return nil, entity.ErrForeignOwnership
	}

	advanceStatus(s.clock.Now(), o)
	return o, nil
}

func (s *service) GetStatus(ctx context.Context, userID string, id string) (entity.OrderStatus, error) {
	o, err := s.Get(ctx, userID, id)
	if err != nil {
		return "", err
	}

	return o.Status, nil
}

func (s *service) ListFrom(ctx context.Context, from time.Time) ([]*entity.Order, error) {
	orders, err := s.repo.ListFrom(ctx, from)
	if err != nil {
		return nil, err
	}

	now := s.clock.Now()
	for _, o := range orders {
		advanceStatus(now, o)
	}

	return orders, nil
}
