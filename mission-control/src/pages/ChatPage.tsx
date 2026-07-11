import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/api/client'

type ChatMessage = { role: string; content: string; at: string; missionId?: string }
type ChatSession = {
  id: string
  title: string
  messages: ChatMessage[]
  provider?: string
  model?: string
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

export function ChatPage() {
  const qc = useQueryClient()
  const sessions = useQuery({
    queryKey: ['chat-sessions'],
    queryFn: () => request<ChatSession[]>('/api/v1/chat/sessions'),
    refetchInterval: 5000,
  })
  const fleet = useQuery({ queryKey: ['provider-models'], queryFn: api.listProviderModels })
  const [activeId, setActiveId] = useState<string | null>(null)
  const [text, setText] = useState('')
  const [provider, setProvider] = useState('')
  const [model, setModel] = useState('')
  const [error, setError] = useState<string | null>(null)

  const active = useQuery({
    queryKey: ['chat-session', activeId],
    queryFn: () => request<ChatSession>(`/api/v1/chat/sessions/${activeId}`),
    enabled: !!activeId,
    refetchInterval: 2000,
  })

  const create = useMutation({
    mutationFn: () => request<ChatSession>('/api/v1/chat/sessions', { method: 'POST', body: '{}' }),
    onSuccess: (s) => {
      setActiveId(s.id)
      void qc.invalidateQueries({ queryKey: ['chat-sessions'] })
    },
  })

  const send = useMutation({
    mutationFn: () =>
      request<ChatSession>(`/api/v1/chat/sessions/${activeId}/messages`, {
        method: 'POST',
        body: JSON.stringify({
          text,
          preferProvider: provider || undefined,
          preferModel: model || undefined,
          skillIds: ['coding'],
        }),
      }),
    onSuccess: () => {
      setText('')
      setError(null)
      void qc.invalidateQueries({ queryKey: ['chat-session', activeId] })
      void qc.invalidateQueries({ queryKey: ['chat-sessions'] })
    },
    onError: (e: Error) => setError(e.message),
  })

  const sess = active.data
  const providers = fleet.data ?? []
  const models = providers.find((p) => p.id === provider)?.models ?? []

  return (
    <div className="page">
      <h1 className="page-title">Chat</h1>
      <p className="page-sub">
        Multi-turn chat host — each message becomes an agent-loop mission with tools (fs, shell, web,
        MCP, memory).
      </p>
      <div className="mt-4 grid grid-cols-1 md:grid-cols-[220px_1fr] gap-4 min-h-[60vh]">
        <aside className="card p-3 space-y-2">
          <button type="button" className="btn btn-primary w-full" onClick={() => create.mutate()}>
            New chat
          </button>
          <ul className="space-y-1 max-h-[50vh] overflow-y-auto">
            {(sessions.data ?? []).map((s) => (
              <li key={s.id}>
                <button
                  type="button"
                  className={`w-full text-left px-2 py-1.5 rounded text-sm ${
                    activeId === s.id ? 'bg-[var(--accent-dim)] text-[var(--cyan-100)]' : 'hover:bg-[var(--bg-2)]'
                  }`}
                  onClick={() => setActiveId(s.id)}
                >
                  {s.title}
                </button>
              </li>
            ))}
          </ul>
        </aside>
        <section className="card flex flex-col min-h-[60vh]">
          {!activeId ? (
            <div className="p-6 text-sm text-[var(--ink-2)]">Create or select a chat session.</div>
          ) : (
            <>
              <div className="flex-1 overflow-y-auto p-4 space-y-3">
                {(sess?.messages ?? []).map((m, i) => (
                  <div
                    key={i}
                    className={`rounded-[var(--radius-control)] px-3 py-2 text-sm whitespace-pre-wrap ${
                      m.role === 'user'
                        ? 'bg-[var(--bg-2)] ml-8'
                        : 'bg-[var(--accent-dim)] border border-[rgba(0,191,255,0.25)] mr-8'
                    }`}
                  >
                    <div className="text-[0.65rem] uppercase tracking-wide text-[var(--ink-2)] mb-1">
                      {m.role}
                      {m.missionId ? ` · ${m.missionId}` : ''}
                    </div>
                    {m.content}
                  </div>
                ))}
              </div>
              <div className="border-t border-[var(--line)] p-3 space-y-2">
                <div className="flex flex-wrap gap-2">
                  <select className="input text-sm" value={provider} onChange={(e) => setProvider(e.target.value)}>
                    <option value="">Provider (auto)</option>
                    {providers.map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.name || p.id}
                      </option>
                    ))}
                  </select>
                  <select className="input text-sm" value={model} onChange={(e) => setModel(e.target.value)} disabled={!provider}>
                    <option value="">Model (default)</option>
                    {models.map((m) => (
                      <option key={m.id} value={m.id}>
                        {m.id}
                      </option>
                    ))}
                  </select>
                </div>
                {error && <p className="text-sm text-[var(--fail)]">{error}</p>}
                <div className="flex gap-2">
                  <textarea
                    className="input flex-1 min-h-[72px]"
                    value={text}
                    onChange={(e) => setText(e.target.value)}
                    placeholder="Message the agent…"
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' && !e.shiftKey) {
                        e.preventDefault()
                        if (text.trim() && !send.isPending) send.mutate()
                      }
                    }}
                  />
                  <button
                    type="button"
                    className="btn btn-primary self-end"
                    disabled={!text.trim() || send.isPending}
                    onClick={() => send.mutate()}
                  >
                    {send.isPending ? '…' : 'Send'}
                  </button>
                </div>
              </div>
            </>
          )}
        </section>
      </div>
    </div>
  )
}
