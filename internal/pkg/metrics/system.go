package metrics

import (
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

var (
	SystemCPUUsage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "system_cpu_usage_percent",
			Help: "CPU usage percentage",
		},
	)

	SystemMemoryUsage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "system_memory_usage_bytes",
			Help: "System memory usage in bytes",
		},
	)

	ApplicationMemoryUsage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "application_memory_usage_bytes",
			Help: "Application memory usage in bytes (Go heap allocation)",
		},
	)
)

func StartSystemMetricsCollector() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			collectSystemMetrics()
		}
	}()
}

func collectSystemMetrics() {
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		SystemCPUUsage.Set(cpuPercent[0])
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	vmStat, err := mem.VirtualMemory()
	if err == nil {
		SystemMemoryUsage.Set(float64(vmStat.Used))
	}

	ApplicationMemoryUsage.Set(float64(m.Alloc))
}
