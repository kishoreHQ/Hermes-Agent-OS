import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

type Todo = { id: string; title: string; done: boolean }

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: { Accept: 'application/json', ...(init?.body ? { 'Content-Type': 'application/json' } : {}) },
  })
  const body = await res.json()
  if (!res.ok || body.error) throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  return body.data as T
}

export function TodosPage() {
  const qc = useQueryClient()
  const todos = useQuery({ queryKey: ['todos'], queryFn: () => request<Todo[]>('/api/v1/todos') })
  const [title, setTitle] = useState('')
  const save = useMutation({
    mutationFn: (t: Partial<Todo>) => request('/api/v1/todos', { method: 'POST', body: JSON.stringify(t) }),
    onSuccess: () => {
      setTitle('')
      void qc.invalidateQueries({ queryKey: ['todos'] })
    },
  })
  return (
    <div className="page">
      <h1 className="page-title">Todos</h1>
      <p className="page-sub">Tasks agents can create via todos.write / todos.list.</p>
      <div className="card p-4 mt-4 flex gap-2 max-w-xl">
        <input className="input flex-1" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="New todo" />
        <button type="button" className="btn btn-primary" disabled={!title} onClick={() => save.mutate({ title, done: false })}>
          Add
        </button>
      </div>
      <ul className="card mt-6 divide-y divide-[var(--line)]">
        {(todos.data ?? []).map((t) => (
          <li key={t.id} className="px-4 py-3 flex items-center gap-3">
            <input
              type="checkbox"
              checked={t.done}
              onChange={() => save.mutate({ id: t.id, title: t.title, done: !t.done })}
            />
            <span className={t.done ? 'line-through text-[var(--ink-2)]' : ''}>{t.title}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}
