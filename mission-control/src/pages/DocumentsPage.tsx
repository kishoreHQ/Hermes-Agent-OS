import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

type Doc = { id: string; title: string; body: string }

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: { Accept: 'application/json', ...(init?.body ? { 'Content-Type': 'application/json' } : {}) },
  })
  const body = await res.json()
  if (!res.ok || body.error) throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  return body.data as T
}

export function DocumentsPage() {
  const qc = useQueryClient()
  const docs = useQuery({ queryKey: ['documents'], queryFn: () => request<Doc[]>('/api/v1/documents') })
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [active, setActive] = useState<Doc | null>(null)
  const save = useMutation({
    mutationFn: () =>
      request<Doc>('/api/v1/documents', {
        method: 'POST',
        body: JSON.stringify({ id: active?.id, title, body }),
      }),
    onSuccess: (d) => {
      setActive(d)
      void qc.invalidateQueries({ queryKey: ['documents'] })
    },
  })
  return (
    <div className="page">
      <h1 className="page-title">Documents</h1>
      <p className="page-sub">Markdown docs (docs.* tools for agents). Separate from platform /api/v1/docs generate.</p>
      <div className="mt-4 grid md:grid-cols-[220px_1fr] gap-4">
        <aside className="card p-2 space-y-1">
          <button
            type="button"
            className="btn w-full"
            onClick={() => {
              setActive(null)
              setTitle('')
              setBody('')
            }}
          >
            New
          </button>
          {(docs.data ?? []).map((d) => (
            <button
              key={d.id}
              type="button"
              className="block w-full text-left px-2 py-1 text-sm hover:bg-[var(--bg-2)] rounded"
              onClick={() => {
                setActive(d)
                setTitle(d.title)
                setBody(d.body)
              }}
            >
              {d.title}
            </button>
          ))}
        </aside>
        <section className="card p-4 space-y-2">
          <input className="input" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Title" />
          <textarea className="input min-h-[240px] font-mono text-sm" value={body} onChange={(e) => setBody(e.target.value)} />
          <button type="button" className="btn btn-primary" disabled={!title} onClick={() => save.mutate()}>
            Save
          </button>
        </section>
      </div>
    </div>
  )
}
