package order_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"service/internal/entities"
	"service/internal/pkg/factory/order_handle"
	service_order "service/internal/service/order"
)

type mock struct {
	MockOrderGateway    *MockOrderGateway
	MockDeliveryService *MockDeliveryService
	MockHandlerFactory  *MockHandlerFactory
}

func newMock(ctrl *gomock.Controller) *mock {
	return &mock{
		MockOrderGateway:    NewMockOrderGateway(ctrl),
		MockDeliveryService: NewMockDeliveryService(ctrl),
		MockHandlerFactory:  NewMockHandlerFactory(ctrl),
	}
}

func errorAssertion(expectedError error, expectedErrMsg string) require.ErrorAssertionFunc {
	return func(t require.TestingT, err error, msgAndArgs ...interface{}) {
		if expectedError != nil || expectedErrMsg != "" {
			require.Error(t, err, msgAndArgs...)
			if expectedError != nil {
				assert.ErrorIs(t, err, expectedError, msgAndArgs...)
			}
			if expectedErrMsg != "" {
				assert.Contains(t, err.Error(), expectedErrMsg, msgAndArgs...)
			}
		} else {
			require.NoError(t, err, msgAndArgs...)
		}
	}
}

func TestServiceProcessOrderStatusChange(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		orderModify    entities.OrderModify
		mockSetup      func(m *mock)
		expectedOrder  *entities.Order
		errorAssertion require.ErrorAssertionFunc
	}{
		{
			name: "нет ID",
			orderModify: entities.OrderModify{
				Status: pointer.To(entities.OrderCreated),
			},
			expectedOrder:  nil,
			errorAssertion: errorAssertion(nil, "order id and status are required"),
		},
		{
			name: "нет статуса",
			orderModify: entities.OrderModify{
				ID: pointer.To("order-2026-001"),
			},
			expectedOrder:  nil,
			errorAssertion: errorAssertion(nil, "order id and status are required"),
		},
		{
			name: "заказ не найден",
			orderModify: entities.OrderModify{
				ID:     pointer.To("order-not-found"),
				Status: pointer.To(entities.OrderCreated),
			},
			mockSetup: func(m *mock) {
				m.MockOrderGateway.EXPECT().
					GetOrderByID(gomock.Any(), "order-not-found").
					Return(nil, service_order.ErrOrderNotFound)
			},
			expectedOrder:  nil,
			errorAssertion: errorAssertion(service_order.ErrOrderNotFound, "get order from order-service"),
		},
		{
			name: "создан - успешно",
			orderModify: entities.OrderModify{
				ID:     pointer.To("order-2026-001"),
				Status: pointer.To(entities.OrderCreated),
			},
			mockSetup: func(m *mock) {
				order := &entities.Order{
					ID:        "order-2026-001",
					Status:    entities.OrderCreated,
					CreatedAt: fixedTime,
				}
				m.MockOrderGateway.EXPECT().
					GetOrderByID(gomock.Any(), "order-2026-001").
					Return(order, nil)

				m.MockHandlerFactory.EXPECT().
					GetHandler(entities.OrderCreated).
					Return(
						func(ctx context.Context, orderID string) error {
							return nil
						},
						nil,
					)
			},
			expectedOrder: &entities.Order{
				ID:        "order-2026-001",
				Status:    entities.OrderCreated,
				CreatedAt: fixedTime,
			},
			errorAssertion: require.NoError,
		},
		{
			name: "несовпадение статусов",
			orderModify: entities.OrderModify{
				ID:     pointer.To("order-2026-001"),
				Status: pointer.To(entities.OrderCancelled),
			},
			mockSetup: func(m *mock) {
				order := &entities.Order{
					ID:        "order-2026-001",
					Status:    entities.OrderCreated,
					CreatedAt: fixedTime,
				}
				m.MockOrderGateway.EXPECT().
					GetOrderByID(gomock.Any(), "order-2026-001").
					Return(order, nil)

				m.MockHandlerFactory.EXPECT().
					GetHandler(entities.OrderCreated).
					Return(
						func(ctx context.Context, orderID string) error {
							return nil
						},
						nil,
					)
			},
			expectedOrder: &entities.Order{
				ID:        "order-2026-001",
				Status:    entities.OrderCreated,
				CreatedAt: fixedTime,
			},
			errorAssertion: require.NoError,
		},
		{
			name: "неизвестный статус",
			orderModify: func() entities.OrderModify {
				invalidStatus := entities.OrderStatusType("invalid")
				return entities.OrderModify{
					ID:     pointer.To("order-2026-001"),
					Status: pointer.To(invalidStatus),
				}
			}(),
			mockSetup: func(m *mock) {
				invalidOrder := &entities.Order{
					ID:        "order-2026-001",
					Status:    entities.OrderStatusType("invalid"),
					CreatedAt: fixedTime,
				}
				m.MockOrderGateway.EXPECT().
					GetOrderByID(gomock.Any(), "order-2026-001").
					Return(invalidOrder, nil)

				m.MockHandlerFactory.EXPECT().
					GetHandler(entities.OrderStatusType("invalid")).
					Return(nil, service_order.ErrUndefinedStatus)
			},
			expectedOrder: &entities.Order{
				ID:        "order-2026-001",
				Status:    entities.OrderStatusType("invalid"),
				CreatedAt: fixedTime,
			},
			errorAssertion: require.NoError,
		},
		{
			name: "ошибка выполнения обработчика",
			orderModify: entities.OrderModify{
				ID:     pointer.To("order-2026-001"),
				Status: pointer.To(entities.OrderCreated),
			},
			mockSetup: func(m *mock) {
				order := &entities.Order{
					ID:        "order-2026-001",
					Status:    entities.OrderCreated,
					CreatedAt: fixedTime,
				}
				m.MockOrderGateway.EXPECT().
					GetOrderByID(gomock.Any(), "order-2026-001").
					Return(order, nil)

				m.MockHandlerFactory.EXPECT().
					GetHandler(entities.OrderCreated).
					Return(
						func(ctx context.Context, orderID string) error {
							return errors.New("handler execution failed")
						},
						nil,
					)
			},
			expectedOrder:  nil,
			errorAssertion: errorAssertion(nil, "handler execution failed"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := newMock(ctrl)
			if tt.mockSetup != nil {
				tt.mockSetup(m)
			}

			service := service_order.New(m.MockOrderGateway, m.MockDeliveryService, m.MockHandlerFactory)

			result, err := service.ProcessOrderStatusChange(context.Background(), tt.orderModify)
			assert.Equal(t, tt.expectedOrder, result)
			tt.errorAssertion(t, err, tt.name)
		})
	}
}

func TestStatusHandlerFactoryGetHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		status         entities.OrderStatusType
		expectedErrMsg string
	}{
		{
			name:   "создан",
			status: entities.OrderCreated,
		},
		{
			name:   "отменен",
			status: entities.OrderCancelled,
		},
		{
			name:   "выполнен",
			status: entities.OrderCompleted,
		},
		{
			name:           "неизвестный статус",
			status:         entities.OrderStatusType("invalid"),
			expectedErrMsg: "undefined order status",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := NewMockDeliveryService(ctrl)
			factory := order_handle.NewStatusHandlerFactory(m)

			_, err := factory.GetHandler(tt.status)
			if tt.expectedErrMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
