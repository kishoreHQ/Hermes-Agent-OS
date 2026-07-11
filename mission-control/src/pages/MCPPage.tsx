import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

type MCPServer = {
  id: string
  name: string
  transport: string
  command?: string
  args?: string[]
  url?: string
  enabled?: boolean
}
type MCPStatus = {
  id: string
  name: string
  state: string
  error?: string
  tools?: string[]
  transport: string
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: {
      Accept: 'application/json',
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...init?.headers,
    },
  })
  const body = await res.json()
  if (!res.ok || body.error) throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  return body.data as T
}

export function MCPPage() {
  const qc = useQueryClient()
  const data = useQuery({
    queryKey: ['mcp'],
    queryFn: () =>
      request<{ servers: MCPServer[]; statuses: MCPStatus[] }>('/api/v1/mcp/servers'),
    refetchInterval: 5000,
  })
  const [id, setId] = useState('filesystem')
  const [name, setName] = useState('Filesystem MCP')
  const [transport, setTransport] = useState<'stdio' | 'http'>('stdio')
  const [command, setCommand] = useState('npx')
  const [args, setArgs] = useState('-y @modelcontextprotocol/server-filesystem /tmp')
  const [url, setUrl] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [ok, setOk] = useState<string | null>(null)

  const add = useMutation({
    mutationFn: () =>
      request<unknown>('/api/v1/mcp/servers', {
        method: 'POST',
        body: JSON.stringify({
          id,
          name,
          transport,
          command: transport === 'stdio' ? command : undefined,
          args: transport === 'stdio' ? args.split(/\s+/).filter(Boolean) : undefined,
          url: transport === 'http' ? url : undefined,
          enabled: true,
        }),
      }),
    onSuccess: () => {
      setOk('Connected / registered')
      setError(null)
      void qc.invalidateQueries({ queryKey: ['mcp'] })
      void qc.invalidateQueries({ queryKey: ['tools'] })
    },
    onError: (e: Error) => {
      setError(e.message)
      setOk(null)
    },
  })

  const del = useMutation({
    mutationFn: (sid: string) =>
      request(`/api/v1/mcp/servers/${encodeURIComponent(sid)}`, { method: 'DELETE' }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['mcp'] }),
  })

  const statuses = data.data?.statuses ?? []
  const servers = data.data?.servers ?? []

  return (
    <div className="page">
      <h1 className="page-title">MCP</h1>
      <p className="page-sub">
        Connect real MCP servers (stdio or HTTP). Tools register as{' '}
        <code className="font-mono text-xs">mcp.&lt;server&gt;.&lt;tool&gt;</code> for agent-loop
        missions.
      </p>

      <section className="card p-4 mt-5 max-w-2xl space-y-3">
        <div className="section-label">Add MCP server</div>
        <label className="block text-sm">
          <span className="text-xs text-[var(--ink-2)]">ID</span>
          <input className="input mt-1" value={id} onChange={(e) => setId(e.target.value)} />
        </label>
        <label className="block text-sm">
          <span className="text-xs text-[var(--ink-2)]">Name</span>
          <input className="input mt-1" value={name} onChange={(e) => setName(e.target.value)} />
        </label>
        <label className="block text-sm">
          <span className="text-xs text-[var(--ink-2)]">Transport</span>
          <select
            className="input mt-1"
            value={transport}
            onChange={(e) => setTransport(e.target.value as 'stdio' | 'http')}
          >
            <option value="stdio">stdio</option>
            <option value="http">http</option>
          </select>
        </label>
        {transport === 'stdio' ? (
          <>
            <label className="block text-sm">
              <span className="text-xs text-[var(--ink-2)]">Command</span>
              <input className="input mt-1 font-mono text-sm" value={command} onChange={(e) => setCommand(e.target.value)} />
            </label>
            <label className="block text-sm">
              <span className="text-xs text-[var(--ink-2)]">Args (space-separated)</span>
              <input className="input mt-1 font-mono text-sm" value={args} onChange={(e) => setArgs(e.target.value)} />
            </label>
          </>
        ) : (
          <label className="block text-sm">
            <span className="text-xs text-[var(--ink-2)]">URL</span>
            <input className="input mt-1 font-mono text-sm" value={url} onChange={(e) => setUrl(e.target.value)} />
          </label>
        )}
        {error && <p className="text-sm text-[var(--fail)]">{error}</p>}
        {ok && <p className="text-sm text-[var(--ok)]">{ok}</p>}
        <button type="button" className="btn btn-primary" disabled={add.isPending} onClick={() => add.mutate()}>
          {add.isPending ? 'Connecting…' : 'Add & connect'}
        </button>
      </section>

      <section className="card mt-6 overflow-hidden">
        <div className="px-4 py-3 border-b border-[var(--line)] section-label !mb-0">
          Servers ({servers.length})
        </div>
        <ul className="divide-y divide-[var(--line)]">
          {statuses.map((st) => (
            <li key={st.id} className="px-4 py-3 flex justify-between gap-3">
              <div>
                <div className="font-medium">
                  {st.name}{' '}
                  <span className="chip">{st.state}</span>{' '}
                  <span className="chip">{st.transport}</span>
                </div>
                <div className="font-mono text-xs text-[var(--ink-2)]">{st.id}</div>
                {st.error && <div className="text-xs text-[var(--fail)] mt-1">{st.error}</div>}
                {st.tools && st.tools.length > 0 && (
                  <div className="text-xs text-[var(--ink-2)] mt-1">tools: {st.tools.join(', ')}</div>
                )}
              </div>
              <button type="button" className="btn btn-danger" onClick={() => del.mutate(st.id)}>
                Remove
              </button>
            </li>
          ))}
          {statuses.length === 0 && (
            <li className="p-4 text-sm text-[var(--ink-2)]">No MCP servers connected yet.</li>
          )}
        </ul>
      </section>
    </div>
  )
}
