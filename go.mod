module github.com/AbsaOSS/k8gb

go 1.15

require (
	github.com/AbsaOSS/gopkg v0.1.2
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.4.0
	github.com/golang/mock v1.5.0
	github.com/gruntwork-io/terratest v0.34.8
	github.com/infobloxopen/infoblox-go-client v1.1.0
	github.com/lixiangzhong/dnsutil v0.0.0-20191203032812-75ad39d2945a
	github.com/miekg/dns v1.1.42
	github.com/onsi/ginkgo v1.14.2 // indirect
	github.com/prometheus/client_golang v1.10.0
	github.com/rs/zerolog v1.21.0
	github.com/stretchr/testify v1.7.0
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.20.6
	k8s.io/apiextensions-apiserver v0.20.2 // indirect
	k8s.io/apimachinery v0.20.6
	k8s.io/client-go v0.20.6
	sigs.k8s.io/controller-runtime v0.7.2
	sigs.k8s.io/external-dns v0.8.0
)
