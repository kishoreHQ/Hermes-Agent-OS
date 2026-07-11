import { NavLink, Outlet } from 'react-router-dom'
import { useHealth, useMissions } from '@/api/hooks'
import { StatusDot } from './StatusDot'
import { cn } from '@/lib/cn'

const nav = [
  { to: '/', label: 'Overview', end: true },
  { to: '/missions', label: 'Missions' },
  { to: '/connect', label: 'Connect' },
  { to: '/sessions', label: 'Sessions' },
  { to: '/board', label: 'Board' },
  { to: '/routines', label: 'Routines' },
  { to: '/fleet', label: 'Fleet' },
  { to: '/tools', label: 'Tools' },
  { to: '/memory', label: 'Memory' },
  { to: '/events', label: 'Events' },
  { to: '/credentials', label: 'Credentials' },
]

export function AppShell() {
  const health = useHealth()
  const missions = useMissions()
  const ok = health.data?.status === 'ok'
  const count = missions.data?.length ?? 0

  return (
    <div className="min-h-screen flex flex-col md:flex-row">
      <aside className="md:w-56 shrink-0 border-b md:border-b-0 md:border-r border-[var(--line)] bg-[var(--bg-1)]/80 backdrop-blur">
        <div className="px-4 py-4 border-b border-[var(--line)]">
          <div className="font-display text-lg tracking-tight text-[var(--cyan-100)]">HERMES</div>
          <div className="text-[0.65rem] uppercase tracking-[0.14em] text-[var(--ink-2)] mt-0.5">
            Mission Control
          </div>
        </div>
        <nav className="flex md:flex-col gap-1 p-2 overflow-x-auto">
          {nav.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                cn(
                  'px-3 py-2 rounded-[var(--radius-control)] text-sm whitespace-nowrap transition-colors',
                  isActive
                    ? 'bg-[var(--accent-dim)] text-[var(--cyan-100)] border border-[rgba(0,191,255,0.35)]'
                    : 'text-[var(--ink-1)] hover:text-[var(--ink-0)] hover:bg-[var(--bg-2)]',
                )
              }
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="hidden md:block px-4 py-4 mt-auto border-t border-[var(--line)] space-y-2 text-xs">
          <StatusDot ok={!!ok} label={ok ? 'kernel online' : 'kernel offline'} />
          <div className="font-mono text-[var(--ink-2)]">missions {count}</div>
          <div className="font-mono text-[var(--ink-2)] truncate" title={health.data?.version}>
            {health.data?.version ?? '—'}
          </div>
          <div className="text-[var(--ink-2)] leading-snug">
            Host-neutral · plugins only · no vendor lock-in
          </div>
        </div>
      </aside>

      <main className="flex-1 min-w-0">
        <header className="sticky top-0 z-10 flex items-center justify-between gap-3 px-4 md:px-6 py-3 border-b border-[var(--line)] bg-[var(--bg-0)]/85 backdrop-blur">
          <div className="text-xs uppercase tracking-[0.12em] text-[var(--ink-2)]">
            AESP-powered · Hermes Agent OS
          </div>
          <div className="flex items-center gap-3">
            <StatusDot ok={!!ok} label={ok ? 'healthy' : 'down'} />
            <span className="font-mono text-xs text-[var(--ink-2)]">
              seq {health.data?.seq ?? 0}
            </span>
          </div>
        </header>
        <Outlet />
      </main>
    </div>
  )
}
