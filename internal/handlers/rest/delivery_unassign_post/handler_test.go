package delivery_unassign_post_test

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
	"service/internal/entities"
	"service/internal/handlers/rest/delivery_unassign_post"
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

func TestDeliveryUnassignPostHandler(t *testing.T) {
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
			name: "Успешное снятие курьера с доставки",
			requestBody: `{
				"order_ID": "order-2026-001"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					DeliveryUnassign(gomock.Any(), "order-2026-001").
					Return(&entities.DeliveryUnassignment{
						CourierID: 1,
						OrderID:   "order-2026-001",
						Status:    "available",
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"courier_ID": float64(1),
				"order_ID":   "order-2026-001",
				"status":     "available",
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
					DeliveryUnassign(gomock.Any(), "").
					Return(nil, delivery.ErrInvalidOrderID)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Назначение не найдено",
			requestBody: `{
				"order_ID": "order-2026-001"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					DeliveryUnassign(gomock.Any(), "order-2026-001").
					Return(nil, delivery.ErrDeliveryNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   nil,
			wantErr:        true,
		},
		{
			name: "Ошибка сервиса при снятии доставки",
			requestBody: `{
				"order_ID": "order-2026-001"
			}`,
			mockSetup: func(m *mock) {
				m.MockService.EXPECT().
					DeliveryUnassign(gomock.Any(), "order-2026-001").
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

			handler := delivery_unassign_post.New(m.MockhandlerLogger, m.MockService)

			req := httptest.NewRequest(http.MethodPost, "/delivery/unassign",
				bytes.NewReader([]byte(tt.requestBody)))
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
