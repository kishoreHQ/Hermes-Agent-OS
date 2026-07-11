import { cn } from '@/lib/cn'

export function StatusDot({ ok, label }: { ok: boolean; label?: string }) {
  return (
    <span className="inline-flex items-center gap-1.5 text-xs text-[var(--ink-1)]">
      <span
        className={cn(
          'inline-block h-2 w-2 rounded-full',
          ok ? 'bg-[var(--ok)] shadow-[0_0_8px_var(--ok)]' : 'bg-[var(--fail)]',
        )}
        aria-hidden
      />
      {label}
    </span>
  )
}
