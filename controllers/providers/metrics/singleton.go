package metrics

import (
	"sync"

	"github.com/AbsaOSS/k8gb/controllers/depresolver"
	"github.com/AbsaOSS/k8gb/controllers/logging"
)

var (
	o       sync.Once
	metrics Metrics
)

var log = logging.Logger()

// Prometheus public static metrics, providing instance of initialised metrics
func Prometheus() Metrics {
	if metrics == nil {
		log.Fatal().Msg("metrics was not initialised")
	}
	return metrics
}

// Init always initialise PrometheusMetrics. The inititalisation happens only once
func Init(c *depresolver.Config) {
	o.Do(func() {
		metrics = NewPrometheusMetrics(*c)
	})
}
