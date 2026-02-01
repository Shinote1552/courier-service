package courier_put_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"service/internal/entities"
	"service/internal/handlers/rest/courier_put"
	"service/internal/service/courier"
)

type mock struct {
	*MockService
	*MockhandlerLogger
}

func newMock(ctrl *gomock.Controller) *mock {
	return &mock{
		MockService:       NewMockService(ctrl),
		MockhandlerLogger: NewMockhandlerLogger(ctrl),
	}
}

func TestCourierPutHandler(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		requestBody    string
		mockSetup      func(m *mock)
		expectedStatus int
		expectedBody   map[string]interface{}
		wantErr        bool
	}{
		{
			name: "Успешное обновление имени курьера",
			requestBody: `{
				"ID": 1,
				"name": "Snake Plissken"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(&entities.Courier{
						ID:            1,
						Name:          "Snake Plissken",
						Phone:         "79999991111",
						Status:        entities.CourierAvailable,
						TransportType: entities.Car,
						CreatedAt:     fixedTime,
						UpdatedAt:     fixedTime,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"ID":             float64(1),
				"name":           "Snake Plissken",
				"phone":          "79999991111",
				"status":         "available",
				"transport_type": "car",
			},
			wantErr: false,
		},
		{
			name: "Успешное обновление всех полей курьера",
			requestBody: `{
				"ID": 2,
				"name": "Renegade Immortal",
				"phone": "79999992222",
				"status": "busy",
				"transport_type": "scooter"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(&entities.Courier{
						ID:            2,
						Name:          "Renegade Immortal",
						Phone:         "79999992222",
						Status:        entities.CourierBusy,
						TransportType: entities.Scooter,
						CreatedAt:     fixedTime,
						UpdatedAt:     fixedTime,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"ID":             float64(2),
				"name":           "Renegade Immortal",
				"phone":          "79999992222",
				"status":         "busy",
				"transport_type": "scooter",
			},
			wantErr: false,
		},
		{
			name: "Успешное обновление только телефона",
			requestBody: `{
				"ID": 3,
				"phone": "79999993333"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(&entities.Courier{
						ID:            3,
						Name:          "Khan Li",
						Phone:         "79999993333",
						Status:        entities.CourierAvailable,
						TransportType: entities.OnFoot,
						CreatedAt:     fixedTime,
						UpdatedAt:     fixedTime,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"ID":             float64(3),
				"name":           "Khan Li",
				"phone":          "79999993333",
				"status":         "available",
				"transport_type": "on_foot",
			},
			wantErr: false,
		},
		{
			name:           "Невалидный JSON в теле запроса",
			requestBody:    "invalid json",
			mockSetup:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Отсутствует ID курьера",
			requestBody: `{
				"name": "Snake Plissken"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(nil, courier.ErrInvalidCourierID)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Курьер не найден",
			requestBody: `{
				"ID": 999,
				"name": "Snake Plissken"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(nil, courier.ErrCourierNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Невалидное имя курьера (пустая строка)",
			requestBody: `{
				"ID": 1,
				"name": ""
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(nil, courier.ErrInvalidName)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Невалидный телефон курьера",
			requestBody: `{
				"ID": 1,
				"phone": "123"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(nil, courier.ErrInvalidPhone)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Невалидный статус курьера",
			requestBody: `{
				"ID": 1,
				"status": "invalid_status"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(nil, courier.ErrInvalidStatus)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Невалидный тип транспорта",
			requestBody: `{
				"ID": 1,
				"transport_type": "invalid_transport"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(nil, courier.ErrInvalidTransport)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Конфликт - телефон уже используется другим курьером",
			requestBody: `{
				"ID": 1,
				"phone": "79999991111"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(nil, courier.ErrConflict)
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Нет полей для обновления",
			requestBody: `{
				"ID": 1
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(nil, courier.ErrMissingRequiredFields)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Ошибка сервиса при обновлении курьера",
			requestBody: `{
				"ID": 1,
				"name": "Snake Plissken"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database connection error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Успешное обновление статуса на paused",
			requestBody: `{
				"ID": 4,
				"status": "paused"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(&entities.Courier{
						ID:            4,
						Name:          "Mega Driver",
						Phone:         "79999994444",
						Status:        entities.CourierPaused,
						TransportType: entities.Scooter,
						CreatedAt:     fixedTime,
						UpdatedAt:     fixedTime,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"ID":             float64(4),
				"name":           "Mega Driver",
				"phone":          "79999994444",
				"status":         "paused",
				"transport_type": "scooter",
			},
			wantErr: false,
		},
		{
			name: "Успешное обновление типа транспорта на on_foot",
			requestBody: `{
				"ID": 5,
				"transport_type": "on_foot"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					UpdateCourier(gomock.Any(), gomock.Any()).
					Return(&entities.Courier{
						ID:            5,
						Name:          "Walker",
						Phone:         "79999995555",
						Status:        entities.CourierAvailable,
						TransportType: entities.OnFoot,
						CreatedAt:     fixedTime,
						UpdatedAt:     fixedTime,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"ID":             float64(5),
				"name":           "Walker",
				"phone":          "79999995555",
				"status":         "available",
				"transport_type": "on_foot",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			m := newMock(ctrl)

			m.MockhandlerLogger.EXPECT().
				With(gomock.Any()).
				Return(m.MockhandlerLogger).
				AnyTimes()

			if tt.mockSetup != nil {
				tt.mockSetup(m)
			}

			handler := courier_put.New(m.MockhandlerLogger, m.MockService)

			req := httptest.NewRequest(http.MethodPut, "/courier", bytes.NewReader([]byte(tt.requestBody)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "unexpected status code")

			if tt.wantErr {
				return
			}

			if tt.expectedBody != nil {
				expectedJSON, err := json.Marshal(tt.expectedBody)
				require.NoError(t, err, "failed to marshal expected body")
				assert.JSONEq(t, string(expectedJSON), w.Body.String(), "unexpected response body")
			}
		})
	}
}
