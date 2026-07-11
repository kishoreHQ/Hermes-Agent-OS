import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/api/client'
import { useRoutines } from '@/api/hooks'

export function RoutinesPage() {
  const routines = useRoutines()
  const qc = useQueryClient()
  const fire = useMutation({
    mutationFn: (id: string) => api.fireRoutine(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['routines'] })
      void qc.invalidateQueries({ queryKey: ['missions'] })
    },
  })

  const list = (routines.data ?? []) as Array<Record<string, unknown>>

  return (
    <div className="page">
      <h1 className="page-title">Routines</h1>
      <p className="page-sub">Scheduled mission templates. Fire runs a capability-routed mission now.</p>
      <ul className="card mt-5 divide-y divide-[var(--line)]">
        {list.map((r) => (
          <li key={String(r.id)} className="px-4 py-3 flex flex-wrap items-center justify-between gap-2">
            <div>
              <div className="font-medium">{String(r.name)}</div>
              <div className="font-mono text-xs text-[var(--ink-2)]">
                {String(r.schedule)} · last {String(r.lastStatus || '—')} · {String(r.lastMissionId || '')}
              </div>
              <div className="text-sm text-[var(--ink-1)] mt-1">{String(r.prompt || '')}</div>
            </div>
            <button
              type="button"
              className="btn btn-primary"
              disabled={!!r.paused || fire.isPending}
              onClick={() => fire.mutate(String(r.id))}
            >
              Fire
            </button>
          </li>
        ))}
      </ul>
    </div>
  )
}
