package timeout

import (
	"context"
	"net/http"
	"time"
)

func Middleware(timout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// r.Context() = ongoingCtx (из BaseContext)
			ctx, cancel := context.WithTimeout(r.Context(), timout)
			defer cancel()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
