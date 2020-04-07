package test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/gruntwork-io/terratest/modules/shell"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Basic ohmyglb deployment test that is verifying that associated ingress is getting created
// Relies on two local clusters deployed by `$make deploy-two-local-clusters`
func TestOhmyglbBasicFailoverExample(t *testing.T) {
	t.Parallel()

	// Path to the Kubernetes resource config we will test
	kubeResourcePath, err := filepath.Abs("../examples/failover.yaml")
	require.NoError(t, err)

	// To ensure we can reuse the resource config on the same cluster to test different scenarios, we setup a unique
	// namespace for the resources for this test.
	// Note that namespaces must be lowercase.
	namespaceName := fmt.Sprintf("ohmyglb-test-%s", strings.ToLower(random.UniqueId()))

	// Here we choose to use the defaults, which is:
	// - HOME/.kube/config for the kubectl config file
	// - Current context of the kubectl config file
	// - Random namespace
	optionsContext1 := k8s.NewKubectlOptions("kind-test-gslb1", "", namespaceName)
	optionsContext2 := k8s.NewKubectlOptions("kind-test-gslb2", "", namespaceName)

	k8s.CreateNamespace(t, optionsContext1, namespaceName)
	k8s.CreateNamespace(t, optionsContext2, namespaceName)
	defer k8s.DeleteNamespace(t, optionsContext1, namespaceName)
	defer k8s.DeleteNamespace(t, optionsContext2, namespaceName)

	gslbName := "test-gslb"

	createGslbWithHealthyApp(t, optionsContext1, kubeResourcePath, gslbName)

	createGslbWithHealthyApp(t, optionsContext2, kubeResourcePath, gslbName)

	expectedIPs := GetIngressIPs(t, optionsContext1, gslbName)

	beforeFailoverResponse, err := DoWithRetryWaitingForValueE(
		t,
		"Wait coredns to pickup dns values...",
		300,
		1*time.Second,
		func() ([]string, error) { return Dig(t, "localhost", 53, "terratest-failover.cloud.example.com") },
		expectedIPs)
	require.NoError(t, err)

	assert.Equal(t, beforeFailoverResponse, expectedIPs)

	k8s.RunKubectl(t, optionsContext1, "scale", "deploy", "frontend-podinfo", "--replicas=0")

	assertGslbStatus(t, optionsContext1, gslbName, "terratest-failover.cloud.example.com:Unhealthy")

	t.Run("failover happens as expected", func(t *testing.T) {
		expectedIPsAfterFailover := GetIngressIPs(t, optionsContext2, gslbName)

		afterFailoverResponse, err := DoWithRetryWaitingForValueE(
			t,
			"Wait for failover to happen and coredns to pickup new values...",
			300,
			1*time.Second,
			func() ([]string, error) { return Dig(t, "localhost", 53, "terratest-failover.cloud.example.com") },
			expectedIPsAfterFailover)
		require.NoError(t, err)

		assert.Equal(t, afterFailoverResponse, expectedIPsAfterFailover)
	})

}

func GetIngressIPs(t *testing.T, options *k8s.KubectlOptions, ingressName string) []string {
	var ingressIPs []string
	ingress := k8s.GetIngress(t, options, ingressName)
	for _, ip := range ingress.Status.LoadBalancer.Ingress {
		ingressIPs = append(ingressIPs, ip.IP)
	}
	return ingressIPs
}

func Dig(t *testing.T, dnsServer string, dnsPort int, dnsName string) ([]string, error) {
	port := fmt.Sprintf("-p%v", dnsPort)
	dnsServer = fmt.Sprintf("@%s", dnsServer)

	digApp := shell.Command{
		Command: "dig",
		Args:    []string{port, dnsServer, dnsName, "+short"},
	}

	digAppOut := shell.RunCommandAndGetOutput(t, digApp)
	digAppSlice := strings.Split(digAppOut, "\n")

	sort.Strings(digAppSlice)

	return digAppSlice, nil
}

// Concept is borrowed from terratest/modules/retry and extended to our use case
func DoWithRetryWaitingForValueE(t *testing.T, actionDescription string, maxRetries int, sleepBetweenRetries time.Duration, action func() ([]string, error), expectedResult []string) ([]string, error) {
	var output []string
	var err error

	for i := 0; i <= maxRetries; i++ {

		output, err = action()
		if err != nil {
			return output, nil
			t.Logf("%s returned an error: %s. Sleeping for %s and will try again.", actionDescription, err.Error(), sleepBetweenRetries)
		}

		if reflect.DeepEqual(output, expectedResult) {
			return output, err
		}

		t.Logf("%s does not match expected result. Expected:(%s). Actual:(%s). Sleeping for %s and will try again.", actionDescription, expectedResult, output, sleepBetweenRetries)
		time.Sleep(sleepBetweenRetries)
	}

	return output, retry.MaxRetriesExceeded{Description: actionDescription, MaxRetries: maxRetries}
}

func createGslbWithHealthyApp(t *testing.T, options *k8s.KubectlOptions, kubeResourcePath string, gslbName string) {
	k8s.KubectlApply(t, options, kubeResourcePath)

	k8s.WaitUntilIngressAvailable(t, options, gslbName, 60, 1*time.Second)
	ingress := k8s.GetIngress(t, options, gslbName)
	require.Equal(t, ingress.Name, gslbName)

	helmRepoAdd := shell.Command{
		Command: "helm",
		Args:    []string{"repo", "add", "podinfo", "https://stefanprodan.github.io/podinfo"},
	}

	helmRepoUpdate := shell.Command{
		Command: "helm",
		Args:    []string{"repo", "update"},
	}
	shell.RunCommand(t, helmRepoAdd)
	shell.RunCommand(t, helmRepoUpdate)
	helmOptions := helm.Options{
		KubectlOptions: options,
	}
	helm.Install(t, &helmOptions, "podinfo/podinfo", "frontend")

	testAppFilter := metav1.ListOptions{
		LabelSelector: "app=frontend-podinfo",
	}

	k8s.WaitUntilNumPodsCreated(t, options, testAppFilter, 1, 60, 1*time.Second)

	var testAppPods []corev1.Pod

	testAppPods = k8s.ListPods(t, options, testAppFilter)

	for _, pod := range testAppPods {
		k8s.WaitUntilPodAvailable(t, options, pod.Name, 60, 1*time.Second)
	}

	k8s.WaitUntilServiceAvailable(t, options, "frontend-podinfo", 60, 1*time.Second)

	assertGslbStatus(t, options, gslbName, "terratest-failover.cloud.example.com:Healthy")
}

func assertGslbStatus(t *testing.T, options *k8s.KubectlOptions, gslbName string, serviceStatus string) {
	// Totally not ideal, but we need to wait until Gslb figures out Healthy status
	// We can optimize it by waiting loop with threshold later
	time.Sleep(10 * time.Second)

	ohmyglbServiceHealth, err := k8s.RunKubectlAndGetOutputE(t, options, "get", "gslb", gslbName, "-o", "jsonpath='{.status.serviceHealth}'")
	if err != nil {
		t.Errorf("Failed to get ohmyglb status with kubectl (%s)", err)
	}

	want := fmt.Sprintf("'map[%s]'", serviceStatus)
	assert.Equal(t, ohmyglbServiceHealth, want)
}
