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
package test

import (
	"fmt"
	"k8gbterratest/utils"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Basic k8gb deployment test that is verifying that associated ingress is getting created
// Relies on two local clusters deployed by `$make deploy-two-local-clusters`
func TestK8gbBasicRoundRobinExample(t *testing.T) {
	t.Parallel()
	var host = "roundrobin-test." + settings.DNSZone
	const gslbName = "roundrobin-test-gslb"

	// Path to the Kubernetes resource config we will test
	kubeResourcePath, err := filepath.Abs("../examples/roundrobin2.yaml")
	require.NoError(t, err)

	// To ensure we can reuse the resource config on the same cluster to test different scenarios, we setup a unique
	// namespace for the resources for this test.
	// Note that namespaces must be lowercase.
	namespaceName := fmt.Sprintf("k8gb-test-roundrobin-%s", strings.ToLower(random.UniqueId()))

	// Here we choose to use the defaults, which is:
	// - HOME/.kube/config for the kubectl config file
	// - Current context of the kubectl config file
	// - Random namespace
	optionsContext1 := k8s.NewKubectlOptions(settings.Cluster1, "", namespaceName)
	optionsContext2 := k8s.NewKubectlOptions(settings.Cluster2, "", namespaceName)

	k8s.CreateNamespace(t, optionsContext1, namespaceName)
	k8s.CreateNamespace(t, optionsContext2, namespaceName)
	defer k8s.DeleteNamespace(t, optionsContext1, namespaceName)
	defer k8s.DeleteNamespace(t, optionsContext2, namespaceName)

	utils.CreateGslbWithHealthyApp(t, optionsContext1, settings, kubeResourcePath, gslbName, host)

	utils.CreateGslbWithHealthyApp(t, optionsContext2, settings, kubeResourcePath, gslbName, host)

	ingressIPs1 := utils.GetIngressIPs(t, optionsContext1, gslbName)
	ingressIPs2 := utils.GetIngressIPs(t, optionsContext2, gslbName)
	var expectedIPs []string
	expectedIPs = append(expectedIPs, ingressIPs2...)
	expectedIPs = append(expectedIPs, ingressIPs1...)

	sort.Strings(expectedIPs)

	t.Run("round-robin on two concurrent clusters with podinfo running", func(t *testing.T) {
		resolvedIPsCoreDNS1, err := utils.WaitForLocalGSLB(t, settings.DNSServer1, settings.Port1, host, expectedIPs)
		require.NoError(t, err)
		resolvedIPsCoreDNS2, err := utils.WaitForLocalGSLB(t, settings.DNSServer2, settings.Port2, host, expectedIPs)
		require.NoError(t, err)

		assert.NotEmpty(t, resolvedIPsCoreDNS1)
		assert.NotEmpty(t, resolvedIPsCoreDNS2)
		assert.Equal(t, len(resolvedIPsCoreDNS1), len(expectedIPs))
		assert.Equal(t, len(resolvedIPsCoreDNS2), len(expectedIPs))
		assert.ElementsMatch(t, resolvedIPsCoreDNS1, expectedIPs, "%s:%s", host, settings.Port1)
		assert.ElementsMatch(t, resolvedIPsCoreDNS2, expectedIPs, "%s:%s", host, settings.Port2)
	})

	t.Run("kill podinfo on the first cluster", func(t *testing.T) {
		// kill app in the first cluster
		k8s.RunKubectl(t, optionsContext1, "scale", "deploy", "frontend-podinfo", "--replicas=0")

		utils.AssertGslbStatus(t, optionsContext1, gslbName, host+":Unhealthy")

		resolvedIPsCoreDNS1, err := utils.WaitForLocalGSLB(t, settings.DNSServer1, settings.Port1, host, ingressIPs2)
		require.NoError(t, err)
		resolvedIPsCoreDNS2, err := utils.WaitForLocalGSLB(t, settings.DNSServer2, settings.Port2, host, ingressIPs2)
		require.NoError(t, err)
		assert.ElementsMatch(t, resolvedIPsCoreDNS1, resolvedIPsCoreDNS2)
	})

	t.Run("kill podinfo on the second cluster", func(t *testing.T) {
		// kill app in the second cluster
		k8s.RunKubectl(t, optionsContext2, "scale", "deploy", "frontend-podinfo", "--replicas=0")

		utils.AssertGslbStatus(t, optionsContext2, gslbName, host+":Unhealthy")

		_, err = utils.WaitForLocalGSLB(t, settings.DNSServer1, settings.Port1, host, []string{""})
		require.NoError(t, err)
		_, err = utils.WaitForLocalGSLB(t, settings.DNSServer2, settings.Port2, host, []string{""})
		require.NoError(t, err)
	})

	t.Run("start podinfo on the both clusters", func(t *testing.T) {
		// start app in the both clusters
		k8s.RunKubectl(t, optionsContext1, "scale", "deploy", "frontend-podinfo", "--replicas=1")
		k8s.RunKubectl(t, optionsContext2, "scale", "deploy", "frontend-podinfo", "--replicas=1")

		utils.AssertGslbStatus(t, optionsContext1, gslbName, host+":Healthy")
		utils.AssertGslbStatus(t, optionsContext2, gslbName, host+":Healthy")

		resolvedIPsCoreDNS1, err := utils.WaitForLocalGSLB(t, settings.DNSServer1, settings.Port1, host, expectedIPs)
		require.NoError(t, err)
		resolvedIPsCoreDNS2, err := utils.WaitForLocalGSLB(t, settings.DNSServer2, settings.Port2, host, expectedIPs)
		require.NoError(t, err)

		assert.NotEmpty(t, resolvedIPsCoreDNS1)
		assert.NotEmpty(t, resolvedIPsCoreDNS2)
		assert.Equal(t, len(resolvedIPsCoreDNS1), len(expectedIPs))
		assert.Equal(t, len(resolvedIPsCoreDNS2), len(expectedIPs))
		assert.ElementsMatch(t, resolvedIPsCoreDNS1, expectedIPs, "%s:%s", host, settings.Port1)
		assert.ElementsMatch(t, resolvedIPsCoreDNS2, expectedIPs, "%s:%s", host, settings.Port2)
	})
}
