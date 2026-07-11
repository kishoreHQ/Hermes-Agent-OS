import { useRegistry } from '@/api/hooks'
import type { PluginItem } from '@/api/types'

function PluginTable({ title, items }: { title: string; items: PluginItem[] }) {
  return (
    <section className="card overflow-hidden">
      <div className="px-4 py-3 border-b border-[var(--line)] section-label !mb-0">
        {title} ({items.length})
      </div>
      {items.length === 0 ? (
        <p className="p-4 text-sm text-[var(--ink-2)]">None registered.</p>
      ) : (
        <ul className="divide-y divide-[var(--line)]">
          {items.map((p) => (
            <li key={p.id} className="px-4 py-3 flex flex-col sm:flex-row sm:items-center gap-2">
              <div className="flex-1 min-w-0">
                <div className="font-medium">{p.name || p.id}</div>
                <div className="font-mono text-xs text-[var(--ink-2)] truncate">{p.id}</div>
              </div>
              <div className="flex flex-wrap gap-1">
                <span className="chip">{p.kind}</span>
                {p.version && <span className="chip">v{p.version}</span>}
                {p.labels?.['hermes.driver'] && (
                  <span className="chip chip-live">{p.labels['hermes.driver']}</span>
                )}
                {typeof p.spec?.costTier === 'string' && (
                  <span className="chip">{String(p.spec.costTier)}</span>
                )}
                {p.spec?.local === true && <span className="chip chip-ok">local</span>}
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}

export function FleetPage() {
  const providers = useRegistry('providers')
  const runtimes = useRegistry('runtimes')
  const tools = useRegistry('tools')

  return (
    <div className="page">
      <h1 className="page-title">Fleet</h1>
      <p className="page-sub">
        Plugin registry — providers supply models; runtimes execute work. Kernel has zero vendor
        names.
      </p>
      <div className="mt-5 space-y-4">
        <PluginTable title="Providers" items={providers.data ?? []} />
        <PluginTable title="Runtimes" items={runtimes.data ?? []} />
        <PluginTable title="Tools" items={tools.data ?? []} />
      </div>
    </div>
  )
}
