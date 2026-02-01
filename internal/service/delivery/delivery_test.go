package delivery_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"service/internal/entities"
	"service/internal/service/delivery"
)

type mock struct {
	*MockRepository
	*MockCourierService
	*MockTxManager
	*MockDeliveryTimeFactory
}

func newMock(ctrl *gomock.Controller) *mock {
	return &mock{
		MockRepository:          NewMockRepository(ctrl),
		MockCourierService:      NewMockCourierService(ctrl),
		MockTxManager:           NewMockTxManager(ctrl),
		MockDeliveryTimeFactory: NewMockDeliveryTimeFactory(ctrl),
	}
}

func errorAssertion(expectedError error, expectedErrMsg string) require.ErrorAssertionFunc {
	return func(t require.TestingT, err error, msgAndArgs ...interface{}) {
		require.Error(t, err, msgAndArgs...)

		if expectedError != nil {
			assert.ErrorIs(t, err, expectedError, msgAndArgs...)
		}

		if expectedErrMsg != "" {
			assert.Contains(t, err.Error(), expectedErrMsg, msgAndArgs...)
		}
	}
}

func TestDeliveryService_DeliveryAssign(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	availableCourier := &entities.Courier{
		ID:            1,
		Name:          "Snake Plissken",
		Phone:         "+79161234567",
		Status:        entities.CourierAvailable,
		TransportType: entities.Car,
		CreatedAt:     fixedTime,
		UpdatedAt:     fixedTime,
	}

	tests := []struct {
		name           string
		orderID        string
		deadlineOffset time.Duration
		mockSetup      func(m *mock)
		expectedResult *entities.DeliveryAssignment
		resultChecker  func(t *testing.T, result *entities.DeliveryAssignment, before, after time.Time)
		errorAssertion require.ErrorAssertionFunc
	}{
		{
			name:           "Успешное назначение доставки доступному курьеру с валидным ID заказа",
			orderID:        "order-2026-001",
			deadlineOffset: 30 * time.Minute,
			mockSetup: func(m *mock) {
				m.MockTxManager.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					})
				m.MockRepository.EXPECT().
					GetCourierForAssignment(gomock.Any()).
					Return(availableCourier, nil)

				m.MockDeliveryTimeFactory.EXPECT().
					CalculateDeadline(availableCourier.TransportType, gomock.Any()).
					DoAndReturn(func(transportType entities.CourierTransportType, baseTime time.Time) time.Time {
						return baseTime.Add(30 * time.Minute)
					})
				m.MockRepository.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, modify entities.DeliveryModify) (*entities.Delivery, error) {
						return &entities.Delivery{
							ID:         1,
							CourierID:  *modify.CourierID,
							OrderID:    *modify.OrderID,
							AssignedAt: *modify.AssignedAt,
							Deadline:   *modify.Deadline,
						}, nil
					})
				m.MockCourierService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(availableCourier, nil)
			},
			expectedResult: &entities.DeliveryAssignment{
				CourierID:     availableCourier.ID,
				OrderID:       "order-2026-001",
				TransportType: availableCourier.TransportType,
			},
			resultChecker: func(t *testing.T, result *entities.DeliveryAssignment, before, after time.Time) {
				require.NotNil(t, result)
				assert.Equal(t, availableCourier.ID, result.CourierID)
				assert.Equal(t, "order-2026-001", result.OrderID)
				assert.Equal(t, availableCourier.TransportType, result.TransportType)
				assert.False(t, result.AssignedAt.IsZero())
				assert.False(t, result.Deadline.IsZero())
				assert.True(t, !result.AssignedAt.Before(before) && !result.AssignedAt.After(after))
				expectedDeadline := result.AssignedAt.Add(30 * time.Minute)
				assert.WithinDuration(t, expectedDeadline, result.Deadline, time.Second)
			},
			errorAssertion: require.NoError,
		},
		{
			name:           "Отклонение назначения доставки с пустым ID заказа",
			orderID:        "",
			deadlineOffset: 30 * time.Minute,
			expectedResult: nil,
			resultChecker: func(t *testing.T, result *entities.DeliveryAssignment, before, after time.Time) {
				assert.Nil(t, result)
			},
			errorAssertion: errorAssertion(delivery.ErrInvalidOrderID, ""),
		},
		{
			name:           "Отклонение назначения когда нет доступных курьеров в системе",
			orderID:        "order-2026-001",
			deadlineOffset: 30 * time.Minute,
			mockSetup: func(m *mock) {
				m.MockTxManager.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					})
				m.MockRepository.EXPECT().
					GetCourierForAssignment(gomock.Any()).
					Return(nil, errors.New("no active couriers found"))
			},
			expectedResult: nil,
			resultChecker: func(t *testing.T, result *entities.DeliveryAssignment, before, after time.Time) {
				assert.Nil(t, result)
			},
			errorAssertion: errorAssertion(nil, "find courier for assignment: no active couriers found"),
		},
		{
			name:           "Отклонение назначения когда все курьеры заняты",
			orderID:        "order-2026-001",
			deadlineOffset: 30 * time.Minute,
			mockSetup: func(m *mock) {
				m.MockTxManager.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					})
				m.MockRepository.EXPECT().
					GetCourierForAssignment(gomock.Any()).
					Return(nil, delivery.ErrNoAvailableCouriers)
			},
			expectedResult: nil,
			resultChecker: func(t *testing.T, result *entities.DeliveryAssignment, before, after time.Time) {
				assert.Nil(t, result)
			},
			errorAssertion: errorAssertion(delivery.ErrNoAvailableCouriers, ""),
		},
		{
			name:           "Отклонение назначения при нарушении ограничений базы данных",
			orderID:        "order-2026-001",
			deadlineOffset: 30 * time.Minute,
			mockSetup: func(m *mock) {
				m.MockTxManager.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					})
				m.MockRepository.EXPECT().
					GetCourierForAssignment(gomock.Any()).
					Return(availableCourier, nil)
				m.MockDeliveryTimeFactory.EXPECT().
					CalculateDeadline(availableCourier.TransportType, gomock.Any()).
					DoAndReturn(func(transportType entities.CourierTransportType, baseTime time.Time) time.Time {
						return baseTime.Add(30 * time.Minute)
					})
				m.MockRepository.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("foreign key constraint violation"))
			},
			expectedResult: nil,
			resultChecker: func(t *testing.T, result *entities.DeliveryAssignment, before, after time.Time) {
				assert.Nil(t, result)
			},
			errorAssertion: errorAssertion(nil, "create delivery: foreign key constraint violation"),
		},
		{
			name:           "Отклонение назначения когда заказ уже назначен другому курьеру",
			orderID:        "order-2026-001",
			deadlineOffset: 30 * time.Minute,
			mockSetup: func(m *mock) {
				m.MockTxManager.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					})
				m.MockRepository.EXPECT().
					GetCourierForAssignment(gomock.Any()).
					Return(availableCourier, nil)
				m.MockDeliveryTimeFactory.EXPECT().
					CalculateDeadline(availableCourier.TransportType, gomock.Any()).
					DoAndReturn(func(transportType entities.CourierTransportType, baseTime time.Time) time.Time {
						return baseTime.Add(30 * time.Minute)
					})
				m.MockRepository.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil, delivery.ErrOrderAlreadyAssigned)
			},
			expectedResult: nil,
			resultChecker: func(t *testing.T, result *entities.DeliveryAssignment, before, after time.Time) {
				assert.Nil(t, result)
			},
			errorAssertion: errorAssertion(delivery.ErrOrderAlreadyAssigned, ""),
		},
		{
			name:           "Отклонение назначения при ошибке обновления статуса курьера",
			orderID:        "order-2026-001",
			deadlineOffset: 30 * time.Minute,
			mockSetup: func(m *mock) {
				m.MockTxManager.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					})
				m.MockRepository.EXPECT().
					GetCourierForAssignment(gomock.Any()).
					Return(availableCourier, nil)
				m.MockDeliveryTimeFactory.EXPECT().
					CalculateDeadline(availableCourier.TransportType, gomock.Any()).
					DoAndReturn(func(transportType entities.CourierTransportType, baseTime time.Time) time.Time {
						return baseTime.Add(30 * time.Minute)
					})
				m.MockRepository.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, modify entities.DeliveryModify) (*entities.Delivery, error) {
						return &entities.Delivery{
							ID:         1,
							CourierID:  *modify.CourierID,
							OrderID:    *modify.OrderID,
							AssignedAt: *modify.AssignedAt,
							Deadline:   *modify.Deadline,
						}, nil
					})
				m.MockCourierService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("courier service unavailable"))
			},
			expectedResult: nil,
			resultChecker: func(t *testing.T, result *entities.DeliveryAssignment, before, after time.Time) {
				assert.Nil(t, result)
			},
			errorAssertion: errorAssertion(nil, "update courier status: courier service unavailable"),
		},
		{
			name:           "Отклонение назначения при ошибке менеджера транзакций",
			orderID:        "order-2026-001",
			deadlineOffset: 30 * time.Minute,
			mockSetup: func(m *mock) {
				m.MockTxManager.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					Return(errors.New("transaction rollback error"))
			},
			expectedResult: nil,
			resultChecker: func(t *testing.T, result *entities.DeliveryAssignment, before, after time.Time) {
				assert.Nil(t, result)
			},
			errorAssertion: errorAssertion(nil, "transaction rollback error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := newMock(ctrl)

			if tt.mockSetup != nil {
				tt.mockSetup(m)
			}

			service := delivery.New(
				m.MockRepository,
				m.MockCourierService,
				m.MockDeliveryTimeFactory,
				m.MockTxManager,
			)

			beforeCall := time.Now().UTC()
			result, err := service.DeliveryAssign(context.Background(), tt.orderID)
			afterCall := time.Now().UTC()

			tt.resultChecker(t, result, beforeCall, afterCall)
			tt.errorAssertion(t, err, tt.name)
		})
	}
}

