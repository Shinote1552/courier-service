package order_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"service/internal/entities"
	"service/internal/gateway/grpc/order"
	proto "service/internal/generated/proto/clients"
)

type mock struct {
	*Mockclient
}

func newMock(ctrl *gomock.Controller) *mock {
	return &mock{
		Mockclient: NewMockclient(ctrl),
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

func TestOrderGateway_GetOrderByID(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC)
	validOrder := &proto.Order{
		Id:        "order-123",
		Status:    "created",
		CreatedAt: timestamppb.New(fixedTime),
	}

	tests := []struct {
		name           string
		orderID        string
		mockSetup      func(m *mock)
		prepareContext func(context.Context) context.Context
		resultChecker  func(t *testing.T, result *entities.Order)
		errorAssertion require.ErrorAssertionFunc
	}{
		{
			name:    "Успешное получение заказа по ID",
			orderID: "order-123",
			mockSetup: func(m *mock) {
				m.Mockclient.EXPECT().
					GetOrderById(gomock.Any(), gomock.Any()).
					Return(&proto.GetOrderByIdResponse{Order: validOrder}, nil)
			},
			resultChecker: func(t *testing.T, result *entities.Order) {
				require.NotNil(t, result)
				assert.Equal(t, "order-123", result.ID)
				assert.Equal(t, entities.OrderCreated, result.Status)
			},
			errorAssertion: require.NoError,
		},
		{
			name:    "Успешное получение после retry при временной недоступности",
			orderID: "order-456",
			mockSetup: func(m *mock) {
				unavailableErr := status.Error(codes.Unavailable, "service unavailable")
				gomock.InOrder(
					m.Mockclient.EXPECT().
						GetOrderById(gomock.Any(), gomock.Any()).
						Return(nil, unavailableErr),
					m.Mockclient.EXPECT().
						GetOrderById(gomock.Any(), gomock.Any()).
						Return(nil, unavailableErr),
					m.Mockclient.EXPECT().
						GetOrderById(gomock.Any(), gomock.Any()).
						Return(&proto.GetOrderByIdResponse{Order: validOrder}, nil),
				)
			},
			resultChecker: func(t *testing.T, result *entities.Order) {
				require.NotNil(t, result)
				assert.Equal(t, "order-123", result.ID)
				assert.Equal(t, entities.OrderCreated, result.Status)
			},
			errorAssertion: require.NoError,
		},
		{
			name:    "Отсутствие retry при NotFound (permanent error)",
			orderID: "nonexistent-order",
			mockSetup: func(m *mock) {
				notFoundErr := status.Error(codes.NotFound, "order not found")
				m.Mockclient.EXPECT().
					GetOrderById(gomock.Any(), gomock.Any()).
					Return(nil, notFoundErr).
					Times(1)
			},
			resultChecker: func(t *testing.T, result *entities.Order) {
				assert.Nil(t, result)
			},
			errorAssertion: errorAssertion(nil, "get order"),
		},
		{
			name:    "Отсутствие retry при InvalidArgument (permanent error)",
			orderID: "invalid-id",
			mockSetup: func(m *mock) {
				invalidErr := status.Error(codes.InvalidArgument, "invalid order id format")
				m.Mockclient.EXPECT().
					GetOrderById(gomock.Any(), gomock.Any()).
					Return(nil, invalidErr).
					Times(1)
			},
			resultChecker: func(t *testing.T, result *entities.Order) {
				assert.Nil(t, result)
			},
			errorAssertion: errorAssertion(nil, "get order"),
		},
		{
			name:    "Retry при ResourceExhausted (rate limit)",
			orderID: "order-789",
			mockSetup: func(m *mock) {
				rateLimitErr := status.Error(codes.ResourceExhausted, "rate limit exceeded")
				gomock.InOrder(
					m.Mockclient.EXPECT().
						GetOrderById(gomock.Any(), gomock.Any()).
						Return(nil, rateLimitErr),
					m.Mockclient.EXPECT().
						GetOrderById(gomock.Any(), gomock.Any()).
						Return(&proto.GetOrderByIdResponse{Order: validOrder}, nil),
				)
			},
			resultChecker: func(t *testing.T, result *entities.Order) {
				require.NotNil(t, result)
				assert.Equal(t, "order-123", result.ID)
			},
			errorAssertion: require.NoError,
		},
		{
			name:    "Retry при DeadlineExceeded",
			orderID: "order-999",
			mockSetup: func(m *mock) {
				timeoutErr := status.Error(codes.DeadlineExceeded, "deadline exceeded")
				gomock.InOrder(
					m.Mockclient.EXPECT().
						GetOrderById(gomock.Any(), gomock.Any()).
						Return(nil, timeoutErr),
					m.Mockclient.EXPECT().
						GetOrderById(gomock.Any(), gomock.Any()).
						Return(&proto.GetOrderByIdResponse{Order: validOrder}, nil),
				)
			},
			resultChecker: func(t *testing.T, result *entities.Order) {
				require.NotNil(t, result)
				assert.Equal(t, "order-123", result.ID)
			},
			errorAssertion: require.NoError,
		},
		{
			name:    "Обработка пустого ответа от сервиса",
			orderID: "order-empty",
			mockSetup: func(m *mock) {
				m.Mockclient.EXPECT().
					GetOrderById(gomock.Any(), gomock.Any()).
					Return(&proto.GetOrderByIdResponse{Order: nil}, nil)
			},
			resultChecker: func(t *testing.T, result *entities.Order) {
				assert.Nil(t, result)
			},
			errorAssertion: errorAssertion(nil, "order not found"),
		},
		{
			name:    "Превышение лимита retry попыток",
			orderID: "order-retry-limit",
			mockSetup: func(m *mock) {
				unavailableErr := status.Error(codes.Unavailable, "service unavailable")
				m.Mockclient.EXPECT().
					GetOrderById(gomock.Any(), gomock.Any()).
					Return(nil, unavailableErr).
					MinTimes(2).
					MaxTimes(10)
			},
			resultChecker: func(t *testing.T, result *entities.Order) {
				assert.Nil(t, result)
			},
			errorAssertion: errorAssertion(nil, "get order"),
		},
		{
			name:    "Отмена контекста во время выполнения запроса",
			orderID: "order-cancelled",
			prepareContext: func(ctx context.Context) context.Context {
				ctx, cancel := context.WithCancel(ctx)
				cancel()
				return ctx
			},
			mockSetup: func(m *mock) {
				m.Mockclient.EXPECT().
					GetOrderById(gomock.Any(), gomock.Any()).
					Return(nil, context.Canceled).
					AnyTimes()
			},
			resultChecker: func(t *testing.T, result *entities.Order) {
				assert.Nil(t, result)
			},
			errorAssertion: errorAssertion(nil, "get order"),
		},
		{
			name:    "Превышение deadline контекста",
			orderID: "order-deadline",
			prepareContext: func(ctx context.Context) context.Context {
				ctx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
				_ = cancel
				return ctx
			},
			mockSetup: func(m *mock) {
				m.Mockclient.EXPECT().
					GetOrderById(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ *proto.GetOrderByIdRequest, _ ...any) (*proto.GetOrderByIdResponse, error) {
						select {
						case <-time.After(50 * time.Millisecond):
							return &proto.GetOrderByIdResponse{Order: validOrder}, nil
						case <-ctx.Done():
							return nil, ctx.Err()
						}
					}).
					AnyTimes()
			},
			resultChecker: func(t *testing.T, result *entities.Order) {
				assert.Nil(t, result)
			},
			errorAssertion: errorAssertion(nil, "get order"),
		},
		{
			name:    "Обработка Internal Server Error от gRPC сервиса",
			orderID: "order-internal-error",
			mockSetup: func(m *mock) {
				internalErr := status.Error(codes.Internal, "internal server error")
				m.Mockclient.EXPECT().
					GetOrderById(gomock.Any(), gomock.Any()).
					Return(nil, internalErr).
					Times(1)
			},
			resultChecker: func(t *testing.T, result *entities.Order) {
				assert.Nil(t, result)
			},
			errorAssertion: errorAssertion(nil, "get order"),
		},
		{
			name:    "Обработка Unknown Error (не gRPC ошибка)",
			orderID: "order-unknown",
			mockSetup: func(m *mock) {
				unknownErr := errors.New("network connection failed")
				m.Mockclient.EXPECT().
					GetOrderById(gomock.Any(), gomock.Any()).
					Return(nil, unknownErr).
					Times(1)
			},
			resultChecker: func(t *testing.T, result *entities.Order) {
				assert.Nil(t, result)
			},
			errorAssertion: errorAssertion(nil, "get order"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := newMock(ctrl)

			ctx := context.Background()
			if tt.prepareContext != nil {
				ctx = tt.prepareContext(ctx)
			}

			if tt.mockSetup != nil {
				tt.mockSetup(m)
			}

			gateway := order.New(m.Mockclient)
			result, err := gateway.GetOrderByID(ctx, tt.orderID)

			tt.resultChecker(t, result)
			tt.errorAssertion(t, err, tt.name)
		})
	}
}

