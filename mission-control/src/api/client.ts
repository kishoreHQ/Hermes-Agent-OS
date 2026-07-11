import type {
  CredentialMeta,
  Envelope,
  Health,
  HostEvent,
  MemoryEntry,
  Mission,
  PluginItem,
} from './types'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: {
      Accept: 'application/json',
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...init?.headers,
    },
  })
  const body = (await res.json()) as Envelope<T>
  if (!res.ok || body.error) {
    const msg = body.error?.message ?? `HTTP ${res.status}`
    throw new Error(msg)
  }
  return body.data
}

export const api = {
  health: () => request<Health>('/api/v1/health'),

  listMissions: (state?: string) =>
    request<Mission[]>(`/api/v1/missions${state ? `?state=${encodeURIComponent(state)}` : ''}`),

  getMission: (id: string) => request<Mission>(`/api/v1/missions/${encodeURIComponent(id)}`),

  createMission: (input: {
    goal: string
    name?: string
    requiredCapabilities: string[]
    preferProvider?: string
    requireProvider?: string
    preferModel?: string
    model?: string
    providers?: string[]
    failover?: boolean
    labels?: Record<string, string>
  }) =>
    request<Mission>('/api/v1/missions', {
      method: 'POST',
      body: JSON.stringify(input),
    }),

  listProviderModels: () =>
    request<
      Array<{
        id: string
        name?: string
        health?: string
        local?: boolean
        costTier?: string
        models?: Array<{ id: string; costTier?: string }>
      }>
    >('/api/v1/providers/models'),

  cancelMission: (id: string, reason = 'ui-cancel') =>
    request<{ id: string; state: string }>(`/api/v1/missions/${encodeURIComponent(id)}/cancel`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    }),

  events: (since = 0, mission?: string) => {
    const q = new URLSearchParams({ since: String(since), format: 'json' })
    if (mission) q.set('mission', mission)
    return request<HostEvent[]>(`/api/v1/events?${q}`)
  },

  replay: (id: string) =>
    request<{ missionId: string; events: HostEvent[] }>(
      `/api/v1/replay/${encodeURIComponent(id)}`,
    ),

  registry: (kind: 'providers' | 'runtimes' | 'tools' | 'agents') =>
    request<PluginItem[]>(`/api/v1/registry/${kind}`),

  plugins: () => request<PluginItem[]>('/api/v1/plugins'),

  memorySearch: (params?: { q?: string; mission?: string; kind?: string }) => {
    const q = new URLSearchParams()
    if (params?.q) q.set('q', params.q)
    if (params?.mission) q.set('mission', params.mission)
    if (params?.kind) q.set('kind', params.kind)
    const s = q.toString()
    return request<MemoryEntry[]>(`/api/v1/memory/search${s ? `?${s}` : ''}`)
  },

  credentials: () => request<CredentialMeta[]>('/api/v1/credentials'),

  // Command Deck (H3.1)
  probeConnections: () => request<Record<string, unknown>[]>('/api/v1/connections/probe'),
  listConnections: () => request<Record<string, unknown>[]>('/api/v1/connections'),
  registerConnection: (body: { pluginId: string; kind?: string; name?: string }) =>
    request<Record<string, unknown>>('/api/v1/connections', {
      method: 'POST',
      body: JSON.stringify(body),
    }),

  listSessions: () => request<Record<string, unknown>[]>('/api/v1/sessions'),
  createSession: (body: { runtime?: string; provider?: string }) =>
    request<Record<string, unknown>>('/api/v1/sessions', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  sessionMessage: (id: string, text: string) =>
    request<Record<string, unknown>>(`/api/v1/sessions/${encodeURIComponent(id)}/message`, {
      method: 'POST',
      body: JSON.stringify({ text }),
    }),

  listBoards: () => request<Record<string, unknown>[]>('/api/v1/boards'),
  listTasks: () => request<Record<string, unknown>[]>('/api/v1/tasks'),
  createTask: (body: { title: string; column?: string; capabilities?: string[] }) =>
    request<Record<string, unknown>>('/api/v1/tasks', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  claimTask: (id: string, assignee = 'operator') =>
    request<Record<string, unknown>>(`/api/v1/tasks/${encodeURIComponent(id)}/claim`, {
      method: 'POST',
      body: JSON.stringify({ assignee }),
    }),
  moveTask: (id: string, column: string) =>
    request<Record<string, unknown>>(`/api/v1/tasks/${encodeURIComponent(id)}/move`, {
      method: 'POST',
      body: JSON.stringify({ column }),
    }),

  listRoutines: () => request<Record<string, unknown>[]>('/api/v1/routines'),
  fireRoutine: (id: string) =>
    request<Record<string, unknown>>(`/api/v1/routines/${encodeURIComponent(id)}/fire`, {
      method: 'POST',
    }),

  listTools: () => request<Record<string, unknown>[]>('/api/v1/tools'),
  invokeTool: (id: string, input: Record<string, unknown>) =>
    request<Record<string, unknown>>(`/api/v1/tools/${encodeURIComponent(id)}/invoke`, {
      method: 'POST',
      body: JSON.stringify({ input }),
    }),
  toolInvocations: () => request<Record<string, unknown>[]>('/api/v1/tools/invocations'),
}

/** WebSocket URL for live events (same origin in prod; vite proxy in dev). */
export function eventsWsUrl(since = 0): string {
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const host = window.location.host
  return `${proto}://${host}/api/v1/events?since=${since}`
}
