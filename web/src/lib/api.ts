import type {
  TestSuite,
  SuiteConfiguration,
  CompiledArtifact,
  HistoricalRun,
} from '@/types/suite'
import type {
  RunnerProfile,
  CreateProfileInput,
  UpdateProfileInput,
} from '@/types/profile'

const BASE_PREFIXES = ['/api/bff/v1', '/api/v1']

function getTargetUrl(prefix: string, path: string): string {
  const relPath = `${prefix}${path}`
  if (typeof window !== 'undefined' && window.location?.origin && window.location.origin !== 'null') {
    try {
      return new URL(relPath, window.location.origin).toString()
    } catch {
      return relPath
    }
  }
  return relPath
}

/**
 * Executes a fetch request trying primary BFF proxy prefix and falling back if 404 or network failure occurs.
 */
async function apiRequest<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  let lastError: any = null

  for (const prefix of BASE_PREFIXES) {
    try {
      const targetUrl = getTargetUrl(prefix, path)
      const response = await fetch(targetUrl, {
        ...options,
        headers: {
          Accept: 'application/json',
          ...(options.headers || {}),
        },
      })

      if (response.status === 404 && prefix === BASE_PREFIXES[0]) {
        // Try next prefix
        continue
      }

      if (!response.ok) {
        let errMessage = `HTTP error ${response.status}`
        try {
          const errBody = await response.json()
          if (errBody.error) errMessage = errBody.error
        } catch {
          // ignore json parse error
        }
        throw new Error(errMessage)
      }

      if (response.status === 204) {
        return undefined as unknown as T
      }

      return (await response.json()) as T
    } catch (err) {
      lastError = err
    }
  }

  throw lastError || new Error('Network request failed')
}

// Default fallback suites for offline / initial development
export const FALLBACK_SUITES: TestSuite[] = [
  {
    id: 'suite-e2e-checkout',
    name: 'Checkout & Payment Stress Test',
    description: 'E2E checkout funnel under heavy synthetic concurrency',
    state: 'ACTIVE',
    buildStatus: 'READY',
    platforms: ['linux/amd64', 'linux/arm64'],
    createdAt: new Date(Date.now() - 3600000 * 24 * 7).toISOString(),
    updatedAt: new Date(Date.now() - 600000).toISOString(),
    runCount: 24,
  },
  {
    id: 'suite-search-catalog',
    name: 'Product Catalog High Throughput',
    description: 'Read-heavy product search and filtering benchmark',
    state: 'ACTIVE',
    buildStatus: 'READY',
    platforms: ['linux/arm64'],
    createdAt: new Date(Date.now() - 3600000 * 24 * 14).toISOString(),
    updatedAt: new Date(Date.now() - 7200000).toISOString(),
    runCount: 16,
  },
  {
    id: 'suite-auth-flood',
    name: 'OAuth2 Token Grant Barrier Test',
    description: 'Distributed synchronization barrier during token refresh bursts',
    state: 'ACTIVE',
    buildStatus: 'READY',
    platforms: ['linux/amd64'],
    createdAt: new Date(Date.now() - 3600000 * 24 * 30).toISOString(),
    updatedAt: new Date(Date.now() - 86400000).toISOString(),
    runCount: 8,
  },
]

export const FALLBACK_PROFILES: RunnerProfile[] = [
  {
    id: 'profile-standard-single-node',
    name: 'standard-single-node',
    description: 'Standard single-node load runner profile (1 vCPU, 1Gi RAM)',
    runner_image: 'alpine:3.20',
    cpu_request: '500m',
    cpu_limit: '1000m',
    memory_request: '512Mi',
    memory_limit: '1Gi',
    node_selector: {},
    affinity: { node_selector_terms: [] },
    tolerations: [],
    active_deadline_seconds: 3600,
    runtime_class_name: null,
    created_at: new Date(Date.now() - 86400000 * 7).toISOString(),
    updated_at: new Date(Date.now() - 86400000).toISOString(),
  },
  {
    id: 'profile-high-throughput-dedicated',
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
    runtime_class_name: null,
    created_at: new Date(Date.now() - 86400000 * 5).toISOString(),
    updated_at: new Date(Date.now() - 3600000 * 4).toISOString(),
  },
  {
    id: 'profile-kernel-isolated-gvisor',
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
    affinity: { node_selector_terms: [] },
    tolerations: [
      {
        key: 'sandbox.gvisor.io/runtime',
        operator: 'Exists',
        effect: 'NoSchedule',
      },
    ],
    active_deadline_seconds: 3600,
    runtime_class_name: 'gvisor',
    created_at: new Date(Date.now() - 86400000 * 2).toISOString(),
    updated_at: new Date(Date.now() - 1800000).toISOString(),
  },
]

