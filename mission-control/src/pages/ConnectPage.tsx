import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/api/client'
import { useConnections, useProbe } from '@/api/hooks'

export function ConnectPage() {
  const probe = useProbe()
  const conns = useConnections()
  const qc = useQueryClient()
  const reg = useMutation({
    mutationFn: api.registerConnection,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['connections'] })
      void qc.invalidateQueries({ queryKey: ['probe'] })
    },
  })

  const candidates = (probe.data ?? []) as Array<Record<string, unknown>>
  const connections = (conns.data ?? []) as Array<Record<string, unknown>>

  return (
    <div className="page">
      <h1 className="page-title">Connect</h1>
      <p className="page-sub">
        Probe plugin fleet and local OpenAI-compatible endpoints. Register connections without
        hardcoding vendors in the kernel.
      </p>

      <section className="card mt-5 overflow-hidden">
        <div className="px-4 py-3 border-b border-[var(--line)] section-label !mb-0">
          Probe candidates ({candidates.length})
        </div>
        <ul className="divide-y divide-[var(--line)]">
          {candidates.map((c) => (
            <li key={String(c.id)} className="px-4 py-3 flex flex-wrap items-center gap-2 justify-between">
              <div className="min-w-0">
                <div className="font-medium">{String(c.name || c.id)}</div>
                <div className="font-mono text-xs text-[var(--ink-2)]">
                  {String(c.id)} · {String(c.kind)} · {String(c.detail || '')}
                </div>
              </div>
              <div className="flex items-center gap-2">
                <span className={c.detected ? 'chip chip-ok' : 'chip chip-fail'}>
                  {c.detected ? 'detected' : 'missing'}
                </span>
                <button
                  type="button"
                  className="btn btn-primary"
                  disabled={reg.isPending}
                  onClick={() =>
                    reg.mutate({
                      pluginId: String(c.id),
                      kind: String(c.kind),
                      name: String(c.name || c.id),
                    })
                  }
                >
                  Register
                </button>
              </div>
            </li>
          ))}
        </ul>
      </section>

      <section className="card mt-4 overflow-hidden">
        <div className="px-4 py-3 border-b border-[var(--line)] section-label !mb-0">
          Registered ({connections.length})
        </div>
        <ul className="divide-y divide-[var(--line)]">
          {connections.map((c) => (
            <li key={String(c.id)} className="px-4 py-3">
              <div className="font-medium">{String(c.name)}</div>
              <div className="font-mono text-xs text-[var(--ink-2)]">
                {String(c.pluginId)} · {String(c.status)}
              </div>
            </li>
          ))}
          {connections.length === 0 && (
            <li className="p-4 text-sm text-[var(--ink-2)]">No connections yet.</li>
          )}
        </ul>
      </section>
    </div>
  )
}
