package metrics

import (
	"github.com/AbsaOSS/k8gb/controllers/depresolver"
	"sync"
)

var (
	once    sync.Once
	metrics Metrics
)

// Prometheus public static metrics, providing instance of initialised metrics
func Prometheus() Metrics {
	return metrics
}

// Init always initialise PrometheusMetrics. The inititalisation happens only once
func Init(c *depresolver.Config) {
	once.Do(func() {
		metrics = NewPrometheusMetrics(*c)
	})
}
