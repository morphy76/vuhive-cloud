export interface Schedule {
  id: string
  suiteId: string
  artifactId: string
  configurationId?: string
  runnerProfileId: string
  name: string
  cronExpression: string
  k8sCronJobName: string
  isActive: boolean
  createdAt: string
  updatedAt: string
}

export interface CreateScheduleInput {
  suite_id: string
  artifact_id: string
  configuration_id?: string
  runner_profile_id: string
  name: string
  cron_expression: string
}

export interface UpdateScheduleInput {
  cron_expression?: string
  is_active?: boolean
}
