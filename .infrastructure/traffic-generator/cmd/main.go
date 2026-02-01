package main

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Метрики
var (
	opsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "myapp_operations_total",
		Help: "Общее количество операций",
	})

	opsDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "myapp_operation_duration_seconds",
		Help:    "Длительность операции в секундах",
		Buckets: []float64{0.1, 0.3, 0.5, 1, 2},
	})
)

func simulateOperation() {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		opsDuration.Observe(duration)
	}()

	time.Sleep(time.Duration(100+rand.Intn(1900)) * time.Millisecond)
	opsCounter.Inc()
}

func main() {
	rand.Seed(time.Now().UnixNano())

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":2112", nil)

	for {
		simulateOperation()
		time.Sleep(5 * time.Second)
	}
}
