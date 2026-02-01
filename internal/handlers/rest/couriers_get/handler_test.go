package couriers_get_test

import (
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
	"service/internal/handlers/rest/couriers_get"
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

func TestCouriersGetHandler(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		mockSetup      func(m *mock)
		expectedStatus int
		expectedBody   []map[string]interface{}
		wantErr        bool
	}{
		{
			name: "Успешное получение списка курьеров",
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					GetCouriers(gomock.Any()).
					Return([]entities.Courier{
						{
							ID:            1,
							Name:          "Snake Plissken",
							Phone:         "79999991111",
							Status:        entities.CourierAvailable,
							TransportType: entities.Car,
							CreatedAt:     fixedTime,
							UpdatedAt:     fixedTime,
						},
						{
							ID:            2,
							Name:          "Renegade Immortal",
							Phone:         "79999992222",
							Status:        entities.CourierBusy,
							TransportType: entities.Scooter,
							CreatedAt:     fixedTime,
							UpdatedAt:     fixedTime,
						},
						{
							ID:            3,
							Name:          "Khan Li",
							Phone:         "79999993333",
							Status:        entities.CourierPaused,
							TransportType: entities.OnFoot,
							CreatedAt:     fixedTime,
							UpdatedAt:     fixedTime,
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: []map[string]interface{}{
				{
					"ID":             float64(1),
					"name":           "Snake Plissken",
					"phone":          "79999991111",
					"status":         "available",
					"transport_type": "car",
				},
				{
					"ID":             float64(2),
					"name":           "Renegade Immortal",
					"phone":          "79999992222",
					"status":         "busy",
					"transport_type": "scooter",
				},
				{
					"ID":             float64(3),
					"name":           "Khan Li",
					"phone":          "79999993333",
					"status":         "paused",
					"transport_type": "on_foot",
				},
			},
			wantErr: false,
		},
		{
			name: "Успешное получение одного курьера",
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					GetCouriers(gomock.Any()).
					Return([]entities.Courier{
						{
							ID:            1,
							Name:          "Snake Plissken",
							Phone:         "79999991111",
							Status:        entities.CourierAvailable,
							TransportType: entities.Car,
							CreatedAt:     fixedTime,
							UpdatedAt:     fixedTime,
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: []map[string]interface{}{
				{
					"ID":             float64(1),
					"name":           "Snake Plissken",
					"phone":          "79999991111",
					"status":         "available",
					"transport_type": "car",
				},
			},
			wantErr: false,
		},
		{
			name: "Успешное получение пустого списка курьеров",
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					GetCouriers(gomock.Any()).
					Return([]entities.Courier{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   []map[string]interface{}{},
			wantErr:        false,
		},
		{
			name: "Ошибка сервиса при получении курьеров",
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					GetCouriers(gomock.Any()).
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

			handler := couriers_get.New(m.MockhandlerLogger, m.MockService)
			req := httptest.NewRequest(http.MethodGet, "/couriers", http.NoBody)
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
