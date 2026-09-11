import type { ProfilePreset } from '@/types/profile'

export const PROFILE_PRESETS: ProfilePreset[] = [
  {
    id: 'standard-single-node',
    name: 'standard-single-node',
    label: 'Standard Single-Node',
    badge: 'Standard',
    description: 'Balanced general-purpose runner for development, functional smoke tests, and low-scale verification.',
    values: {
      name: 'standard-single-node',
      description: 'Standard single-node load runner profile (1 vCPU, 1Gi RAM)',
      runner_image: 'alpine:3.20',
      cpu_request: '500m',
      cpu_limit: '1000m',
      memory_request: '512Mi',
      memory_limit: '1Gi',
      node_selector: {},
      affinity: {
        node_selector_terms: [],
      },
      tolerations: [],
      active_deadline_seconds: 3600,
      runtime_class_name: '',
    },
  },
  {
    id: 'high-throughput-dedicated',
    name: 'high-throughput-dedicated',
    label: 'High-Throughput Dedicated',
    badge: 'High Compute',
    description: 'Guaranteed QoS with dedicated node affinity and tolerations for compute-intensive distributed stress tests.',
    values: {
      name: 'high-throughput-dedicated',
      description: 'Dedicated high-compute performance testing node pool profile',
      runner_image: 'alpine:3.20',
      cpu_request: '2000m',
      cpu_limit: '4000m',
      memory_request: '4Gi',
      memory_limit: '8Gi',
      node_selector: {
        'node-role.kubernetes.io/performance-runner': 'true',
      },
      affinity: {
        node_selector_terms: [
          {
            key: 'node.kubernetes.io/instance-type',
            operator: 'In',
            values: ['c5.4xlarge', 'c6i.4xlarge'],
          },
        ],
      },
      tolerations: [
        {
          key: 'dedicated',
          operator: 'Equal',
          value: 'loadgen',
          effect: 'NoSchedule',
        },
      ],
      active_deadline_seconds: 7200,
      runtime_class_name: '',
    },
  },
  {
    id: 'kernel-isolated-gvisor',
    name: 'kernel-isolated-gvisor',
    label: 'Kernel-Isolated (gVisor)',
    badge: 'Sandboxed',
    description: 'Hardened user-space kernel sandbox via gVisor runtime class for multi-tenant or untrusted scenarios.',
    values: {
      name: 'kernel-isolated-gvisor',
      description: 'Multi-tenant gVisor sandboxed runner for untrusted scenarios',
      runner_image: 'alpine:3.20',
      cpu_request: '1000m',
      cpu_limit: '2000m',
      memory_request: '1Gi',
      memory_limit: '2Gi',
      node_selector: {
        'kubernetes.io/arch': 'amd64',
      },
      affinity: {
        node_selector_terms: [],
      },
      tolerations: [
        {
          key: 'sandbox.gvisor.io/runtime',
          operator: 'Exists',
          effect: 'NoSchedule',
        },
      ],
      active_deadline_seconds: 3600,
      runtime_class_name: 'gvisor',
    },
  },
]
