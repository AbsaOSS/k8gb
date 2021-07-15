/*
Copyright 2021 The k8gb Contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Generated by GoLic, for more details see: https://github.com/AbsaOSS/golic
*/
package dns

import (
	"fmt"
	"github.com/AbsaOSS/k8gb/controllers/providers/metrics"

	"github.com/AbsaOSS/k8gb/controllers/depresolver"
	"github.com/AbsaOSS/k8gb/controllers/providers/assistant"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ProviderFactory struct {
	config depresolver.Config
	client client.Client
	metrics metrics.Metrics
}

func NewDNSProviderFactory(client client.Client, config depresolver.Config, metrics metrics.Metrics) (f *ProviderFactory, err error) {
	if client == nil {
		err = fmt.Errorf("nil client")
	}
	// TODO: test this
	if metrics == nil {
		err = fmt.Errorf("nil metrics")
	}
	f = &ProviderFactory{
		config: config,
		client: client,
		metrics: metrics,
	}
	return
}

func (f *ProviderFactory) Provider() Provider {
	a := assistant.NewGslbAssistant(f.client, f.config.K8gbNamespace, f.config.EdgeDNSServer, f.config.EdgeDNSServerPort)
	switch f.config.EdgeDNSType {
	case depresolver.DNSTypeNS1:
		return NewExternalDNS(externalDNSTypeNS1, f.config, a)
	case depresolver.DNSTypeRoute53:
		return NewExternalDNS(externalDNSTypeRoute53, f.config, a)
	case depresolver.DNSTypeInfoblox:
		return NewInfobloxDNS(f.config, a)
	}
	return NewEmptyDNS(f.config, a)
}
