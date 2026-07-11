import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import { useCreateMission, useMissions } from '@/api/hooks'
import { stateChip } from '@/lib/state'

export function MissionsPage() {
  const missions = useMissions()
  const create = useCreateMission()
  const fleet = useQuery({ queryKey: ['provider-models'], queryFn: api.listProviderModels, refetchInterval: 15000 })
  const [goal, setGoal] = useState('')
  const [caps, setCaps] = useState('coding, tools')
  const [provider, setProvider] = useState('')
  const [model, setModel] = useState('')
  const [failover, setFailover] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const providers = fleet.data ?? []
  const selected = providers.find((p) => p.id === provider)
  const models = selected?.models ?? []

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    const requiredCapabilities = caps
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)
    try {
      await create.mutateAsync({
        goal,
        requiredCapabilities,
        preferProvider: provider || undefined,
        preferModel: model || undefined,
        failover,
      })
      setGoal('')
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    }
  }

  const list = [...(missions.data ?? [])].sort((a, b) =>
    (b.createdAt ?? '').localeCompare(a.createdAt ?? ''),
  )

  return (
    <div className="page">
      <h1 className="page-title">Missions</h1>
      <p className="page-sub">
        Capability-routed work units. Never pass model names as capabilities.
      </p>

      <form onSubmit={onSubmit} className="card p-4 mt-5 space-y-3 max-w-2xl">
        <div className="section-label">Launch mission</div>
        <label className="block text-sm">
          <span className="text-[var(--ink-2)] text-xs">Goal</span>
          <input
            className="input mt-1"
            value={goal}
            onChange={(e) => setGoal(e.target.value)}
            placeholder="e.g. prove H3 host binding"
            required
          />
        </label>
        <label className="block text-sm">
          <span className="text-[var(--ink-2)] text-xs">Required capabilities (comma-separated)</span>
          <input
            className="input mt-1 font-mono"
            value={caps}
            onChange={(e) => setCaps(e.target.value)}
            placeholder="coding, tools"
            required
          />
        </label>
        <div className="grid sm:grid-cols-2 gap-3">
          <label className="block text-sm">
            <span className="text-[var(--ink-2)] text-xs">Prefer provider (optional)</span>
            <select
              className="input mt-1"
              value={provider}
              onChange={(e) => {
                setProvider(e.target.value)
                setModel('')
              }}
            >
              <option value="">Auto (capability + tier)</option>
              {providers.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name || p.id} {p.health?.startsWith('healthy') ? '' : '(unhealthy)'}
                </option>
              ))}
            </select>
          </label>
          <label className="block text-sm">
            <span className="text-[var(--ink-2)] text-xs">Prefer model (optional, auto-discovered)</span>
            <select
              className="input mt-1 font-mono"
              value={model}
              onChange={(e) => setModel(e.target.value)}
              disabled={!provider}
            >
              <option value="">Default for provider</option>
              {models.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.id}
                </option>
              ))}
            </select>
          </label>
        </div>
        <label className="flex items-center gap-2 text-sm text-[var(--ink-1)]">
          <input
            type="checkbox"
            checked={failover}
            onChange={(e) => setFailover(e.target.checked)}
          />
          Failover to next healthy provider if complete fails
        </label>
        {error && <p className="text-sm text-[var(--fail)]">{error}</p>}
        <button type="submit" className="btn btn-primary" disabled={create.isPending || !goal}>
          {create.isPending ? 'Submitting…' : 'Submit mission'}
        </button>
      </form>

      <div className="mt-6 card overflow-hidden">
        <div className="px-4 py-3 border-b border-[var(--line)] section-label !mb-0">
          Fleet missions ({list.length})
        </div>
        {missions.isLoading && <p className="p-4 text-sm text-[var(--ink-2)]">Loading…</p>}
        {list.length === 0 && !missions.isLoading && (
          <p className="p-4 text-sm text-[var(--ink-2)]">No missions.</p>
        )}
        <ul className="divide-y divide-[var(--line)]">
          {list.map((m) => (
            <li key={m.id}>
              <Link
                to={`/missions/${m.id}`}
                className="flex flex-col sm:flex-row sm:items-center gap-2 px-4 py-3 hover:bg-[var(--bg-2)]/60"
              >
                <div className="flex-1 min-w-0">
                  <div className="font-medium truncate">{m.name || m.goal}</div>
                  <div className="font-mono text-xs text-[var(--ink-2)] truncate mt-0.5">
                    {m.id}
                    {m.providerId ? ` · ${m.providerId}` : ''}
                    {m.runtimeId ? ` → ${m.runtimeId}` : ''}
                  </div>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  {(m.requiredCapabilities ?? []).slice(0, 3).map((c) => (
                    <span key={c} className="chip">
                      {c}
                    </span>
                  ))}
                  <span className={stateChip(m.state)}>{m.state}</span>
                </div>
              </Link>
            </li>
          ))}
        </ul>
      </div>
    </div>
  )
}
