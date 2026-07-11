import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

type Note = { id: string; title: string; body: string }

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: {
      Accept: 'application/json',
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
    },
  })
  const body = await res.json()
  if (!res.ok || body.error) throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  return body.data as T
}

export function NotesPage() {
  const qc = useQueryClient()
  const notes = useQuery({ queryKey: ['notes'], queryFn: () => request<Note[]>('/api/v1/notes') })
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const save = useMutation({
    mutationFn: () => request('/api/v1/notes', { method: 'POST', body: JSON.stringify({ title, body }) }),
    onSuccess: () => {
      setTitle('')
      setBody('')
      void qc.invalidateQueries({ queryKey: ['notes'] })
    },
  })
  return (
    <div className="page">
      <h1 className="page-title">Notes</h1>
      <p className="page-sub">Workspace notes (also available to agents via notes.* tools).</p>
      <section className="card p-4 mt-4 max-w-xl space-y-2">
        <input className="input" placeholder="Title" value={title} onChange={(e) => setTitle(e.target.value)} />
        <textarea className="input min-h-[100px]" placeholder="Body" value={body} onChange={(e) => setBody(e.target.value)} />
        <button type="button" className="btn btn-primary" disabled={!title || save.isPending} onClick={() => save.mutate()}>
          Save note
        </button>
      </section>
      <ul className="card mt-6 divide-y divide-[var(--line)]">
        {(notes.data ?? []).map((n) => (
          <li key={n.id} className="px-4 py-3">
            <div className="font-medium">{n.title}</div>
            <div className="text-xs font-mono text-[var(--ink-2)]">{n.id}</div>
            <pre className="text-sm mt-1 whitespace-pre-wrap">{n.body}</pre>
          </li>
        ))}
      </ul>
    </div>
  )
}