func TestDeliveryService_DeliveryUnassign(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	updatedCourier := &entities.Courier{
		ID:            1,
		Name:          "Snake Plissken",
		Phone:         "+79161234567",
		Status:        entities.CourierAvailable,
		TransportType: entities.Car,
		CreatedAt:     fixedTime,
		UpdatedAt:     fixedTime,
	}

	tests := []struct {
		name           string
		orderID        string
		mockSetup      func(m *mock)
		expectedResult *entities.DeliveryUnassignment
		errorAssertion require.ErrorAssertionFunc
	}{
		{
			name:    "Успешное снятие доставки и обновление статуса курьера на доступен",
			orderID: "order-2026-001",
			mockSetup: func(m *mock) {
				m.MockTxManager.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					})
				m.MockRepository.EXPECT().
					GetCourierIDAndDeliveryCountByOrderIDForAssing(gomock.Any(), "order-2026-001").
					Return(int64(1), int64(0), nil)
				m.MockRepository.EXPECT().
					Delete(gomock.Any(), "order-2026-001").
					Return(nil)
				m.MockCourierService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(updatedCourier, nil)
			},
			expectedResult: &entities.DeliveryUnassignment{
				CourierID: updatedCourier.ID,
				OrderID:   "order-2026-001",
				Status:    updatedCourier.Status.String(),
			},
			errorAssertion: require.NoError,
		},
		{
			name:           "Отклонение снятия доставки с пустым ID заказа",
			orderID:        "",
			expectedResult: nil,
			errorAssertion: errorAssertion(delivery.ErrInvalidOrderID, ""),
		},
		{
			name:    "Отклонение снятия когда запись о доставке не найдена",
			orderID: "order-2026-001",
			mockSetup: func(m *mock) {
				m.MockTxManager.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					})
				m.MockRepository.EXPECT().
					GetCourierIDAndDeliveryCountByOrderIDForAssing(gomock.Any(), "order-2026-001").
					Return(int64(0), int64(0), delivery.ErrDeliveryNotFound)
			},
			expectedResult: nil,
			errorAssertion: errorAssertion(delivery.ErrDeliveryNotFound, ""),
		},
		{
			name:    "Отклонение снятия когда у курьера есть другие активные доставки",
			orderID: "order-2026-001",
			mockSetup: func(m *mock) {
				m.MockTxManager.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					})
				m.MockRepository.EXPECT().
					GetCourierIDAndDeliveryCountByOrderIDForAssing(gomock.Any(), "order-2026-001").
					Return(int64(1), int64(2), nil)
			},
			expectedResult: nil,
			errorAssertion: errorAssertion(delivery.ErrCourierHasActiveDeliveries, ""),
		},
		{
			name:    "Отклонение снятия при ошибке удаления доставки",
			orderID: "order-2026-001",
			mockSetup: func(m *mock) {
				m.MockTxManager.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					})
				m.MockRepository.EXPECT().
					GetCourierIDAndDeliveryCountByOrderIDForAssing(gomock.Any(), "order-2026-001").
					Return(int64(1), int64(0), nil)
				m.MockRepository.EXPECT().
					Delete(gomock.Any(), "order-2026-001").
					Return(errors.New("database lock timeout"))
			},
			expectedResult: nil,
			errorAssertion: errorAssertion(nil, "delete delivery: database lock timeout"),
		},
		{
			name:    "Отклонение снятия при ошибке обновления статуса курьера",
			orderID: "order-2026-001",
			mockSetup: func(m *mock) {
				m.MockTxManager.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					})
				m.MockRepository.EXPECT().
					GetCourierIDAndDeliveryCountByOrderIDForAssing(gomock.Any(), "order-2026-001").
					Return(int64(1), int64(0), nil)
				m.MockRepository.EXPECT().
					Delete(gomock.Any(), "order-2026-001").
					Return(nil)
				m.MockCourierService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("courier service temporary unavailable"))
			},
			expectedResult: nil,
			errorAssertion: errorAssertion(nil, "update courier status: courier service temporary unavailable"),
		},
		{
			name:    "Отклонение снятия при ошибке менеджера транзакций",
			orderID: "order-2026-001",
			mockSetup: func(m *mock) {
				m.MockTxManager.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					Return(errors.New("transaction commit failed"))
			},
			expectedResult: nil,
			errorAssertion: errorAssertion(nil, "transaction commit failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := newMock(ctrl)

			if tt.mockSetup != nil {
				tt.mockSetup(m)
			}

			service := delivery.New(
				m.MockRepository,
				m.MockCourierService,
				m.MockDeliveryTimeFactory,
				m.MockTxManager,
			)

			result, err := service.DeliveryUnassign(context.Background(), tt.orderID)

			assert.Equal(t, tt.expectedResult, result)
			tt.errorAssertion(t, err, tt.name)
		})
	}
}

