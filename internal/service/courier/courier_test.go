package courier_test

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
	"service/internal/service/courier"
)

type mock struct {
	*MockRepository
	*MockTxManager
}

func newMock(ctrl *gomock.Controller) *mock {
	return &mock{
		MockRepository: NewMockRepository(ctrl),
		MockTxManager:  NewMockTxManager(ctrl),
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

func TestCourierService_CreateCourier(t *testing.T) {
	t.Parallel()

	validModify := entities.CourierModify{
		Name:          pointer.To("John Wick"),
		Phone:         pointer.To("+79161234567"),
		Status:        pointer.To(entities.CourierAvailable),
		TransportType: pointer.To(entities.Car),
	}

	tests := []struct {
		name       string
		modify     entities.CourierModify
		mockSetup  func(m *mock)
		expectedID int64
		assertion  require.ErrorAssertionFunc
	}{
		{
			name:   "Успешная регистрация нового курьера",
			modify: validModify,
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					Create(gomock.Any(), validModify).
					Return(int64(1), nil)
			},
			expectedID: 1,
			assertion:  require.NoError,
		},
		{
			name:       "Отклонение создания курьера без обязательных полей",
			modify:     entities.CourierModify{},
			expectedID: 0,
			assertion:  errorAssertion(courier.ErrMissingRequiredFields, ""),
		},
		{
			name: "Отклонение создания курьера с пустым именем",
			modify: entities.CourierModify{
				Name:          pointer.To(""),
				Phone:         pointer.To("+79161234567"),
				Status:        pointer.To(entities.CourierAvailable),
				TransportType: pointer.To(entities.Car),
			},
			expectedID: 0,
			assertion:  errorAssertion(courier.ErrInvalidName, ""),
		},
		{
			name: "Отклонение создания курьера с именем только из пробелов",
			modify: entities.CourierModify{
				Name:          pointer.To("   "),
				Phone:         pointer.To("+79161234567"),
				Status:        pointer.To(entities.CourierAvailable),
				TransportType: pointer.To(entities.Car),
			},
			expectedID: 0,
			assertion:  errorAssertion(courier.ErrInvalidName, ""),
		},
		{
			name: "Отклонение создания курьера с номером телефона без кода страны",
			modify: entities.CourierModify{
				Name:          pointer.To("Test"),
				Phone:         pointer.To("79161234567"),
				Status:        pointer.To(entities.CourierAvailable),
				TransportType: pointer.To(entities.Car),
			},
			expectedID: 0,
			assertion:  errorAssertion(courier.ErrInvalidPhone, ""),
		},
		{
			name: "Отклонение создания курьера с номером телефона содержащим буквы",
			modify: entities.CourierModify{
				Name:          pointer.To("Test"),
				Phone:         pointer.To("+7abc1234567"),
				Status:        pointer.To(entities.CourierAvailable),
				TransportType: pointer.To(entities.Car),
			},
			expectedID: 0,
			assertion:  errorAssertion(courier.ErrInvalidPhone, ""),
		},
		{
			name: "Отклонение создания курьера с номером телефона содержащим спецсимволы",
			modify: entities.CourierModify{
				Name:          pointer.To("Test"),
				Phone:         pointer.To("+7916-123-45-67"),
				Status:        pointer.To(entities.CourierAvailable),
				TransportType: pointer.To(entities.Car),
			},
			expectedID: 0,
			assertion:  errorAssertion(courier.ErrInvalidPhone, ""),
		},
		{
			name: "Отклонение создания курьера с пустым номером телефона",
			modify: entities.CourierModify{
				Name:          pointer.To("Test"),
				Phone:         pointer.To(""),
				Status:        pointer.To(entities.CourierAvailable),
				TransportType: pointer.To(entities.Car),
			},
			expectedID: 0,
			assertion:  errorAssertion(courier.ErrInvalidPhone, ""),
		},
		{
			name: "Отклонение создания курьера с невалидным статусом",
			modify: entities.CourierModify{
				Name:          pointer.To("Test"),
				Phone:         pointer.To("+79161234567"),
				Status:        pointer.To(entities.CourierStatusType("offline")),
				TransportType: pointer.To(entities.Car),
			},
			expectedID: 0,
			assertion:  errorAssertion(courier.ErrInvalidStatus, ""),
		},
		{
			name: "Отклонение создания курьера с невалидным типом транспорта",
			modify: entities.CourierModify{
				Name:          pointer.To("Test"),
				Phone:         pointer.To("+79161234567"),
				Status:        pointer.To(entities.CourierAvailable),
				TransportType: pointer.To(entities.CourierTransportType("helicopter")),
			},
			expectedID: 0,
			assertion:  errorAssertion(courier.ErrInvalidTransport, ""),
		},
		{
			name:   "Обработка ошибок репозитория при создании",
			modify: validModify,
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					Create(gomock.Any(), validModify).
					Return(int64(0), errors.New("repository error"))
			},
			expectedID: 0,
			assertion:  errorAssertion(nil, "create courier"),
		},
		{
			name:   "Обработка конфликта дублирования курьера",
			modify: validModify,
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					Create(gomock.Any(), validModify).
					Return(int64(0), courier.ErrConflict)
			},
			expectedID: 0,
			assertion:  errorAssertion(nil, "create courier"),
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

			service := courier.New(m.MockRepository, m.MockTxManager)
			id, err := service.CreateCourier(context.Background(), tt.modify)

			assert.Equal(t, tt.expectedID, id)
			tt.assertion(t, err)
		})
	}
}