func TestOrderGateway_GetOrdersAfter(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC)
	validOrders := []*proto.Order{
		{
			Id:        "order-1",
			Status:    "created",
			CreatedAt: timestamppb.New(fixedTime),
		},
		{
			Id:        "order-2",
			Status:    "accepted",
			CreatedAt: timestamppb.New(fixedTime.Add(1 * time.Hour)),
		},
	}

	tests := []struct {
		name           string
		after          time.Time
		mockSetup      func(m *mock)
		prepareContext func(context.Context) context.Context
		resultChecker  func(t *testing.T, result []entities.Order)
		errorAssertion require.ErrorAssertionFunc
	}{
		{
			name:  "Успешное получение заказов после указанного времени",
			after: fixedTime,
			mockSetup: func(m *mock) {
				m.Mockclient.EXPECT().
					GetOrders(gomock.Any(), gomock.Any()).
					Return(&proto.GetOrdersResponse{Orders: validOrders}, nil)
			},
			resultChecker: func(t *testing.T, result []entities.Order) {
				require.NotNil(t, result)
				assert.Len(t, result, 2)
				assert.Equal(t, "order-1", result[0].ID)
				assert.Equal(t, "order-2", result[1].ID)
			},
			errorAssertion: require.NoError,
		},
		{
			name:  "Возврат пустого списка когда заказы отсутствуют",
			after: fixedTime,
			mockSetup: func(m *mock) {
				m.Mockclient.EXPECT().
					GetOrders(gomock.Any(), gomock.Any()).
					Return(&proto.GetOrdersResponse{Orders: []*proto.Order{}}, nil)
			},
			resultChecker: func(t *testing.T, result []entities.Order) {
				require.NotNil(t, result)
				assert.Len(t, result, 0)
			},
			errorAssertion: require.NoError,
		},
		{
			name:  "Retry при Unavailable с последующим успехом",
			after: fixedTime,
			mockSetup: func(m *mock) {
				unavailableErr := status.Error(codes.Unavailable, "service unavailable")
				gomock.InOrder(
					m.Mockclient.EXPECT().
						GetOrders(gomock.Any(), gomock.Any()).
						Return(nil, unavailableErr),
					m.Mockclient.EXPECT().
						GetOrders(gomock.Any(), gomock.Any()).
						Return(&proto.GetOrdersResponse{Orders: validOrders}, nil),
				)
			},
			resultChecker: func(t *testing.T, result []entities.Order) {
				require.NotNil(t, result)
				assert.Len(t, result, 2)
			},
			errorAssertion: require.NoError,
		},
		{
			name:  "Обработка ошибки базы данных от сервиса",
			after: fixedTime,
			mockSetup: func(m *mock) {
				internalErr := status.Error(codes.Internal, "database error")
				m.Mockclient.EXPECT().
					GetOrders(gomock.Any(), gomock.Any()).
					Return(nil, internalErr).
					Times(1)
			},
			resultChecker: func(t *testing.T, result []entities.Order) {
				assert.Nil(t, result)
			},
			errorAssertion: errorAssertion(nil, "get orders"),
		},
		{
			name:  "Отмена контекста во время запроса списка заказов",
			after: fixedTime,
			prepareContext: func(ctx context.Context) context.Context {
				ctx, cancel := context.WithCancel(ctx)
				cancel()
				return ctx
			},
			mockSetup: func(m *mock) {
				m.Mockclient.EXPECT().
					GetOrders(gomock.Any(), gomock.Any()).
					Return(nil, context.Canceled).
					AnyTimes()
			},
			resultChecker: func(t *testing.T, result []entities.Order) {
				assert.Nil(t, result)
			},
			errorAssertion: errorAssertion(nil, "get orders"),
		},
		{
			name:  "Retry при ResourceExhausted",
			after: fixedTime,
			mockSetup: func(m *mock) {
				rateLimitErr := status.Error(codes.ResourceExhausted, "rate limit")
				gomock.InOrder(
					m.Mockclient.EXPECT().
						GetOrders(gomock.Any(), gomock.Any()).
						Return(nil, rateLimitErr),
					m.Mockclient.EXPECT().
						GetOrders(gomock.Any(), gomock.Any()).
						Return(nil, rateLimitErr),
					m.Mockclient.EXPECT().
						GetOrders(gomock.Any(), gomock.Any()).
						Return(&proto.GetOrdersResponse{Orders: validOrders}, nil),
				)
			},
			resultChecker: func(t *testing.T, result []entities.Order) {
				require.NotNil(t, result)
				assert.Len(t, result, 2)
			},
			errorAssertion: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := newMock(ctrl)

			ctx := context.Background()
			if tt.prepareContext != nil {
				ctx = tt.prepareContext(ctx)
			}

			if tt.mockSetup != nil {
				tt.mockSetup(m)
			}

			gateway := order.New(m.Mockclient)
			result, err := gateway.GetOrdersAfter(ctx, tt.after)

			tt.resultChecker(t, result)
			tt.errorAssertion(t, err, tt.name)
		})
	}
}

