package rate_limiter

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var RateLimitExceededTotal = promauto.NewCounterVec(

	prometheus.CounterOpts{
		Name: "rate_limit_exceeded_total",
		Help: "Total number of requests rejected due to rate limiting",
	},
	[]string{"method", "route"},
)
