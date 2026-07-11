import { useState } from 'react'
import { useMemory } from '@/api/hooks'

export function MemoryPage() {
  const [q, setQ] = useState('')
  const memory = useMemory(q)

  return (
    <div className="page">
      <h1 className="page-title">Memory</h1>
      <p className="page-sub">
        Unified Hermes memory (INV-06). Shared across runtimes — trust-labeled, not vendor-owned.
      </p>
      <div className="mt-4 max-w-xl">
        <input
          className="input"
          placeholder="Search content…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </div>
      <div className="mt-5 card overflow-hidden">
        <div className="px-4 py-3 border-b border-[var(--line)] section-label !mb-0">
          Entries ({memory.data?.length ?? 0})
        </div>
        <ul className="divide-y divide-[var(--line)]">
          {(memory.data ?? []).map((e) => (
            <li key={e.id} className="px-4 py-3">
              <div className="flex flex-wrap items-center gap-2 mb-1">
                <span className="font-mono text-xs text-[var(--ink-2)]">{e.id}</span>
                {e.kind && <span className="chip">{e.kind}</span>}
                {e.trust && <span className="chip chip-live">{e.trust}</span>}
                {e.missionId && (
                  <span className="font-mono text-[0.65rem] text-[var(--ink-2)]">{e.missionId}</span>
                )}
              </div>
              <p className="text-sm whitespace-pre-wrap">{e.content}</p>
            </li>
          ))}
          {(memory.data?.length ?? 0) === 0 && (
            <li className="p-4 text-sm text-[var(--ink-2)]">No memory entries yet. Run a mission.</li>
          )}
        </ul>
      </div>
    </div>
  )
}
