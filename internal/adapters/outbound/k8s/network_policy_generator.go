package k8s

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// NetworkPolicyOptions specifies the configuration for generating runner egress NetworkPolicies.
type NetworkPolicyOptions struct {
	Name                  string
	Namespace             string
	ControlPlaneNamespace string
	ControlPlanePort      int32
	S3Port                int32
	DenyMetadata          bool
	DenyClusterCIDRs      []string
	AllowedTargetCIDRs    []string
	AllowedEgressRules    []networkingv1.NetworkPolicyEgressRule
}

// NetworkPolicyGenerator creates Kubernetes networking.k8s.io/v1 NetworkPolicy manifests.
type NetworkPolicyGenerator struct{}

// NewNetworkPolicyGenerator constructs a new NetworkPolicyGenerator.
func NewNetworkPolicyGenerator() *NetworkPolicyGenerator {
	return &NetworkPolicyGenerator{}
}

// Generate constructs a restricted egress NetworkPolicy manifest for runner pods.
func (g *NetworkPolicyGenerator) Generate(opts NetworkPolicyOptions) (*networkingv1.NetworkPolicy, error) {
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		return nil, fmt.Errorf("network policy name cannot be empty")
	}
	namespace := strings.TrimSpace(opts.Namespace)
	if namespace == "" {
		return nil, fmt.Errorf("network policy namespace cannot be empty")
	}

	udpProto := corev1.ProtocolUDP
	tcpProto := corev1.ProtocolTCP
	dnsPort := intstr.FromInt32(53)

	var egressRules []networkingv1.NetworkPolicyEgressRule

	// 1. DNS Resolution (UDP/TCP 53)
	egressRules = append(egressRules, networkingv1.NetworkPolicyEgressRule{
		Ports: []networkingv1.NetworkPolicyPort{
			{
				Protocol: &udpProto,
				Port:     &dnsPort,
			},
			{
				Protocol: &tcpProto,
				Port:     &dnsPort,
			},
		},
	})

	// 2. Control Plane & S3 Callbacks
	cpPort := opts.ControlPlanePort
	if cpPort <= 0 {
		cpPort = 8080
	}
	s3Port := opts.S3Port
	if s3Port <= 0 {
		s3Port = 9000
	}

	cpPorts := []networkingv1.NetworkPolicyPort{
		{
			Protocol: &tcpProto,
			Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: cpPort},
		},
		{
			Protocol: &tcpProto,
			Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: s3Port},
		},
	}

	cpNamespace := strings.TrimSpace(opts.ControlPlaneNamespace)
	if cpNamespace == "" {
		cpNamespace = namespace
	}

	if cpNamespace != namespace {
		egressRules = append(egressRules, networkingv1.NetworkPolicyEgressRule{
			To: []networkingv1.NetworkPolicyPeer{
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"kubernetes.io/metadata.name": cpNamespace,
						},
					},
				},
			},
			Ports: cpPorts,
		})
	} else {
		egressRules = append(egressRules, networkingv1.NetworkPolicyEgressRule{
			To: []networkingv1.NetworkPolicyPeer{
				{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/name": "vuhive-cloud",
						},
					},
				},
			},
			Ports: cpPorts,
		})
	}

	// 3. Target Egress (Excluding Cloud Metadata & Internal Cluster CIDRs)
	var exceptions []string
	if opts.DenyMetadata {
		exceptions = append(exceptions, "169.254.169.254/32")
	}
	for _, cidr := range opts.DenyClusterCIDRs {
		trimmed := strings.TrimSpace(cidr)
		if trimmed != "" {
			exceptions = append(exceptions, trimmed)
		}
	}

	targetCIDRs := opts.AllowedTargetCIDRs
	if len(targetCIDRs) == 0 {
		targetCIDRs = []string{"0.0.0.0/0"}
	}

	for _, target := range targetCIDRs {
		trimmed := strings.TrimSpace(target)
		if trimmed == "" {
			continue
		}
		rule := networkingv1.NetworkPolicyEgressRule{
			To: []networkingv1.NetworkPolicyPeer{
				{
					IPBlock: &networkingv1.IPBlock{
						CIDR:   trimmed,
						Except: exceptions,
					},
				},
			},
		}
		egressRules = append(egressRules, rule)
	}

	// 4. Custom additional egress rules if provided
	if len(opts.AllowedEgressRules) > 0 {
		egressRules = append(egressRules, opts.AllowedEgressRules...)
	}

	netPol := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "vuhive-runner-isolation",
				"app.kubernetes.io/managed-by": "vuhive-cloud",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": "vuhive-runner",
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeEgress,
			},
			Egress: egressRules,
		},
	}

	return netPol, nil
}
