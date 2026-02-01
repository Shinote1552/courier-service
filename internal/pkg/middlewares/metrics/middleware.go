package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"service/pkg/logger"
)

func Middleware(log handlerLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			statusCode := strconv.Itoa(rw.statusCode)

			// Пробуем взять из mux-роут
			handlerPath := r.URL.Path
			route := mux.CurrentRoute(r)
			if route != nil {
				if template, err := route.GetPathTemplate(); err == nil {
					handlerPath = template
				}
			}

			// Метрики Prometheus
			HTTPRequestDuration.WithLabelValues(r.Method, handlerPath, statusCode).Observe(duration.Seconds())
			HTTPRequestTotal.WithLabelValues(r.Method, handlerPath, statusCode).Inc()

			log.With(
				logger.NewField("timestamp", time.Now().Format("2006/01/02 15:04:05")),
				logger.NewField("method", r.Method),
				logger.NewField("path", r.URL.Path),
				logger.NewField("route", handlerPath),
				logger.NewField("status", statusCode),
				logger.NewField("duration", duration.String()),
			).Info("HTTP request")
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