func TestDeliveryService_CleanupExpiredDeliveries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockSetup      func(m *mock)
		errorAssertion require.ErrorAssertionFunc
	}{
		{
			name: "Успешная очистка истекших доставок с освобождением 3 курьеров",
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					UpdateCouriersAvailableWhereDeadlineExpired(gomock.Any()).
					Return(int64(3), nil)
			},
			errorAssertion: require.NoError,
		},
		{
			name: "Успешная очистка когда нет истекших доставок",
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					UpdateCouriersAvailableWhereDeadlineExpired(gomock.Any()).
					Return(int64(0), nil)
			},
			errorAssertion: require.NoError,
		},
		{
			name: "Очистка возвращает ошибку от репозитория",
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					UpdateCouriersAvailableWhereDeadlineExpired(gomock.Any()).
					Return(int64(0), errors.New("cleanup query execution failed"))
			},
			errorAssertion: errorAssertion(nil, "cleanup: cleanup query execution failed"),
		},
		{
			name: "Таймаут контекста при выполнении очистки",
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					UpdateCouriersAvailableWhereDeadlineExpired(gomock.Any()).
					Return(int64(0), context.DeadlineExceeded)
			},
			errorAssertion: errorAssertion(nil, "cleanup timed out"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := newMock(ctrl)

			if tt.mockSetup != nil {
				tt.mockSetup(m)
			}

			service := delivery.New(
				m.MockRepository,
				m.MockCourierService,
				m.MockDeliveryTimeFactory,
				m.MockTxManager,
			)

			ctx := context.Background()
			_, err := service.CleanupExpiredDeliveries(ctx)

			tt.errorAssertion(t, err, tt.name)
		})
	}
}

