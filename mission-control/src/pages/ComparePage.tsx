import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: { Accept: 'application/json', ...(init?.body ? { 'Content-Type': 'application/json' } : {}) },
  })
  const body = await res.json()
  if (!res.ok || body.error) throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  return body.data as T
}

type Arm = { providerId: string; missionId?: string; state: string; output?: string; error?: string }

export function ComparePage() {
  const fleet = useQuery({ queryKey: ['provider-models'], queryFn: api.listProviderModels })
  const [goal, setGoal] = useState('Say hello in one sentence.')
  const [selected, setSelected] = useState<string[]>([])
  const [results, setResults] = useState<Arm[] | null>(null)
  const run = useMutation({
    mutationFn: () =>
      request<Arm[]>('/api/v1/compare', {
        method: 'POST',
        body: JSON.stringify({ goal, providers: selected }),
      }),
    onSuccess: setResults,
  })
  const providers = fleet.data ?? []
  return (
    <div className="page">
      <h1 className="page-title">Compare</h1>
      <p className="page-sub">Run the same goal on multiple providers side-by-side (no failover).</p>
      <section className="card p-4 mt-4 max-w-2xl space-y-3">
        <textarea className="input min-h-[80px]" value={goal} onChange={(e) => setGoal(e.target.value)} />
        <div className="flex flex-wrap gap-2">
          {providers.map((p) => (
            <label key={p.id} className="chip cursor-pointer flex items-center gap-1">
              <input
                type="checkbox"
                checked={selected.includes(p.id)}
                onChange={(e) =>
                  setSelected((s) => (e.target.checked ? [...s, p.id] : s.filter((x) => x !== p.id)))
                }
              />
              {p.name || p.id}
            </label>
          ))}
        </div>
        <button
          type="button"
          className="btn btn-primary"
          disabled={selected.length < 1 || run.isPending}
          onClick={() => run.mutate()}
        >
          {run.isPending ? 'Running…' : 'Compare'}
        </button>
        {run.error && <p className="text-sm text-[var(--fail)]">{(run.error as Error).message}</p>}
      </section>
      {results && (
        <div className="mt-6 grid md:grid-cols-2 gap-4">
          {results.map((r) => (
            <div key={r.providerId} className="card p-4">
              <div className="font-medium font-mono text-sm">{r.providerId}</div>
              <div className="chip mt-1">{r.state}</div>
              {r.error && <p className="text-[var(--fail)] text-sm mt-2">{r.error}</p>}
              <pre className="text-sm mt-2 whitespace-pre-wrap">{r.output}</pre>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
