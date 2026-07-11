import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

type Job = {
  id: string
  name: string
  goal: string
  intervalSec: number
  enabled: boolean
  lastMissionId?: string
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: { Accept: 'application/json', ...(init?.body ? { 'Content-Type': 'application/json' } : {}) },
  })
  const body = await res.json()
  if (!res.ok || body.error) throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  return body.data as T
}

export function JobsPage() {
  const qc = useQueryClient()
  const jobs = useQuery({ queryKey: ['jobs'], queryFn: () => request<Job[]>('/api/v1/jobs'), refetchInterval: 5000 })
  const [name, setName] = useState('Hourly check')
  const [goal, setGoal] = useState('Summarize workspace status in one paragraph.')
  const [intervalSec, setIntervalSec] = useState(3600)
  const create = useMutation({
    mutationFn: () =>
      request('/api/v1/jobs', {
        method: 'POST',
        body: JSON.stringify({ name, goal, intervalSec, enabled: true }),
      }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['jobs'] }),
  })
  const run = useMutation({
    mutationFn: (id: string) => request(`/api/v1/jobs/${id}/run`, { method: 'POST' }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['jobs'] }),
  })
  return (
    <div className="page">
      <h1 className="page-title">Jobs</h1>
      <p className="page-sub">Scheduled agent missions (interval seconds).</p>
      <section className="card p-4 mt-4 max-w-xl space-y-2">
        <input className="input" value={name} onChange={(e) => setName(e.target.value)} />
        <textarea className="input min-h-[60px]" value={goal} onChange={(e) => setGoal(e.target.value)} />
        <input
          className="input"
          type="number"
          value={intervalSec}
          onChange={(e) => setIntervalSec(Number(e.target.value))}
        />
        <button type="button" className="btn btn-primary" onClick={() => create.mutate()}>
          Create job
        </button>
      </section>
      <ul className="card mt-6 divide-y divide-[var(--line)]">
        {(jobs.data ?? []).map((j) => (
          <li key={j.id} className="px-4 py-3 flex justify-between gap-2">
            <div>
              <div className="font-medium">{j.name}</div>
              <div className="text-xs text-[var(--ink-2)]">
                every {j.intervalSec}s · {j.enabled ? 'enabled' : 'disabled'}
                {j.lastMissionId ? ` · last ${j.lastMissionId}` : ''}
              </div>
            </div>
            <button type="button" className="btn" onClick={() => run.mutate(j.id)}>
              Run now
            </button>
          </li>
        ))}
      </ul>
    </div>
  )
}
