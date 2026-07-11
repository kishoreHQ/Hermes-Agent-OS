import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'

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

export function ResearchPage() {
  const fleet = useQuery({ queryKey: ['provider-models'], queryFn: api.listProviderModels })
  const [topic, setTopic] = useState('')
  const [provider, setProvider] = useState('')
  const [model, setModel] = useState('')
  const [result, setResult] = useState<{ output?: string; id?: string; state?: string } | null>(null)
  const [error, setError] = useState<string | null>(null)

  const run = useMutation({
    mutationFn: () =>
      request<{ id: string; output: string; state: string }>('/api/v1/research', {
        method: 'POST',
        body: JSON.stringify({
          topic,
          preferProvider: provider || undefined,
          preferModel: model || undefined,
        }),
      }),
    onSuccess: (m) => {
      setResult(m)
      setError(null)
    },
    onError: (e: Error) => setError(e.message),
  })

  const providers = fleet.data ?? []
  const models = providers.find((p) => p.id === provider)?.models ?? []

  return (
    <div className="page">
      <h1 className="page-title">Deep Research</h1>
      <p className="page-sub">
        Multi-step research via agent loop + research skill + web.search / web.fetch tools.
      </p>
      <section className="card p-4 mt-5 max-w-2xl space-y-3">
        <textarea
          className="input min-h-[80px]"
          placeholder="Research topic…"
          value={topic}
          onChange={(e) => setTopic(e.target.value)}
        />
        <div className="flex flex-wrap gap-2">
          <select className="input text-sm" value={provider} onChange={(e) => setProvider(e.target.value)}>
            <option value="">Provider (auto)</option>
            {providers.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name || p.id}
              </option>
            ))}
          </select>
          <select className="input text-sm" value={model} onChange={(e) => setModel(e.target.value)}>
            <option value="">Model</option>
            {models.map((m) => (
              <option key={m.id} value={m.id}>
                {m.id}
              </option>
            ))}
          </select>
        </div>
        {error && <p className="text-sm text-[var(--fail)]">{error}</p>}
        <button
          type="button"
          className="btn btn-primary"
          disabled={!topic.trim() || run.isPending}
          onClick={() => run.mutate()}
        >
          {run.isPending ? 'Researching…' : 'Run research'}
        </button>
      </section>
      {result && (
        <section className="card p-4 mt-6 max-w-3xl">
          <div className="section-label">
            Result · {result.id} · {result.state}
          </div>
          <pre className="text-sm whitespace-pre-wrap mt-2">{result.output}</pre>
        </section>
      )}
    </div>
  )
}
