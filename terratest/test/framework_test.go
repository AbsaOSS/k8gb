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
	"k8gbterratest/utils"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGslbCreated(t *testing.T) {
	t.Parallel()
	const host = "roundrobin-test.cloud.example.com"
	const gslbPath = "../examples/roundrobin2.yaml"

	instance1, err := utils.NewWorkflow(t, "k3d-test-gslb1", 5053).
		WithGslb(gslbPath, host).
		WithTestApp().
		Start()
	require.NoError(t, err)
	defer instance1.Kill()
	instance2, err := utils.NewWorkflow(t, "k3d-test-gslb2", 5054).
		WithGslb(gslbPath, host).
		WithTestApp().
		Start()
	require.NoError(t, err)
	defer instance2.Kill()

	ingressIP1 := instance1.GetIngressIPs()
	ingressIP2 := instance2.GetIngressIPs()
	expectedIPs := append(ingressIP1, ingressIP2...)

	t.Run("round-robin on two concurrent clusters with podinfo running", func(t *testing.T) {
		ip1, err := instance1.WaitForGSLB(instance2)
		require.NoError(t, err)
		ip2, err := instance2.WaitForGSLB(instance1)
		require.NoError(t, err)
		assert.ElementsMatch(t, expectedIPs, ip1)
		assert.ElementsMatch(t, expectedIPs, ip2)
	})

	t.Run("kill podinfo on the first cluster", func(t *testing.T) {
		instance1.StopTestApp()
		ip1, err := instance1.WaitForGSLB(instance2)
		require.NoError(t, err)
		ip2, err := instance2.WaitForGSLB(instance1)
		require.NoError(t, err)
		require.True(t, reflect.DeepEqual(ingressIP2, ip1))
		require.True(t, reflect.DeepEqual(ingressIP2, ip2))
	})

	t.Run("kill podinfo on the second cluster", func(t *testing.T) {
		instance2.StopTestApp()
		ip2, err := instance2.WaitForGSLB(instance1)
		require.NoError(t, err)
		ip1, err := instance1.WaitForGSLB(instance2)
		require.NoError(t, err)
		assert.ElementsMatch(t, ip1, ip2)
	})

	t.Run("start podinfo on the both clusters", func(t *testing.T) {
		// start app in the both clusters
		instance1.StartTestApp()
		instance2.StartTestApp()

		ip1, err := instance1.WaitForGSLB(instance2)
		require.NoError(t, err)
		ip2, err := instance2.WaitForGSLB(instance1)
		require.NoError(t, err)

		assert.NotEmpty(t, ip1)
		assert.NotEmpty(t, ip2)
		assert.Equal(t, len(ip1), len(expectedIPs))
		assert.Equal(t, len(ip2), len(expectedIPs))
		assert.ElementsMatch(t, ip1, expectedIPs)
		assert.ElementsMatch(t, ip2, expectedIPs)
	})
}
