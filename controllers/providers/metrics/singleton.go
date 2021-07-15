package metrics

import (
	"sync"

	"github.com/AbsaOSS/k8gb/controllers/depresolver"
)

var (
	o       sync.Once
	metrics Metrics
)

// Prometheus public static metrics, providing instance of initialised metrics
func Prometheus() Metrics {
	return metrics
}

// Init always initialise PrometheusMetrics. The inititalisation happens only once
func Init(c *depresolver.Config) {
	o.Do(func() {
		metrics = newPrometheusMetrics(*c)
	})
}
