package test

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func TestGslbCreated(t *testing.T) {
	t.Parallel()
	const host = "roundrobin-test.cloud.example.com"
	const gslbPath = "../examples/roundrobin2.yaml"

	instance1, err := NewWorkflow(t, "k3d-test-gslb1", 5053).
		WithGslb(gslbPath, host).
		WithTestApp().
		Start()
	require.NoError(t, err)
	defer instance1.Kill()
	instance2, err := NewWorkflow(t, "k3d-test-gslb2", 5054).
		WithGslb(gslbPath, host).
		WithTestApp().
		Start()
	require.NoError(t, err)
	defer instance2.Kill()


	ingressIP1 := instance1.GetIngressIPs()
	ingressIP2 := instance2.GetIngressIPs()
	expectedIPs := append(ingressIP1,ingressIP2...)

	//t.Run("round-robin on two concurrent clusters with podinfo running", func(t *testing.T) {
	ip1, err := instance1.WaitForGSLB(instance2)
	require.NoError(t, err)
	ip2, err := instance2.WaitForGSLB(instance1)
	require.NoError(t, err)
	assert.ElementsMatch(t, expectedIPs, ip1)
	assert.ElementsMatch(t, expectedIPs, ip2)
	//})

	//t.Run("kill podinfo on the first cluster", func(t *testing.T) {
	instance1.StopTestApp()
	ip1, err = instance1.WaitForGSLB(instance2)
	require.NoError(t, err)
	ip2, err = instance2.WaitForGSLB(instance1)
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(ingressIP2, ip1))
	require.True(t,reflect.DeepEqual(ingressIP2, ip2))
	//})

	//t.Run("kill podinfo on the second cluster", func(t *testing.T) {
	instance2.StopTestApp()
	ip2,err = instance2.WaitForGSLB( instance1)
	require.NoError(t, err)
	ip1,err = instance1.WaitForGSLB( instance2)
	require.NoError(t, err)
	assert.ElementsMatch(t, ip1, ip2)
	//})

	//t.Run("start podinfo on the both clusters", func(t *testing.T) {
	// start app in the both clusters
	instance1.StartTestApp()
	instance2.StartTestApp()

	ip1, err = instance1.WaitForGSLB(instance2)
	require.NoError(t, err)
	ip2, err = instance1.WaitForGSLB(instance1)
	require.NoError(t, err)

	assert.NotEmpty(t, ip1)
	assert.NotEmpty(t, ip2)
	assert.Equal(t, len(ip1), len(expectedIPs))
	assert.Equal(t, len(ip2), len(expectedIPs))
	assert.ElementsMatch(t, ip1, expectedIPs)
	assert.ElementsMatch(t, ip2, expectedIPs)
	//})
}