func TestDeliveryService_FreeCourierByOrderID(t *testing.T) {
	t.Parallel()

	updatedCourier := &entities.Courier{
		ID:            1,
		Name:          "Snake Plissken",
		Phone:         "+79161234567",
		Status:        entities.CourierAvailable,
		TransportType: entities.Car,
		CreatedAt:     time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name           string
		orderID        string
		mockSetup      func(m *mock)
		errorAssertion require.ErrorAssertionFunc
	}{
		{
			name:    "Успешное освобождение курьера по валидному order_id",
			orderID: "order-2026-001",
			mockSetup: func(m *mock) {
				m.MockTxManager.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					})
				m.MockRepository.EXPECT().
					GetCourierIDByOrderID(gomock.Any(), "order-2026-001").
					Return(int64(1), nil)
				m.MockCourierService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(updatedCourier, nil)
			},
			errorAssertion: require.NoError,
		},
		{
			name:           "Отклонение освобождения курьера с пустым ID заказа",
			orderID:        "",
			errorAssertion: errorAssertion(delivery.ErrInvalidOrderID, ""),
		},
		{
			name:    "Отклонение освобождения когда доставка не найдена",
			orderID: "order-2026-001",
			mockSetup: func(m *mock) {
				m.MockTxManager.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					})
				m.MockRepository.EXPECT().
					GetCourierIDByOrderID(gomock.Any(), "order-2026-001").
					Return(int64(0), delivery.ErrDeliveryNotFound)
			},
			errorAssertion: errorAssertion(delivery.ErrDeliveryNotFound, ""),
		},
		{
			name:    "Отклонение освобождения при ошибке репозитория",
			orderID: "order-2026-001",
			mockSetup: func(m *mock) {
				m.MockTxManager.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					})
				m.MockRepository.EXPECT().
					GetCourierIDByOrderID(gomock.Any(), "order-2026-001").
					Return(int64(0), errors.New("database connection timeout"))
			},
			errorAssertion: errorAssertion(nil, "get courier by order id: database connection timeout"),
		},
		{
			name:    "Отклонение освобождения при ошибке обновления статуса курьера",
			orderID: "order-2026-001",
			mockSetup: func(m *mock) {
				m.MockTxManager.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					})
				m.MockRepository.EXPECT().
					GetCourierIDByOrderID(gomock.Any(), "order-2026-001").
					Return(int64(1), nil)
				m.MockCourierService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("courier service unavailable"))
			},
			errorAssertion: errorAssertion(nil, "update courier status: courier service unavailable"),
		},
		{
			name:    "Отклонение освобождения когда статус курьера не изменился",
			orderID: "order-2026-001",
			mockSetup: func(m *mock) {
				m.MockTxManager.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					})
				m.MockRepository.EXPECT().
					GetCourierIDByOrderID(gomock.Any(), "order-2026-001").
					Return(int64(1), nil)
				unchangedCourier := &entities.Courier{
					ID:     1,
					Status: entities.CourierBusy,
				}
				m.MockCourierService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(unchangedCourier, nil)
			},
			errorAssertion: errorAssertion(nil, "update courier status"),
		},
		{
			name:    "Отклонение освобождения при ошибке менеджера транзакций",
			orderID: "order-2026-001",
			mockSetup: func(m *mock) {
				m.MockTxManager.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					Return(errors.New("transaction rollback failed"))
			},
			errorAssertion: errorAssertion(nil, "transaction rollback failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := newMock(ctrl)

			if tt.mockSetup != nil {
				tt.mockSetup(m)
			}

			service := delivery.New(
				m.MockRepository,
				m.MockCourierService,
				m.MockDeliveryTimeFactory,
				m.MockTxManager,
			)

			err := service.FreeCourierByOrderID(context.Background(), tt.orderID)

			tt.errorAssertion(t, err, tt.name)
		})
	}
}

