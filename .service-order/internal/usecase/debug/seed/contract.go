package seed

//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

import (
	"context"
	"time"

	"github.com/nikolaev/service-order/internal/domain/entity"
)

type Repository interface {
	Create(ctx context.Context, o *entity.Order) error
}

type Clock interface{ Now() time.Time }

type Service interface {
	Seed(ctx context.Context, userID string, n int) ([]*entity.Order, error)
}

type service struct {
	repo Repository
	clk  Clock
}

func New(repo Repository, clk Clock) Service { return &service{repo: repo, clk: clk} }
