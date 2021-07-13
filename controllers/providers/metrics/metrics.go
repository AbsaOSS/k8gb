package metrics

import k8gbv1beta1 "github.com/AbsaOSS/k8gb/api/v1beta1"

type Metrics interface {
	// Register prometheus metrics. Read register documentation, but shortly:
	// You can register metric with given name only once
	Register() (err error)

	// Unregister prometheus metrics. Should be called as deferred function after successful Register
	Unregister()

	UpdateIngressHostsPerStatusMetric(gslb *k8gbv1beta1.Gslb, serviceHealth map[string]string) error

	UpdateHealthyRecordsMetric(gslb *k8gbv1beta1.Gslb, healthyRecords map[string][]string) error
}
