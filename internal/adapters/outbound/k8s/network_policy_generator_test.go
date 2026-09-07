package k8s_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/k8s"
)

func TestNetworkPolicyGenerator_Generate(t *testing.T) {
	generator := k8s.NewNetworkPolicyGenerator()

	t.Run("generate default runner egress network policy", func(t *testing.T) {
		opts := k8s.NetworkPolicyOptions{
			Name:                  "vuhive-runner-isolation",
			Namespace:             "vuhive-runners",
			ControlPlaneNamespace: "vuhive-system",
			ControlPlanePort:      8080,
			S3Port:                9000,
			DenyMetadata:          true,
			DenyClusterCIDRs:      []string{"10.96.0.0/12"},
			AllowedTargetCIDRs:    []string{"0.0.0.0/0"},
		}

		netPol, err := generator.Generate(opts)
		require.NoError(t, err)
		require.NotNil(t, netPol)

		// Metadata & Selector
		assert.Equal(t, "vuhive-runner-isolation", netPol.Name)
		assert.Equal(t, "vuhive-runners", netPol.Namespace)
		assert.Equal(t, "vuhive-runner", netPol.Spec.PodSelector.MatchLabels["app.kubernetes.io/name"])
		assert.Contains(t, netPol.Spec.PolicyTypes, networkingv1.PolicyTypeEgress)

		// Check Egress Rules
		require.NotEmpty(t, netPol.Spec.Egress)

		// 1. DNS Rule
		dnsRule := netPol.Spec.Egress[0]
		require.Len(t, dnsRule.Ports, 2)
		assert.Equal(t, corev1.ProtocolUDP, *dnsRule.Ports[0].Protocol)
		assert.Equal(t, intstr.FromInt32(53), *dnsRule.Ports[0].Port)
		assert.Equal(t, corev1.ProtocolTCP, *dnsRule.Ports[1].Protocol)
		assert.Equal(t, intstr.FromInt32(53), *dnsRule.Ports[1].Port)

		// 2. Control Plane & S3 Rule
		cpRule := netPol.Spec.Egress[1]
		require.NotEmpty(t, cpRule.To)
		assert.Equal(t, "vuhive-system", cpRule.To[0].NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"])
		require.Len(t, cpRule.Ports, 2)
		assert.Equal(t, intstr.FromInt32(8080), *cpRule.Ports[0].Port)
		assert.Equal(t, intstr.FromInt32(9000), *cpRule.Ports[1].Port)

		// 3. Target rule with exceptions
		targetRule := netPol.Spec.Egress[2]
		require.NotEmpty(t, targetRule.To)
		ipBlock := targetRule.To[0].IPBlock
		require.NotNil(t, ipBlock)
		assert.Equal(t, "0.0.0.0/0", ipBlock.CIDR)
		assert.Contains(t, ipBlock.Except, "169.254.169.254/32")
		assert.Contains(t, ipBlock.Except, "10.96.0.0/12")
	})

	t.Run("generate network policy in same release namespace", func(t *testing.T) {
		opts := k8s.NetworkPolicyOptions{
			Name:                  "vuhive-runner-isolation",
			Namespace:             "vuhive-system",
			ControlPlaneNamespace: "vuhive-system",
			ControlPlanePort:      8080,
			S3Port:                9000,
			DenyMetadata:          true,
			DenyClusterCIDRs:      []string{"10.96.0.0/12"},
			AllowedTargetCIDRs:    []string{"192.0.2.0/24"},
		}

		netPol, err := generator.Generate(opts)
		require.NoError(t, err)
		require.NotNil(t, netPol)

		// PodSelector in same namespace selects control plane pods
		cpRule := netPol.Spec.Egress[1]
		require.NotEmpty(t, cpRule.To)
		assert.Equal(t, "vuhive-cloud", cpRule.To[0].PodSelector.MatchLabels["app.kubernetes.io/name"])

		// Target CIDR is explicit
		targetRule := netPol.Spec.Egress[2]
		require.NotNil(t, targetRule.To[0].IPBlock)
		assert.Equal(t, "192.0.2.0/24", targetRule.To[0].IPBlock.CIDR)
	})

	t.Run("validation error on missing name or namespace", func(t *testing.T) {
		_, err := generator.Generate(k8s.NetworkPolicyOptions{})
		assert.Error(t, err)
	})
}
