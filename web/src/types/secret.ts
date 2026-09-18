export interface SuiteSecret {
  id: string
  suiteId: string
  key: string
  createdAt: string
  updatedAt: string
}

export interface CreateSecretPayload {
  key: string
  value: string
}

export interface UpdateSecretPayload {
  value: string
}
