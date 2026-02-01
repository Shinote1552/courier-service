package order_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/nikolaev/service-order/internal/domain/entity"
	uc "github.com/nikolaev/service-order/internal/usecase/order"
)

type fixedClock struct{ t time.Time }

func (f fixedClock) Now() time.Time { return f.t }

type nopLog struct{}

func (nopLog) WithFields(ctx context.Context, fields map[string]any) context.Context { return ctx }
func (nopLog) Info(ctx context.Context, args ...any)                                 {}

type nopMetric struct{}

func (nopMetric) Increment(key string) {}

func TestService_Create_ProducesAndPersists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMockRepository(ctrl)
	prod := NewMockProducer(ctrl)

	fixed := time.Date(2025, 8, 31, 12, 0, 0, 0, time.UTC)
	clk := fixedClock{t: fixed}
	svc := uc.NewWithDeps(repo, prod, clk, nopLog{}, nopMetric{})

	in := uc.CreateInput{
		RestaurantID: "rest-1",
		Items: []entity.Item{
			{
				FoodID:   "f1",
				Name:     "Pizza",
				Quantity: 1,
				Price:    500,
			},
		},
		TotalPrice: 500,
		Address: entity.DeliveryAddress{
			Street: "Main",
		},
	}

	// Expect repository create and producer event; accept any order pointer
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	prod.EXPECT().OrderCreated(gomock.Any(), gomock.Any()).Return(nil)

	o, err := svc.Create(context.Background(), "user-1", in)
	assert.NoError(t, err)
	if assert.NotNil(t, o) {
		assert.Equal(t, "user-1", o.UserID)
		assert.False(t, o.CreatedAt.IsZero())
		assert.Equal(t, entity.OrderStatusCreated, o.Status)
	}
}

func TestService_Delete_ProducerCalled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMockRepository(ctrl)
	prod := NewMockProducer(ctrl)

	clk := fixedClock{t: time.Now().UTC()}
	svc := uc.NewWithDeps(repo, prod, clk, nopLog{}, nopMetric{})
	order := &entity.Order{
		ID:     "id-1",
		UserID: "u1",
		Status: entity.OrderStatusCreated,
	}

	repo.EXPECT().GetByID(gomock.Any(), "id-1").Return(order, nil)
	repo.EXPECT().MarkDeleted(gomock.Any(), "id-1", "u1").Return(nil)
	prod.EXPECT().OrderDeleted(gomock.Any(), "id-1", "u1").Return(nil)

	err := svc.Delete(context.Background(), "u1", "id-1")
	assert.NoError(t, err)
}
