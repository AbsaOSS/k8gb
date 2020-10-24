# K8GB - Kubernetes Global Balancer

## Project Health

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://github.com/AbsaOSS/k8gb/workflows/build/badge.svg)](https://github.com/AbsaOSS/k8gb/actions?query=workflow%3A%22Golang+lint+and+test%22)
[![Gosec](https://github.com/AbsaOSS/k8gb/workflows/Gosec/badge.svg)](https://github.com/AbsaOSS/k8gb/actions?query=workflow%3AGosec)
[![Terratest Status](https://github.com/AbsaOSS/k8gb/workflows/Terratest/badge.svg)](https://github.com/AbsaOSS/k8gb/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/AbsaOSS/k8gb)](https://goreportcard.com/report/github.com/AbsaOSS/k8gb)
[![Helm Publish](https://github.com/AbsaOSS/k8gb/workflows/Helm%20Publish/badge.svg)](https://github.com/AbsaOSS/k8gb/actions?query=workflow%3A%22Helm+Publish%22)
[![Docker Pulls](https://img.shields.io/docker/pulls/absaoss/k8gb)](https://hub.docker.com/r/absaoss/k8gb)

A Global Service Load Balancing solution with a focus on having cloud native qualities and work natively in a Kubernetes context.

<< ascicinema / gif >>

Global load balancing, commonly referred to as GSLB (Global Server Load Balancing) solutions, have typically been the domain of proprietary network software and hardware vendors and installed and managed by siloed network teams.

k8gb is a completely open source, cloud native, global load balancing solution for Kubernetes.

k8gb focuses on load balancing traffic across geographically dispersed Kubernetes clusters using multiple load balancing strategies to meet requirements such as region failover for high availability.

Global load balancing for any Kubernetes Service can now be enabled and managed by any operations or development teams in the same Kubernetes native way as any other custom resource.

## Key Differentiators

* Load balancing is based on timeproof DNS protocol which is perfect for global scope and extremely reliable
* No dedicated management cluster and no single point of failure
* Kubernetes native application health checks utilizing status of Liveness and Readiness probes for load balancing decisions
* Configuration with a single Kubernetes CRD of Gslb kind

## Motivation and Architecture

k8gb was born out of need for a open source, cloud native GSLB solution at Absa bank in South Africa.

As part of the bank's wider container adoption running multiple, geographically dispersed Kubernetes clusters, the need for a global load balancer that was driven from the health of Kubernetes Services was required and for which there did not seem to be an existing solution.

Yes, there are proprietary network software and hardware vendors with GSLB solutions and products, however, these were costly, heavy weight in terms of complexity and adoption and in most cases were not Kubernetes native, requiring dedicated hardware or software to be run outside of Kubernetes.

This was the problem we set out to solve with k8gb.

Born as a completely open source project and following the popular Kubernetes operator pattern, k8gb can be installed in a Kubernetes cluster and via a Gslb custom resource, can provide independent GSLB capability to any Ingress or Service in the cluster, without the need for handoffs and coordination between dedicated network teams.

k8gb commoditises GSLB for Kubernetes, putting teams in complete control of exposing Services across geographically dispersed Kubernetes clusters across public and private clouds.

k8gb requires no specialised software or hardware, relying completely on other OSS/CNCF projects, has no single point of failure and fits in with any existing Kubernetes deployment workflow (e.g. GitOps, Kustomize, Helm, etc.) or tools.

Please see the extended acrhitecture documentation [here](/docs/index.md)

## Installation and Configuration Tutorials

* [General deployment with Infoblox integration](/docs/deploy_infoblox.md)
* [AWS based deployment with Route53 integration](/docs/deploy_route53.md)
* [Local playground for testing and development](/docs/local.md)
* [Metrics](/docs/metrics.md)

## Production Readiness

k8gb is very well tested with the following environment options

| Type                            | Implementation                                     |
|---------------------------------|----------------------------------------------------|
| Kubernetes Version              | >= 1.14 (with install workaround) >= 1.15 (Stable) |
| Ingress Controller              | Nginx                                              |
| EdgeDNS                         | Infoblox                                           |
| Number of k8gb enabled clusters | 2                                                  |

## Contributing

TODO: Create proper Contributing.md
