import { useCredentials } from '@/api/hooks'

export function CredentialsPage() {
  const creds = useCredentials()

  return (
    <div className="page">
      <h1 className="page-title">Credentials</h1>
      <p className="page-sub">
        Unified broker (INV-07). Host UI shows <strong>handles only</strong> — secrets never cross
        this boundary.
      </p>
      <div className="mt-5 card overflow-hidden">
        <div className="px-4 py-3 border-b border-[var(--line)] section-label !mb-0">
          Handles ({creds.data?.length ?? 0})
        </div>
        <ul className="divide-y divide-[var(--line)]">
          {(creds.data ?? []).map((c) => (
            <li key={c.handle} className="px-4 py-3 grid sm:grid-cols-2 gap-1 text-sm">
              <div>
                <div className="text-[var(--ink-2)] text-xs">handle</div>
                <div className="font-mono text-xs break-all">{c.handle}</div>
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <div className="text-[var(--ink-2)] text-xs">scope</div>
                  <div className="font-mono text-xs">{c.scope || '—'}</div>
                </div>
                <div>
                  <div className="text-[var(--ink-2)] text-xs">label</div>
                  <div className="font-mono text-xs">{c.label || '—'}</div>
                </div>
              </div>
            </li>
          ))}
          {(creds.data?.length ?? 0) === 0 && (
            <li className="p-4 text-sm text-[var(--ink-2)]">
              No handles yet. Submitting a mission issues a demo handle for the routed provider.
            </li>
          )}
        </ul>
      </div>
    </div>
  )
}