func TestCourierService_UpdateCourier(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	existingCourier := &entities.Courier{
		ID:            1,
		Name:          "Snake Plissken",
		Phone:         "+79031112233",
		Status:        entities.CourierAvailable,
		TransportType: entities.Car,
		CreatedAt:     fixedTime,
		UpdatedAt:     fixedTime,
	}

	tests := []struct {
		name           string
		modify         entities.CourierModify
		mockSetup      func(m *mock)
		expectedResult *entities.Courier
		assertion      require.ErrorAssertionFunc
	}{
		{
			name: "Успешное обновление имени курьера",
			modify: entities.CourierModify{
				ID:   pointer.To(int64(1)),
				Name: pointer.To("John McClane"),
			},
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(existingCourier, nil)
			},
			expectedResult: existingCourier,
			assertion:      require.NoError,
		},
		{
			name: "Успешное обновление номера телефона курьера",
			modify: entities.CourierModify{
				ID:    pointer.To(int64(1)),
				Phone: pointer.To("+79264445566"),
			},
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(existingCourier, nil)
			},
			expectedResult: existingCourier,
			assertion:      require.NoError,
		},
		{
			name: "Успешное обновление статуса курьера на 'занят'",
			modify: entities.CourierModify{
				ID:     pointer.To(int64(1)),
				Status: pointer.To(entities.CourierBusy),
			},
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(existingCourier, nil)
			},
			expectedResult: existingCourier,
			assertion:      require.NoError,
		},
		{
			name: "Успешное обновление типа транспорта курьера на 'самокат'",
			modify: entities.CourierModify{
				ID:            pointer.To(int64(1)),
				TransportType: pointer.To(entities.Scooter),
			},
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(existingCourier, nil)
			},
			expectedResult: existingCourier,
			assertion:      require.NoError,
		},
		{
			name: "Отклонение обновления без полей для изменения",
			modify: entities.CourierModify{
				ID: pointer.To(int64(1)),
			},
			expectedResult: nil,
			assertion:      errorAssertion(courier.ErrMissingRequiredFields, ""),
		},
		{
			name: "Отклонение обновления с пустым именем",
			modify: entities.CourierModify{
				ID:   pointer.To(int64(1)),
				Name: pointer.To(""),
			},
			expectedResult: nil,
			assertion:      errorAssertion(courier.ErrInvalidName, ""),
		},
		{
			name: "Отклонение обновления с именем только из пробелов",
			modify: entities.CourierModify{
				ID:   pointer.To(int64(1)),
				Name: pointer.To("   "),
			},
			expectedResult: nil,
			assertion:      errorAssertion(courier.ErrInvalidName, ""),
		},
		{
			name: "Отклонение обновления с номером телефона без кода страны",
			modify: entities.CourierModify{
				ID:    pointer.To(int64(1)),
				Phone: pointer.To("79264445566"),
			},
			expectedResult: nil,
			assertion:      errorAssertion(courier.ErrInvalidPhone, ""),
		},
		{
			name: "Отклонение обновления с номером телефона содержащим буквы",
			modify: entities.CourierModify{
				ID:    pointer.To(int64(1)),
				Phone: pointer.To("+7abc9999999"),
			},
			expectedResult: nil,
			assertion:      errorAssertion(courier.ErrInvalidPhone, ""),
		},
		{
			name: "Отклонение обновления с пустым номером телефона",
			modify: entities.CourierModify{
				ID:    pointer.To(int64(1)),
				Phone: pointer.To(""),
			},
			expectedResult: nil,
			assertion:      errorAssertion(courier.ErrInvalidPhone, ""),
		},
		{
			name: "Отклонение обновления с невалидным статусом",
			modify: entities.CourierModify{
				ID:     pointer.To(int64(1)),
				Status: pointer.To(entities.CourierStatusType("inactive")),
			},
			expectedResult: nil,
			assertion:      errorAssertion(courier.ErrInvalidStatus, ""),
		},
		{
			name: "Отклонение обновления с невалидным типом транспорта",
			modify: entities.CourierModify{
				ID:            pointer.To(int64(1)),
				TransportType: pointer.To(entities.CourierTransportType("bicycle")),
			},
			expectedResult: nil,
			assertion:      errorAssertion(courier.ErrInvalidTransport, ""),
		},
		{
			name: "Обработка ошибки базы данных при обновлении",
			modify: entities.CourierModify{
				ID:   pointer.To(int64(1)),
				Name: pointer.To("Ellen Ripley"),
			},
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database constraint violation"))
			},
			expectedResult: nil,
			assertion:      errorAssertion(nil, "failed to update courier: database constraint violation"),
		},
		{
			name: "Обработка попытки обновления несуществующего курьера",
			modify: entities.CourierModify{
				ID:   pointer.To(int64(999)),
				Name: pointer.To("Solid Snake"),
			},
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil, courier.ErrCourierNotFound)
			},
			expectedResult: nil,
			assertion:      errorAssertion(nil, "failed to update courier"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := newMock(ctrl)
			service := courier.New(m.MockRepository, m.MockTxManager)

			if tt.mockSetup != nil {
				tt.mockSetup(m)
			}

			result, err := service.UpdateCourier(context.Background(), tt.modify)

			assert.Equal(t, tt.expectedResult, result)
			tt.assertion(t, err)
		})
	}
}

