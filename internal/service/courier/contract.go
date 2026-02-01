//go:generate mockgen -source=contract.go -destination=./contract_mocks_test.go -package=courier_test
package courier

import (
	"context"

	"service/internal/entities"
)

type Repository interface {
	Create(ctx context.Context, courierModifyEntity entities.CourierModify) (int64, error)
	GetByID(ctx context.Context, id int64) (*entities.Courier, error)
	GetAll(ctx context.Context) ([]entities.Courier, error)
	Update(ctx context.Context, courierModifyEntity entities.CourierModify) (*entities.Courier, error)
}

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}
