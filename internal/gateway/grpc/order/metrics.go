package order

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	GatewayRetriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_retries_total",
			Help: "Total number of gateway retry attempts",
		},
		[]string{"service", "method", "reason"},
	)

	GatewayRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_request_duration_seconds",
			Help:    "Duration of gateway requests including retries",
			Buckets: []float64{.01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"service", "method", "grpc_code"},
	)
)