func TestCourierService_GetCourier(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	existingCourier := &entities.Courier{
		ID:            1,
		Name:          "Snake Plissken",
		Phone:         "+79031112233",
		Status:        entities.CourierAvailable,
		TransportType: entities.Car,
		CreatedAt:     fixedTime,
		UpdatedAt:     fixedTime,
	}

	tests := []struct {
		name           string
		id             int64
		mockSetup      func(m *mock)
		expectedResult *entities.Courier
		assertion      require.ErrorAssertionFunc
	}{
		{
			name: "Успешное получение деталей курьера",
			id:   1,
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					GetByID(gomock.Any(), int64(1)).
					Return(existingCourier, nil)
			},
			expectedResult: existingCourier,
			assertion:      require.NoError,
		},
		{
			name: "Курьер не найден в системе",
			id:   999,
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					GetByID(gomock.Any(), int64(999)).
					Return(nil, courier.ErrCourierNotFound)
			},
			expectedResult: nil,
			assertion:      errorAssertion(nil, "failed to get courier"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			m := newMock(ctrl)
			service := courier.New(m.MockRepository, m.MockTxManager)

			if tt.mockSetup != nil {
				tt.mockSetup(m)
			}

			result, err := service.GetCourier(context.Background(), tt.id)

			assert.Equal(t, tt.expectedResult, result)
			tt.assertion(t, err)
		})
	}
}

func TestCourierService_GetCouriers(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	couriers := []entities.Courier{
		{
			ID:            1,
			Name:          "Barry Lyndon",
			Phone:         "+79161234567",
			Status:        entities.CourierAvailable,
			TransportType: entities.Car,
			CreatedAt:     fixedTime,
			UpdatedAt:     fixedTime,
		},
		{
			ID:            2,
			Name:          "Xian Ni",
			Phone:         "+79265554433",
			Status:        entities.CourierBusy,
			TransportType: entities.Scooter,
			CreatedAt:     fixedTime,
			UpdatedAt:     fixedTime,
		},
	}

	tests := []struct {
		name           string
		mockSetup      func(m *mock)
		expectedResult []entities.Courier
		assertion      require.ErrorAssertionFunc
	}{
		{
			name: "Успешное получение всех курьеров",
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					GetAll(gomock.Any()).
					Return(couriers, nil)
			},
			expectedResult: couriers,
			assertion:      require.NoError,
		},
		{
			name: "Покрытие обработки ошибок базы данных",
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					GetAll(gomock.Any()).
					Return(nil, errors.New("query execution failed"))
			},
			expectedResult: nil,
			assertion:      errorAssertion(nil, "failed to get couriers: query execution failed"),
		},
		{
			name: "Возврат пустого списка когда курьеры отсутствуют",
			mockSetup: func(m *mock) {
				m.MockRepository.EXPECT().
					GetAll(gomock.Any()).
					Return([]entities.Courier{}, nil)
			},
			expectedResult: []entities.Courier{},
			assertion:      require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			m := newMock(ctrl)
			service := courier.New(m.MockRepository, m.MockTxManager)

			if tt.mockSetup != nil {
				tt.mockSetup(m)
			}

			result, err := service.GetCouriers(context.Background())

			assert.Equal(t, tt.expectedResult, result)
			tt.assertion(t, err)
		})
	}
}

func TestCourierService_ContextCancellation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		prepareContext func(context.Context) context.Context
		mockSetup      func(ctx context.Context, m *mock)
		assertion      require.ErrorAssertionFunc
	}{
		{
			name: "Отмена контекста во время операции",
			prepareContext: func(ctx context.Context) context.Context {
				ctx, cancel := context.WithCancel(ctx)
				cancel()
				return ctx
			},
			mockSetup: func(ctx context.Context, m *mock) {
				m.MockRepository.EXPECT().
					GetByID(ctx, int64(1)).
					Return(nil, context.Canceled)
			},
			assertion: errorAssertion(context.Canceled, ""),
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
				tt.mockSetup(ctx, m)
			}

			service := courier.New(m.MockRepository, m.MockTxManager)
			result, err := service.GetCourier(ctx, 1)

			assert.Nil(t, result)
			tt.assertion(t, err)
		})
	}
}
