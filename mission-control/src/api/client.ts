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

  createMission: (input: { goal: string; name?: string; requiredCapabilities: string[] }) =>
    request<Mission>('/api/v1/missions', {
      method: 'POST',
      body: JSON.stringify(input),
    }),

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
}

/** WebSocket URL for live events (same origin in prod; vite proxy in dev). */
export function eventsWsUrl(since = 0): string {
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const host = window.location.host
  return `${proto}://${host}/api/v1/events?since=${since}`
}
