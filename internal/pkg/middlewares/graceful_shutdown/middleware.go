package graceful_shutdown

import (
	"context"
	"net/http"
	"sync/atomic"
)

func Middleware(isShuttingDown *atomic.Bool, ongoingCtx context.Context) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-ongoingCtx.Done():
				if isShuttingDown.Load() {
					http.Error(w, "Service is shutting down", http.StatusServiceUnavailable)
					return
				}
			default:
			}
			next.ServeHTTP(w, r)
		})
	}
}
