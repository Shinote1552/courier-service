package order

//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

import (
	"context"
	"time"

	"github.com/nikolaev/service-order/internal/domain/entity"
)

type Repository interface {
	Create(ctx context.Context, o *entity.Order) error
	GetByID(ctx context.Context, id string) (*entity.Order, error)
	Update(ctx context.Context, o *entity.Order) error
	MarkDeleted(ctx context.Context, id string, userID string) error
	ListFrom(ctx context.Context, from time.Time) ([]*entity.Order, error)
}

type Producer interface {
	OrderCreated(ctx context.Context, o *entity.Order) error
	OrderUpdated(ctx context.Context, o *entity.Order) error
	OrderDeleted(ctx context.Context, id string, userID string) error
}

type Service interface {
	Create(ctx context.Context, userID string, in CreateInput) (*entity.Order, error)
	Get(ctx context.Context, userID string, id string) (*entity.Order, error)
	GetStatus(ctx context.Context, userID string, id string) (entity.OrderStatus, error)
	ListFrom(ctx context.Context, from time.Time) ([]*entity.Order, error)
	Update(ctx context.Context, userID string, id string, in UpdateInput) (*entity.Order, error)
	Delete(ctx context.Context, userID string, id string) error
}

type CreateInput struct {
	OrderNumber  string
	FIO          string
	RestaurantID string
	Items        []entity.Item
	TotalPrice   int64
	Address      entity.DeliveryAddress
}

type UpdateInput struct {
	OrderNumber *string
	FIO         *string
	Items       *[]entity.Item
	TotalPrice  *int64
	Address     *entity.DeliveryAddress
}

type Clock interface{ Now() time.Time }

type log interface {
	WithFields(ctx context.Context, fields map[string]any) context.Context
	Info(ctx context.Context, args ...any)
}

type metric interface{ Increment(key string) }

type service struct {
	repo     Repository
	producer Producer
	clock    Clock
	log      log
	metric   metric
}

func New(repo Repository, producer Producer) Service {
	return NewWithDeps(repo, producer, systemClock{}, noopLog{}, noopMetric{})
}

func NewWithDeps(repo Repository, producer Producer, clk Clock, l log, m metric) Service {
	return &service{repo: repo, producer: producer, clock: clk, log: l, metric: m}
}

func advanceStatus(now time.Time, o *entity.Order) {
	dur := now.Sub(o.CreatedAt)
	switch {
	case o.Status == entity.OrderStatusCanceled || o.Status == entity.OrderStatusDeleted:
		return
	case dur >= 10*time.Minute:
		o.Status = entity.OrderStatusDelivering
	case dur >= 1*time.Minute:
		if o.Status == entity.OrderStatusPending || o.Status == entity.OrderStatusCreated {
			o.Status = entity.OrderStatusCooking
		}
	default:
		if o.Status == "" {
			o.Status = entity.OrderStatusPending
		}
	}
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

type noopLog struct{}

func (noopLog) WithFields(ctx context.Context, fields map[string]any) context.Context { return ctx }
func (noopLog) Info(ctx context.Context, args ...any)                                 {}

type noopMetric struct{}

func (noopMetric) Increment(key string) {}
