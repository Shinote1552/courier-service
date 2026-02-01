package courier_get_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"service/internal/entities"
	"service/internal/handlers/rest/courier_get"
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

func TestCourierGetHandler(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		courierID      string
		mockSetup      func(m *mock)
		expectedStatus int
		expectedBody   map[string]interface{}
		wantErr        bool
	}{
		{
			name:      "Успешное получение курьера по ID",
			courierID: "1",
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					GetCourier(gomock.Any(), int64(1)).
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
			name:      "Успешное получение курьера со статусом busy",
			courierID: "2",
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					GetCourier(gomock.Any(), int64(2)).
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
			name:           "Невалидный ID курьера (не число)",
			courierID:      "abc",
			mockSetup:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name:      "Курьер не найден",
			courierID: "999",
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					GetCourier(gomock.Any(), int64(999)).
					Return(nil, courier.ErrCourierNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name:      "Невалидный ID курьера (отрицательное число)",
			courierID: "-1",
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					GetCourier(gomock.Any(), int64(-1)).
					Return(nil, courier.ErrInvalidCourierID)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name:      "Ошибка сервиса при получении курьера",
			courierID: "1",
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					GetCourier(gomock.Any(), int64(1)).
					Return(nil, errors.New("database connection error"))
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

			handler := courier_get.New(m.MockhandlerLogger, m.MockService)

			req := httptest.NewRequest(http.MethodGet, "/courier/"+tt.courierID, http.NoBody)
			req = mux.SetURLVars(req, map[string]string{"id": tt.courierID})
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
