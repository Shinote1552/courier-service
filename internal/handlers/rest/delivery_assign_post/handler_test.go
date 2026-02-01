package delivery_assign_post_test

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
	"service/internal/handlers/rest/delivery_assign_post"
	"service/internal/service/delivery"
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

func TestDeliveryAssignPostHandler(t *testing.T) {
	t.Parallel()

	assignedAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	deadline := assignedAt.Add(30 * time.Minute)
	deadlineStr := deadline.Format(time.RFC3339)

	tests := []struct {
		name           string
		requestBody    string
		mockSetup      func(m *mock)
		expectedStatus int
		expectedBody   map[string]interface{}
		wantErr        bool
	}{
		{
			name: "Успешное назначение курьера на доставку",
			requestBody: `{
				"order_ID": "order-2026-001"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					DeliveryAssign(gomock.Any(), "order-2026-001").
					Return(&entities.DeliveryAssignment{
						CourierID:     1,
						OrderID:       "order-2026-001",
						AssignedAt:    assignedAt,
						Deadline:      deadline,
						TransportType: entities.Car,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"courier_ID":        float64(1),
				"order_ID":          "order-2026-001",
				"transport_type":    "car",
				"delivery_deadline": deadlineStr,
			},
			wantErr: false,
		},
		{
			name: "Успешное назначение курьера на скутере",
			requestBody: `{
				"order_ID": "order-2026-002"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					DeliveryAssign(gomock.Any(), "order-2026-002").
					Return(&entities.DeliveryAssignment{
						CourierID:     2,
						OrderID:       "order-2026-002",
						AssignedAt:    assignedAt,
						Deadline:      deadline,
						TransportType: entities.Scooter,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"courier_ID":        float64(2),
				"order_ID":          "order-2026-002",
				"transport_type":    "scooter",
				"delivery_deadline": deadlineStr,
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
			name: "Невалидный ID заказа (пустая строка)",
			requestBody: `{
				"order_ID": ""
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					DeliveryAssign(gomock.Any(), "").
					Return(nil, delivery.ErrInvalidOrderID)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Нет доступных курьеров",
			requestBody: `{
				"order_ID": "order-2026-001"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					DeliveryAssign(gomock.Any(), "order-2026-001").
					Return(nil, delivery.ErrNoAvailableCouriers)
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Заказ уже назначен",
			requestBody: `{
				"order_ID": "order-2026-001"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					DeliveryAssign(gomock.Any(), "order-2026-001").
					Return(nil, delivery.ErrOrderAlreadyAssigned)
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name:        "Отсутствуют обязательные поля",
			requestBody: `{}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					DeliveryAssign(gomock.Any(), "").
					Return(nil, delivery.ErrMissingRequiredFields)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Ошибка сервиса при назначении доставки",
			requestBody: `{
				"order_ID": "order-2026-001"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					DeliveryAssign(gomock.Any(), "order-2026-001").
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

			handler := delivery_assign_post.New(m.MockhandlerLogger, m.MockService)

			req := httptest.NewRequest(http.MethodPost, "/delivery/assign", bytes.NewReader([]byte(tt.requestBody)))
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