func TestOrderGateway_RetryBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		errorCode        codes.Code
		expectRetry      bool
		minAttempts      int
		maxAttempts      int
		maxExecutionTime time.Duration
	}{
		{
			name:             "ResourceExhausted должен ретраиться",
			errorCode:        codes.ResourceExhausted,
			expectRetry:      true,
			minAttempts:      2,
			maxAttempts:      10,
			maxExecutionTime: 2 * time.Second,
		},
		{
			name:             "Unavailable должен ретраиться",
			errorCode:        codes.Unavailable,
			expectRetry:      true,
			minAttempts:      2,
			maxAttempts:      10,
			maxExecutionTime: 2 * time.Second,
		},
		{
			name:             "DeadlineExceeded должен ретраиться",
			errorCode:        codes.DeadlineExceeded,
			expectRetry:      true,
			minAttempts:      2,
			maxAttempts:      10,
			maxExecutionTime: 2 * time.Second,
		},
		{
			name:             "NotFound НЕ должен ретраиться",
			errorCode:        codes.NotFound,
			expectRetry:      false,
			minAttempts:      1,
			maxAttempts:      1,
			maxExecutionTime: 500 * time.Millisecond,
		},
		{
			name:             "InvalidArgument НЕ должен ретраиться",
			errorCode:        codes.InvalidArgument,
			expectRetry:      false,
			minAttempts:      1,
			maxAttempts:      1,
			maxExecutionTime: 500 * time.Millisecond,
		},
		{
			name:             "Internal НЕ должен ретраиться",
			errorCode:        codes.Internal,
			expectRetry:      false,
			minAttempts:      1,
			maxAttempts:      1,
			maxExecutionTime: 500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := newMock(ctrl)

			testErr := status.Error(tt.errorCode, tt.name)
			attemptCount := 0

			m.Mockclient.EXPECT().
				GetOrderById(gomock.Any(), gomock.Any()).
				DoAndReturn(func(context.Context, *proto.GetOrderByIdRequest, ...any) (*proto.GetOrderByIdResponse, error) {
					attemptCount++
					return nil, testErr
				}).
				MinTimes(tt.minAttempts).
				MaxTimes(tt.maxAttempts)

			gateway := order.New(m.Mockclient)

			start := time.Now()
			_, err := gateway.GetOrderByID(context.Background(), "test-order")
			elapsed := time.Since(start)

			assert.Error(t, err)
			assert.GreaterOrEqual(t, attemptCount, tt.minAttempts, "Expected at least %d attempts, got %d", tt.minAttempts, attemptCount)
			assert.LessOrEqual(t, attemptCount, tt.maxAttempts, "Expected at most %d attempts, got %d", tt.maxAttempts, attemptCount)
			assert.LessOrEqual(t, elapsed, tt.maxExecutionTime, "Execution took %v, expected max %v", elapsed, tt.maxExecutionTime)

			t.Logf("Retry behavior for %s: %d attempts in %v", tt.errorCode, attemptCount, elapsed)
		})
	}
}
