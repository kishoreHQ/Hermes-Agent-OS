import { useEvents } from '@/api/hooks'

export function EventsPage() {
  const events = useEvents(0)
  const list = [...(events.data ?? [])].sort((a, b) => b.seq - a.seq)

  return (
    <div className="page">
      <h1 className="page-title">Events</h1>
      <p className="page-sub">
        Global journal with monotonic <code className="font-mono">seq</code> (INV-10). JSON
        catch-up and WebSocket live path on <code className="font-mono">/api/v1/events</code>.
      </p>
      <div className="mt-5 card overflow-hidden">
        <div className="px-4 py-3 border-b border-[var(--line)] section-label !mb-0">
          Journal ({list.length})
        </div>
        <ol className="divide-y divide-[var(--line)] max-h-[70vh] overflow-auto">
          {list.map((e) => (
            <li key={`${e.seq}-${e.type}`} className="px-4 py-3">
              <div className="flex flex-wrap items-center gap-2 text-xs">
                <span className="font-mono text-[var(--cyan-200)]">#{e.seq}</span>
                <span className="font-mono text-[var(--ink-0)]">{e.type}</span>
                {e.missionId && (
                  <span className="font-mono text-[var(--ink-2)]">{e.missionId}</span>
                )}
                {e.ts && <span className="text-[var(--ink-2)]">{e.ts}</span>}
              </div>
              {e.data && (
                <pre className="mt-1 font-mono text-[0.7rem] text-[var(--ink-1)] whitespace-pre-wrap">
                  {JSON.stringify(e.data, null, 2)}
                </pre>
              )}
            </li>
          ))}
          {list.length === 0 && (
            <li className="p-4 text-sm text-[var(--ink-2)]">No events. Submit a mission.</li>
          )}
        </ol>
      </div>
    </div>
  )
}
