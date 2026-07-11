/** Hermes Host API types — product surface only (no vendor fields). */

export type Envelope<T> = {
  data: T
  error: { code: string; message: string; remediation?: string } | null
}

export type Health = {
  status: string
  profile?: string
  version?: string
  product?: string
  seq?: number
}

export type Mission = {
  id: string
  name: string
  goal: string
  state: string
  requiredCapabilities: string[]
  labels?: Record<string, string>
  costUsd?: number
  output?: string
  providerId?: string
  runtimeId?: string
  modelId?: string
  routeReason?: string
  createdAt?: string
  updatedAt?: string
  cancelReason?: string
}

export type HostEvent = {
  seq: number
  type: string
  missionId?: string
  ts?: string
  data?: Record<string, unknown>
}

export type PluginItem = {
  id: string
  name?: string
  version?: string
  kind: string
  spec?: Record<string, unknown>
  labels?: Record<string, string>
  enabled?: boolean
}

export type MemoryEntry = {
  id: string
  kind?: string
  content?: string
  trust?: string
  missionId?: string
  provenance?: Record<string, string>
}

export type CredentialMeta = {
  handle: string
  scope?: string
  label?: string
  pluginId?: string
  createdAt?: string
}