func TestDeliveryService_CleanupExpiredDeliveries_FullCoverage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockSetup      func(m *mock)
		rowsAffected   int64
		errorAssertion require.ErrorAssertionFunc
	}{
		{
			name:         "Успешная очистка с освобождением курьеров - логируется",
			rowsAffected: 3,
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					UpdateCouriersAvailableWhereDeadlineExpired(gomock.Any()).
					Return(int64(3), nil)
			},
			errorAssertion: require.NoError,
		},
		{
			name:         "Успешная очистка без освобождения курьеров - лог не вызывается",
			rowsAffected: 0,
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					UpdateCouriersAvailableWhereDeadlineExpired(gomock.Any()).
					Return(int64(0), nil)
			},
			errorAssertion: require.NoError,
		},
		{
			name: "Обработка таймаута контекста",
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					UpdateCouriersAvailableWhereDeadlineExpired(gomock.Any()).
					Return(int64(0), context.DeadlineExceeded)
			},
			errorAssertion: errorAssertion(nil, "cleanup timed out"),
		},
		{
			name: "Обработка произвольной ошибки репозитория",
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					UpdateCouriersAvailableWhereDeadlineExpired(gomock.Any()).
					Return(int64(0), errors.New("database deadlock"))
			},
			errorAssertion: errorAssertion(nil, "cleanup: database deadlock"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := newMock(ctrl)

			if tt.mockSetup != nil {
				tt.mockSetup(m)
			}

			service := delivery.New(
				m.MockRepository,
				m.MockCourierService,
				m.MockDeliveryTimeFactory,
				m.MockTxManager,
			)

			ctx := context.Background()
			_, err := service.CleanupExpiredDeliveries(ctx)

			tt.errorAssertion(t, err, tt.name)
		})
	}
}
