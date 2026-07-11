import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

type Approval = {
  id: string
  missionId?: string
  toolId?: string
  reason: string
  status: string
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

export function ApprovalsPage() {
  const qc = useQueryClient()
  const list = useQuery({
    queryKey: ['approvals'],
    queryFn: () => request<Approval[]>('/api/v1/approvals?status=pending'),
    refetchInterval: 3000,
  })
  const resolve = useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: string }) =>
      request(`/api/v1/approvals/${id}/${decision}`, { method: 'POST' }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['approvals'] }),
  })
  return (
    <div className="page">
      <h1 className="page-title">Approvals</h1>
      <p className="page-sub">HITL gate for dangerous tools in assist mode (shell, fs.write, http).</p>
      <ul className="card mt-6 divide-y divide-[var(--line)]">
        {(list.data ?? []).map((a) => (
          <li key={a.id} className="px-4 py-3 flex justify-between gap-3">
            <div>
              <div className="font-medium">
                {a.toolId} · {a.id}
              </div>
              <div className="text-xs text-[var(--ink-2)]">
                {a.reason} {a.missionId ? `· ${a.missionId}` : ''}
              </div>
            </div>
            <div className="flex gap-2">
              <button type="button" className="btn btn-primary" onClick={() => resolve.mutate({ id: a.id, decision: 'approve' })}>
                Approve
              </button>
              <button type="button" className="btn btn-danger" onClick={() => resolve.mutate({ id: a.id, decision: 'deny' })}>
                Deny
              </button>
            </div>
          </li>
        ))}
        {(list.data ?? []).length === 0 && <li className="p-4 text-sm text-[var(--ink-2)]">No pending approvals.</li>}
      </ul>
    </div>
  )
}
