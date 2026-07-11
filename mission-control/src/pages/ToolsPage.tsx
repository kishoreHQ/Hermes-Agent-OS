import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/api/client'
import { useTools } from '@/api/hooks'

export function ToolsPage() {
  const tools = useTools()
  const qc = useQueryClient()
  const inv = useQuery({
    queryKey: ['tool-invocations'],
    queryFn: api.toolInvocations,
    refetchInterval: 4000,
  })
  const run = useMutation({
    mutationFn: (id: string) =>
      api.invokeTool(id, id === 'echo' ? { text: 'deck-tool-check' } : {}),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['tool-invocations'] })
    },
  })

  const list = (tools.data ?? []) as Array<Record<string, unknown>>
  const log = (inv.data ?? []) as Array<Record<string, unknown>>

  return (
    <div className="page">
      <h1 className="page-title">Tools</h1>
      <p className="page-sub">
        Hermes-defined tools with invocation audit records (AESP INT-TOOLS). Runtimes consume these —
        not the reverse.
      </p>
      <div className="mt-5 grid lg:grid-cols-2 gap-4">
        <section className="card overflow-hidden">
          <div className="px-4 py-3 border-b border-[var(--line)] section-label !mb-0">
            Registry ({list.length})
          </div>
          <ul className="divide-y divide-[var(--line)]">
            {list.map((t) => (
              <li key={String(t.id)} className="px-4 py-3 flex items-center justify-between gap-2">
                <div>
                  <div className="font-medium">{String(t.name || t.id)}</div>
                  <div className="text-xs text-[var(--ink-2)]">{String(t.description || '')}</div>
                </div>
                <button type="button" className="btn" onClick={() => run.mutate(String(t.id))}>
                  Invoke
                </button>
              </li>
            ))}
          </ul>
        </section>
        <section className="card overflow-hidden">
          <div className="px-4 py-3 border-b border-[var(--line)] section-label !mb-0">
            Invocations
          </div>
          <ul className="divide-y divide-[var(--line)] max-h-[24rem] overflow-auto">
            {[...log].reverse().map((i) => (
              <li key={String(i.id)} className="px-4 py-2 text-sm">
                <span className="font-mono text-xs text-[var(--cyan-200)]">{String(i.toolId)}</span>{' '}
                <span className="chip">{String(i.status)}</span>
                <div className="text-[var(--ink-1)] truncate">{String(i.output || i.error || '')}</div>
              </li>
            ))}
          </ul>
        </section>
      </div>
    </div>
  )
}
