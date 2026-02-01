package courier_post_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"service/internal/handlers/rest/courier_post"
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

func TestCourierPostHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		requestBody    string
		mockSetup      func(m *mock)
		expectedStatus int
		expectedBody   map[string]interface{}
		wantErr        bool
	}{
		{
			name: "Успешное создание курьера",
			requestBody: `{
				"name": "Snake Plissken",
				"phone": "79999991111",
				"status": "available",
				"transport_type": "car"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					CreateCourier(gomock.Any(), gomock.Any()).
					Return(int64(1), nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"ID": float64(1),
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
			name: "Невалидное имя курьера (пустая строка)",
			requestBody: `{
				"name": "",
				"phone": "79999991111",
				"status": "available",
				"transport_type": "car"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					CreateCourier(gomock.Any(), gomock.Any()).
					Return(int64(0), courier.ErrInvalidName)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Невалидный телефон курьера",
			requestBody: `{
				"name": "Snake Plissken",
				"phone": "123",
				"status": "available",
				"transport_type": "car"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					CreateCourier(gomock.Any(), gomock.Any()).
					Return(int64(0), courier.ErrInvalidPhone)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Невалидный статус курьера",
			requestBody: `{
				"name": "Snake Plissken",
				"phone": "79999991111",
				"status": "invalid_status",
				"transport_type": "car"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					CreateCourier(gomock.Any(), gomock.Any()).
					Return(int64(0), courier.ErrInvalidStatus)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Невалидный тип транспорта",
			requestBody: `{
				"name": "Snake Plissken",
				"phone": "79999991111",
				"status": "available",
				"transport_type": "invalid_transport"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					CreateCourier(gomock.Any(), gomock.Any()).
					Return(int64(0), courier.ErrInvalidTransport)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Отсутствуют обязательные поля",
			requestBody: `{
				"name": "Snake Plissken",
				"status": "available",
				"transport_type": "car"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					CreateCourier(gomock.Any(), gomock.Any()).
					Return(int64(0), courier.ErrMissingRequiredFields)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Конфликт - курьер с таким телефоном уже существует",
			requestBody: `{
				"name": "Snake Plissken",
				"phone": "79999991111",
				"status": "available",
				"transport_type": "car"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					CreateCourier(gomock.Any(), gomock.Any()).
					Return(int64(0), courier.ErrConflict)
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Ошибка сервиса при создании курьера",
			requestBody: `{
				"name": "Snake Plissken",
				"phone": "79999991111",
				"status": "available",
				"transport_type": "car"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					CreateCourier(gomock.Any(), gomock.Any()).
					Return(int64(0), errors.New("database connection error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   nil,
			wantErr:        true,
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

			handler := courier_post.New(m.MockhandlerLogger, m.MockService)

			req := httptest.NewRequest(http.MethodPost, "/courier", bytes.NewReader([]byte(tt.requestBody)))
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
