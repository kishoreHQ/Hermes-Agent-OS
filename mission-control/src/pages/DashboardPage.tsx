import { Link } from 'react-router-dom'
import { useCredentials, useHealth, useMissions, useRegistry } from '@/api/hooks'
import { stateChip } from '@/lib/state'

export function DashboardPage() {
  const health = useHealth()
  const missions = useMissions()
  const providers = useRegistry('providers')
  const runtimes = useRegistry('runtimes')
  const creds = useCredentials()

  const list = missions.data ?? []
  const succeeded = list.filter((m) => m.state === 'succeeded').length
  const failed = list.filter((m) => m.state === 'failed').length
  const recent = [...list].sort((a, b) => (b.updatedAt ?? '').localeCompare(a.updatedAt ?? '')).slice(0, 6)

  const kpis = [
    { label: 'Missions', value: list.length, tone: 'var(--cyan-100)' },
    { label: 'Succeeded', value: succeeded, tone: 'var(--ok)' },
    { label: 'Failed', value: failed, tone: 'var(--fail)' },
    { label: 'Providers', value: providers.data?.length ?? 0, tone: 'var(--cyan-100)' },
    { label: 'Runtimes', value: runtimes.data?.length ?? 0, tone: 'var(--cyan-100)' },
    { label: 'Cred handles', value: creds.data?.length ?? 0, tone: 'var(--warn)' },
  ]

  return (
    <div className="page">
      <h1 className="page-title">Overview</h1>
      <p className="page-sub">
        Operator surface for Hermes Agent OS. Binds only to Host API — providers and runtimes are
        plugins.
      </p>

      <div className="mt-6 grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
        {kpis.map((k) => (
          <div key={k.label} className="card p-3">
            <div className="section-label !mb-1">{k.label}</div>
            <div className="font-display text-2xl" style={{ color: k.tone }}>
              {k.value}
            </div>
          </div>
        ))}
      </div>

      <div className="mt-6 grid lg:grid-cols-2 gap-4">
        <section className="card p-4">
          <div className="section-label">Kernel</div>
          {health.isError && (
            <p className="text-sm text-[var(--fail)]">
              Cannot reach Host API. Run <code className="font-mono">make serve</code> on :8080.
            </p>
          )}
          {health.data && (
            <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
              <dt className="text-[var(--ink-2)]">status</dt>
              <dd className="font-mono">{health.data.status}</dd>
              <dt className="text-[var(--ink-2)]">product</dt>
              <dd className="font-mono">{health.data.product ?? 'Hermes'}</dd>
              <dt className="text-[var(--ink-2)]">version</dt>
              <dd className="font-mono">{health.data.version}</dd>
              <dt className="text-[var(--ink-2)]">profile</dt>
              <dd className="font-mono">{health.data.profile}</dd>
              <dt className="text-[var(--ink-2)]">journal seq</dt>
              <dd className="font-mono">{health.data.seq}</dd>
            </dl>
          )}
        </section>

        <section className="card p-4">
          <div className="flex items-center justify-between mb-2">
            <div className="section-label !mb-0">Recent missions</div>
            <Link to="/missions" className="text-xs text-[var(--cyan-200)] hover:underline">
              View all
            </Link>
          </div>
          {recent.length === 0 && (
            <p className="text-sm text-[var(--ink-2)]">No missions yet. Launch one from Missions.</p>
          )}
          <ul className="space-y-2">
            {recent.map((m) => (
              <li key={m.id}>
                <Link
                  to={`/missions/${m.id}`}
                  className="flex items-center justify-between gap-2 rounded-[var(--radius-control)] px-2 py-1.5 hover:bg-[var(--bg-2)]"
                >
                  <span className="truncate text-sm">{m.name || m.goal}</span>
                  <span className={stateChip(m.state)}>{m.state}</span>
                </Link>
              </li>
            ))}
          </ul>
        </section>
      </div>
    </div>
  )
}
