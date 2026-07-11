import { Link, useParams } from 'react-router-dom'
import { useCancelMission, useMission, useReplay } from '@/api/hooks'
import { stateChip } from '@/lib/state'

export function MissionDetailPage() {
  const { id } = useParams()
  const mission = useMission(id)
  const replay = useReplay(id)
  const cancel = useCancelMission()
  const m = mission.data
  const events = replay.data?.events ?? []

  return (
    <div className="page">
      <div className="mb-3">
        <Link to="/missions" className="text-xs text-[var(--cyan-200)] hover:underline">
          ← Missions
        </Link>
      </div>
      {mission.isLoading && <p className="text-[var(--ink-2)]">Loading…</p>}
      {mission.isError && (
        <p className="text-[var(--fail)]">Mission not found or Host API error.</p>
      )}
      {m && (
        <>
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h1 className="page-title">{m.name || m.goal}</h1>
              <p className="page-sub font-mono">{m.id}</p>
            </div>
            <div className="flex items-center gap-2">
              <span className={stateChip(m.state)}>{m.state}</span>
              {m.state !== 'cancelled' && (
                <button
                  type="button"
                  className="btn btn-danger"
                  disabled={cancel.isPending}
                  onClick={() => cancel.mutate({ id: m.id, reason: 'operator-ui' })}
                >
                  Cancel
                </button>
              )}
            </div>
          </div>

          <div className="mt-5 grid lg:grid-cols-2 gap-4">
            <section className="card p-4 space-y-3">
              <div className="section-label">Mission</div>
              <dl className="grid grid-cols-[7rem_1fr] gap-y-2 text-sm">
                <dt className="text-[var(--ink-2)]">goal</dt>
                <dd>{m.goal}</dd>
                <dt className="text-[var(--ink-2)]">capabilities</dt>
                <dd className="flex flex-wrap gap-1">
                  {(m.requiredCapabilities ?? []).map((c) => (
                    <span key={c} className="chip">
                      {c}
                    </span>
                  ))}
                </dd>
                <dt className="text-[var(--ink-2)]">provider</dt>
                <dd className="font-mono text-xs">{m.providerId || '—'}</dd>
                <dt className="text-[var(--ink-2)]">runtime</dt>
                <dd className="font-mono text-xs">{m.runtimeId || '—'}</dd>
                <dt className="text-[var(--ink-2)]">model</dt>
                <dd className="font-mono text-xs">{m.modelId || '—'}</dd>
                <dt className="text-[var(--ink-2)]">route</dt>
                <dd className="font-mono text-xs">{m.routeReason || '—'}</dd>
                <dt className="text-[var(--ink-2)]">cost</dt>
                <dd className="font-mono text-xs">${(m.costUsd ?? 0).toFixed(4)}</dd>
              </dl>
              {m.output && (
                <div>
                  <div className="section-label">Output</div>
                  <pre className="font-mono text-xs whitespace-pre-wrap rounded-[var(--radius-control)] border border-[var(--line)] bg-[var(--bg-0)] p-3 text-[var(--ink-0)]">
                    {m.output}
                  </pre>
                </div>
              )}
            </section>

            <section className="card p-4">
              <div className="section-label">Replay journal</div>
              <p className="text-xs text-[var(--ink-2)] mb-3">
                Monotonic seq events — routing and execution are replayable.
              </p>
              <ol className="space-y-2 max-h-[28rem] overflow-auto pr-1">
                {events.map((e) => (
                  <li
                    key={`${e.seq}-${e.type}`}
                    className="rounded-[var(--radius-control)] border border-[var(--line)] bg-[var(--bg-0)]/60 px-3 py-2"
                  >
                    <div className="flex items-center justify-between gap-2 text-xs">
                      <span className="font-mono text-[var(--cyan-200)]">#{e.seq}</span>
                      <span className="font-mono text-[var(--ink-1)]">{e.type}</span>
                    </div>
                    {e.data && (
                      <pre className="mt-1 font-mono text-[0.65rem] text-[var(--ink-2)] whitespace-pre-wrap overflow-x-auto">
                        {JSON.stringify(e.data, null, 0)}
                      </pre>
                    )}
                  </li>
                ))}
                {events.length === 0 && (
                  <li className="text-sm text-[var(--ink-2)]">No events yet.</li>
                )}
              </ol>
            </section>
          </div>
        </>
      )}
    </div>
  )
}