export const api = {
  async getSuites(): Promise<TestSuite[]> {
    try {
      const data = await apiRequest<{ suites: any[]; count: number }>('/suites')
      if (data && Array.isArray(data.suites)) {
        return data.suites.map((s) => ({
          id: s.id,
          name: s.name,
          description: s.description || '',
          state: s.state || 'DRAFT',
          createdAt: s.created_at || s.createdAt || new Date().toISOString(),
          updatedAt: s.updated_at || s.updatedAt || new Date().toISOString(),
          buildStatus: 'READY',
          platforms: ['linux/amd64', 'linux/arm64'],
          runCount: s.run_count || 0,
        }))
      }
      return FALLBACK_SUITES
    } catch (e) {
      console.warn('Unable to load live suites from API, using fallback cache:', e)
      return FALLBACK_SUITES
    }
  },

  async getSuite(id: string): Promise<TestSuite> {
    try {
      const s = await apiRequest<any>(`/suites/${encodeURIComponent(id)}`)
      return {
        id: s.id,
        name: s.name,
        description: s.description || '',
        state: s.state || 'DRAFT',
        createdAt: s.created_at || s.createdAt || new Date().toISOString(),
        updatedAt: s.updated_at || s.updatedAt || new Date().toISOString(),
        buildStatus: 'READY',
        platforms: ['linux/amd64', 'linux/arm64'],
      }
    } catch {
      const found = FALLBACK_SUITES.find((s) => s.id === id)
      if (found) return found
      throw new Error(`Suite ${id} not found`)
    }
  },

  async createSuite(data: { name: string; description?: string }): Promise<TestSuite> {
    const s = await apiRequest<any>('/suites', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })

    return {
      id: s.id,
      name: s.name,
      description: s.description || '',
      state: s.state || 'DRAFT',
      createdAt: s.created_at || s.createdAt || new Date().toISOString(),
      updatedAt: s.updated_at || s.updatedAt || new Date().toISOString(),
      buildStatus: 'PENDING',
      platforms: [],
    }
  },

  async updateSuite(
    id: string,
    data: { name: string; description?: string; state?: string }
  ): Promise<TestSuite> {
    const s = await apiRequest<any>(`/suites/${encodeURIComponent(id)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })

    return {
      id: s.id,
      name: s.name,
      description: s.description || '',
      state: s.state,
      createdAt: s.created_at || s.createdAt,
      updatedAt: s.updated_at || s.updatedAt,
    }
  },

  async deleteSuite(id: string): Promise<void> {
    await apiRequest<void>(`/suites/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    })
  },

  async getSuiteConfigs(suiteId: string): Promise<SuiteConfiguration[]> {
    try {
      const res = await apiRequest<{ configs: any[] }>(
        `/suites/${encodeURIComponent(suiteId)}/configs`
      )
      if (res && Array.isArray(res.configs)) {
        return res.configs.map((c) => ({
          id: c.id,
          suiteId: c.suite_id || suiteId,
          name: c.name,
          contentYaml: c.content_yaml,
          s3ConfigKey: c.s3_config_key,
          isDefault: !!c.is_default,
          createdAt: c.created_at,
        }))
      }
      return []
    } catch {
      return [
        {
          id: `cfg-${suiteId}-default`,
          suiteId,
          name: 'default-config.yaml',
          contentYaml: 'duration: 5m\nconcurrency: 50\nramp_up: 30s\n',
          isDefault: true,
          createdAt: new Date().toISOString(),
        },
      ]
    }
  },

  async createSuiteConfig(
    suiteId: string,
    data: { name: string; content_yaml: string; is_default?: boolean }
  ): Promise<SuiteConfiguration> {
    const c = await apiRequest<any>(`/suites/${encodeURIComponent(suiteId)}/configs`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })

    return {
      id: c.id,
      suiteId: c.suite_id || suiteId,
      name: c.name,
      contentYaml: c.content_yaml,
      s3ConfigKey: c.s3_config_key,
      isDefault: !!c.is_default,
      createdAt: c.created_at,
    }
  },

  async updateSuiteConfig(
    suiteId: string,
    configId: string,
    data: { name: string; content_yaml: string; is_default?: boolean }
  ): Promise<SuiteConfiguration> {
    const c = await apiRequest<any>(
      `/suites/${encodeURIComponent(suiteId)}/configs/${encodeURIComponent(configId)}`,
      {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      }
    )

    return {
      id: c.id,
      suiteId: c.suite_id || suiteId,
      name: c.name,
      contentYaml: c.content_yaml,
      s3ConfigKey: c.s3_config_key,
      isDefault: !!c.is_default,
      createdAt: c.created_at,
    }
  },

  async deleteSuiteConfig(suiteId: string, configId: string): Promise<void> {
    await apiRequest<void>(
      `/suites/${encodeURIComponent(suiteId)}/configs/${encodeURIComponent(configId)}`,
      {
        method: 'DELETE',
      }
    )
  },

  async getSuiteArtifacts(suiteId: string): Promise<CompiledArtifact[]> {
    try {
      const res = await apiRequest<{ artifacts: any[] }>(
        `/suites/${encodeURIComponent(suiteId)}/artifacts`
      )
      if (res && Array.isArray(res.artifacts)) {
        return res.artifacts.map((a) => ({
          id: a.id,
          suiteId: a.suite_id || suiteId,
          platform: a.platform,
          s3BinaryKey: a.s3_binary_key,
          sha256Checksum: a.sha256_checksum,
          buildLogsS3Key: a.build_logs_s3_key,
          status: a.status || 'READY',
          errorMessage: a.error_message,
          createdAt: a.created_at,
        }))
      }
      return []
    } catch {
      return [
        {
          id: `art-${suiteId}-amd64`,
          suiteId,
          platform: 'linux/amd64',
          sha256Checksum: 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
          status: 'READY',
          createdAt: new Date(Date.now() - 3600000).toISOString(),
        },
        {
          id: `art-${suiteId}-arm64`,
          suiteId,
          platform: 'linux/arm64',
          sha256Checksum: 'a7c938144298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852ba',
          status: 'READY',
          createdAt: new Date(Date.now() - 3600000).toISOString(),
        },
      ]
    }
  },

  async uploadSuiteBuild(
    suiteId: string,
    formData: FormData,
    onProgress?: (percent: number) => void
  ): Promise<any> {
    if (!onProgress || typeof XMLHttpRequest === 'undefined') {
      return apiRequest<any>(`/suites/${encodeURIComponent(suiteId)}/builds`, {
        method: 'POST',
        body: formData,
      })
    }

    return new Promise((resolve, reject) => {
      const path = `/suites/${encodeURIComponent(suiteId)}/builds`
      let prefixIndex = 0

      const attemptUpload = () => {
        const prefix = BASE_PREFIXES[prefixIndex]
        const targetUrl = getTargetUrl(prefix, path)
        const xhr = new XMLHttpRequest()

        xhr.open('POST', targetUrl)
        xhr.setRequestHeader('Accept', 'application/json')

        xhr.upload.onprogress = (event) => {
          if (event.lengthComputable) {
            const percent = Math.round((event.loaded / event.total) * 100)
            onProgress(percent)
          }
        }

        xhr.onload = () => {
          if (xhr.status === 404 && prefixIndex < BASE_PREFIXES.length - 1) {
            prefixIndex++
            attemptUpload()
            return
          }

          if (xhr.status >= 200 && xhr.status < 300) {
            try {
              const res = xhr.responseText ? JSON.parse(xhr.responseText) : {}
              resolve(res)
            } catch {
              resolve({})
            }
            return
          }

          let errMessage = `HTTP error ${xhr.status}`
          try {
            const body = JSON.parse(xhr.responseText)
            if (body.error) errMessage = body.error
          } catch {
            // ignore
          }
          reject(new Error(errMessage))
        }

        xhr.onerror = () => {
          if (prefixIndex < BASE_PREFIXES.length - 1) {
            prefixIndex++
            attemptUpload()
          } else {
            reject(new Error('Network request failed'))
          }
        }

        xhr.send(formData)
      }

      attemptUpload()
    })
  },

  async getSuiteRuns(suiteId: string): Promise<HistoricalRun[]> {
    try {
      const res = await apiRequest<{ runs: any[] }>(
        `/runs?suite_id=${encodeURIComponent(suiteId)}`
      )
      if (res && Array.isArray(res.runs)) {
        return res.runs.map((r) => ({
          id: r.id,
          suiteId: r.suite_id,
          artifactId: r.artifact_id,
          configurationId: r.configuration_id,
          runnerProfileId: r.runner_profile_id,
          status: r.status,
          k8sJobName: r.k8s_job_name,
          startedAt: r.started_at,
          finishedAt: r.finished_at,
          durationMs: r.duration_ms,
          exitCode: r.exit_code,
          slaPassed: r.sla_passed,
          metrics: r.metrics
            ? {
                totalIterations: r.metrics.total_iterations,
                totalRequests: r.metrics.total_requests,
                avgTps: r.metrics.avg_tps,
                p50DurationMs: r.metrics.p50_duration_ms,
                p90DurationMs: r.metrics.p90_duration_ms,
                p95DurationMs: r.metrics.p95_duration_ms,
                p99DurationMs: r.metrics.p99_duration_ms,
                errorRatePct: r.metrics.error_rate_pct,
              }
            : undefined,
          createdAt: r.created_at,
        }))
      }
      return []
    } catch {
      return [
        {
          id: `run-${suiteId}-01`,
          suiteId,
          status: 'COMPLETED',
          durationMs: 300000,
          exitCode: 0,
          slaPassed: true,
          startedAt: new Date(Date.now() - 7200000).toISOString(),
          finishedAt: new Date(Date.now() - 6900000).toISOString(),
          metrics: {
            avgTps: 1850,
            p95DurationMs: 42,
            errorRatePct: 0.0,
          },
          createdAt: new Date(Date.now() - 7200000).toISOString(),
        },
      ]
    }
  },

  async getProfiles(): Promise<RunnerProfile[]> {
    try {
      const res = await apiRequest<{ profiles: any[]; count: number }>('/profiles')
      if (res && Array.isArray(res.profiles)) {
        return res.profiles.map((p) => ({
          id: p.id,
          name: p.name,
          description: p.description || '',
          runner_image: p.runner_image || 'alpine:3.20',
          cpu_request: p.cpu_request || '500m',
          cpu_limit: p.cpu_limit || '1000m',
          memory_request: p.memory_request || '512Mi',
          memory_limit: p.memory_limit || '1Gi',
          node_selector: p.node_selector || {},
          affinity: p.affinity || { node_selector_terms: [] },
          tolerations: p.tolerations || [],
          active_deadline_seconds: p.active_deadline_seconds,
          runtime_class_name: p.runtime_class_name,
          created_at: p.created_at || new Date().toISOString(),
          updated_at: p.updated_at || new Date().toISOString(),
        }))
      }
      return FALLBACK_PROFILES
    } catch (e) {
      console.warn('Unable to load live profiles from API, using fallback cache:', e)
      return FALLBACK_PROFILES
    }
  },

  async getProfile(id: string): Promise<RunnerProfile> {
    try {
      const p = await apiRequest<any>(`/profiles/${encodeURIComponent(id)}`)
      return {
        id: p.id,
        name: p.name,
        description: p.description || '',
        runner_image: p.runner_image || 'alpine:3.20',
        cpu_request: p.cpu_request || '500m',
        cpu_limit: p.cpu_limit || '1000m',
        memory_request: p.memory_request || '512Mi',
        memory_limit: p.memory_limit || '1Gi',
        node_selector: p.node_selector || {},
        affinity: p.affinity || { node_selector_terms: [] },
        tolerations: p.tolerations || [],
        active_deadline_seconds: p.active_deadline_seconds,
        runtime_class_name: p.runtime_class_name,
        created_at: p.created_at || new Date().toISOString(),
        updated_at: p.updated_at || new Date().toISOString(),
      }
    } catch {
      const found = FALLBACK_PROFILES.find((p) => p.id === id)
      if (found) return found
      throw new Error(`Profile ${id} not found`)
    }
  },

  async createProfile(data: CreateProfileInput): Promise<RunnerProfile> {
    const p = await apiRequest<any>('/profiles', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })
    return {
      id: p.id,
      name: p.name,
      description: p.description || '',
      runner_image: p.runner_image || 'alpine:3.20',
      cpu_request: p.cpu_request || '500m',
      cpu_limit: p.cpu_limit || '1000m',
      memory_request: p.memory_request || '512Mi',
      memory_limit: p.memory_limit || '1Gi',
      node_selector: p.node_selector || {},
      affinity: p.affinity || { node_selector_terms: [] },
      tolerations: p.tolerations || [],
      active_deadline_seconds: p.active_deadline_seconds,
      runtime_class_name: p.runtime_class_name,
      created_at: p.created_at || new Date().toISOString(),
      updated_at: p.updated_at || new Date().toISOString(),
    }
  },

  async updateProfile(id: string, data: UpdateProfileInput): Promise<RunnerProfile> {
    const p = await apiRequest<any>(`/profiles/${encodeURIComponent(id)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })
    return {
      id: p.id,
      name: p.name,
      description: p.description || '',
      runner_image: p.runner_image || 'alpine:3.20',
      cpu_request: p.cpu_request || '500m',
      cpu_limit: p.cpu_limit || '1000m',
      memory_request: p.memory_request || '512Mi',
      memory_limit: p.memory_limit || '1Gi',
      node_selector: p.node_selector || {},
      affinity: p.affinity || { node_selector_terms: [] },
      tolerations: p.tolerations || [],
      active_deadline_seconds: p.active_deadline_seconds,
      runtime_class_name: p.runtime_class_name,
      created_at: p.created_at || new Date().toISOString(),
      updated_at: p.updated_at || new Date().toISOString(),
    }
  },

  async deleteProfile(id: string): Promise<void> {
    await apiRequest<void>(`/profiles/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    })
  },
}
