package ping_get_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"service/internal/handlers/rest/ping_get"
)

func TestPingGetHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:           "Успешный запрос возвращает pong",
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"message": "pong",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockLog := NewMockhandlerLogger(ctrl)

			mockLog.EXPECT().
				With(gomock.Any()).
				Return(mockLog).
				AnyTimes()

			handler := ping_get.New(mockLog)
			req := httptest.NewRequest(http.MethodGet, "/ping", http.NoBody)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "unexpected status code")

			if tt.expectedBody != nil {
				expectedJSON, err := json.Marshal(tt.expectedBody)
				require.NoError(t, err, "failed to marshal expected body")
				assert.JSONEq(t, string(expectedJSON), w.Body.String(), "unexpected response body")
			}
		})
	}
}
