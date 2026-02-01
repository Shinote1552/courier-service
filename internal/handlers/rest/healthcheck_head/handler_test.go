package healthcheck_head_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"service/internal/handlers/rest/healthcheck_head"
)

type mock struct {
	isShuttingDown atomic.Bool
}

func newMock() *mock {
	return &mock{}
}

func (m *mock) SetShuttingDown(value bool) {
	m.isShuttingDown.Store(value)
}

func TestHealthcheckHeadHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		isShuttingDown bool
		expectedStatus int
	}{
		{
			name:           "Сервис работает, возвращает 204",
			isShuttingDown: false,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Сервис останавливается, возвращает 503",
			isShuttingDown: true,
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := newMock()
			m.SetShuttingDown(tt.isShuttingDown)

			handler := healthcheck_head.New(&m.isShuttingDown)
			req := httptest.NewRequest(http.MethodHead, "/healthcheck", http.NoBody)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "unexpected status code")
		})
	}
}
