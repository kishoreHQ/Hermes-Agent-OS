import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

type Skill = { id: string; name: string; description?: string; body: string; source?: string }

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

export function SkillsPage() {
  const qc = useQueryClient()
  const skills = useQuery({
    queryKey: ['skills'],
    queryFn: () => request<Skill[]>('/api/v1/skills'),
  })
  const [id, setId] = useState('')
  const [name, setName] = useState('')
  const [body, setBody] = useState('')
  const [error, setError] = useState<string | null>(null)

  const save = useMutation({
    mutationFn: () =>
      request<Skill>('/api/v1/skills', {
        method: 'POST',
        body: JSON.stringify({ id, name, body, source: 'ui' }),
      }),
    onSuccess: () => {
      setError(null)
      setId('')
      setName('')
      setBody('')
      void qc.invalidateQueries({ queryKey: ['skills'] })
    },
    onError: (e: Error) => setError(e.message),
  })

  return (
    <div className="page">
      <h1 className="page-title">Skills</h1>
      <p className="page-sub">
        Reusable prompt packs injected into agent-loop missions (label{' '}
        <code className="font-mono text-xs">skills=coding,research</code>).
      </p>

      <section className="card p-4 mt-5 max-w-2xl space-y-3">
        <div className="section-label">Add skill</div>
        <input className="input" placeholder="id (e.g. my-skill)" value={id} onChange={(e) => setId(e.target.value)} />
        <input className="input" placeholder="Display name" value={name} onChange={(e) => setName(e.target.value)} />
        <textarea
          className="input min-h-[120px] font-mono text-sm"
          placeholder="Skill body (markdown instructions)"
          value={body}
          onChange={(e) => setBody(e.target.value)}
        />
        {error && <p className="text-sm text-[var(--fail)]">{error}</p>}
        <button
          type="button"
          className="btn btn-primary"
          disabled={!id || !body || save.isPending}
          onClick={() => save.mutate()}
        >
          Save skill
        </button>
      </section>

      <section className="card mt-6 overflow-hidden">
        <div className="px-4 py-3 border-b border-[var(--line)] section-label !mb-0">
          Skills ({(skills.data ?? []).length})
        </div>
        <ul className="divide-y divide-[var(--line)]">
          {(skills.data ?? []).map((sk) => (
            <li key={sk.id} className="px-4 py-3">
              <div className="font-medium">
                {sk.name} <span className="chip">{sk.source || '—'}</span>
              </div>
              <div className="font-mono text-xs text-[var(--ink-2)]">{sk.id}</div>
              <pre className="text-xs mt-2 text-[var(--ink-1)] whitespace-pre-wrap max-h-32 overflow-y-auto">
                {sk.body}
              </pre>
            </li>
          ))}
        </ul>
      </section>
    </div>
  )
}
