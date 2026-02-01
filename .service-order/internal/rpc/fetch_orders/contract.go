package fetch_orders

import (
	"context"
	"github.com/nikolaev/service-order/internal/domain/entity"
	"time"
)

type usecase interface {
	ListFrom(ctx context.Context, from time.Time) ([]*entity.Order, error)
	GetByID(ctx context.Context, id string) (*entity.Order, error)
}
