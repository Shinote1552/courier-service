package rate_limiter

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"service/pkg/logger"
)

// rateLimiterQPS - в будущем будет отдельный конфиг для rate limiter,
// пока принмаем поле от конфига сервера простым int
func Middleware(log handlerLogger, rateLimiterQPS int, rlimiter Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rlimiter.Allow() {
				handlerPath := r.URL.Path
				route := mux.CurrentRoute(r)
				if route != nil {
					if template, err := route.GetPathTemplate(); err == nil {
						handlerPath = template
					}
				}

				log.With(
					logger.NewField("method", r.Method),
					logger.NewField("path", r.URL.Path),
					logger.NewField("route", handlerPath),
					logger.NewField("remote_addr", r.RemoteAddr),
				).Warn("rate limit exceeded")

				RateLimitExceededTotal.WithLabelValues(r.Method, handlerPath).Inc()

				log.With(
					logger.NewField("limit exceeded", r.URL.Path),
				).Info("rate limiter")
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rateLimiterQPS))
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)

				_, err := w.Write([]byte(`{"error":"Too Many Requests","message":"Rate limit exceeded. Try again later."}`))
				if err != nil {
					log.With(
						logger.NewField("error", err),
						logger.NewField("path", r.URL.Path),
					).Error("failed to write rate limit response")
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
