export type AffinityOperator =
  | 'In'
  | 'NotIn'
  | 'Exists'
  | 'DoesNotExist'
  | 'Gt'
  | 'Lt'

export interface NodeAffinityTerm {
  key: string
  operator: AffinityOperator
  values?: string[]
}

export interface AffinitySpec {
  node_selector_terms?: NodeAffinityTerm[]
}

export type TolerationOperator = 'Equal' | 'Exists'
export type TolerationEffect = 'NoSchedule' | 'PreferNoSchedule' | 'NoExecute'

export interface TolerationSpec {
  key?: string
  operator?: TolerationOperator
  value?: string
  effect?: TolerationEffect
  toleration_seconds?: number | null
}

export interface RunnerProfile {
  id: string
  name: string
  description?: string
  runner_image: string
  cpu_request: string
  cpu_limit: string
  memory_request: string
  memory_limit: string
  node_selector?: Record<string, string>
  affinity?: AffinitySpec
  tolerations?: TolerationSpec[]
  active_deadline_seconds?: number | null
  runtime_class_name?: string | null
  created_at: string
  updated_at: string
}

export interface CreateProfileInput {
  name: string
  description?: string
  runner_image?: string
  cpu_request?: string
  cpu_limit?: string
  memory_request?: string
  memory_limit?: string
  node_selector?: Record<string, string>
  affinity?: AffinitySpec
  tolerations?: TolerationSpec[]
  active_deadline_seconds?: number | null
  runtime_class_name?: string | null
}

export interface UpdateProfileInput {
  name: string
  description?: string
  runner_image?: string
  cpu_request?: string
  cpu_limit?: string
  memory_request?: string
  memory_limit?: string
  node_selector?: Record<string, string>
  affinity?: AffinitySpec
  tolerations?: TolerationSpec[]
  active_deadline_seconds?: number | null
  runtime_class_name?: string | null
}

export interface ProfilePreset {
  id: string
  name: string
  label: string
  badge: string
  description: string
  values: CreateProfileInput
}
