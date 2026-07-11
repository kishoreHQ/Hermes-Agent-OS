import { useState, type FormEvent } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/api/client'
import { useSessions } from '@/api/hooks'

export function SessionsPage() {
  const sessions = useSessions()
  const qc = useQueryClient()
  const [active, setActive] = useState<string | null>(null)
  const [text, setText] = useState('')

  const create = useMutation({
    mutationFn: () => api.createSession({ runtime: 'runtime.example.echo' }),
    onSuccess: (s) => {
      void qc.invalidateQueries({ queryKey: ['sessions'] })
      setActive(String(s.id))
    },
  })
  const send = useMutation({
    mutationFn: () => api.sessionMessage(active!, text),
    onSuccess: () => {
      setText('')
      void qc.invalidateQueries({ queryKey: ['sessions'] })
      void qc.invalidateQueries({ queryKey: ['missions'] })
    },
  })

  const list = (sessions.data ?? []) as Array<Record<string, unknown>>
  const current = list.find((s) => s.id === active) as Record<string, unknown> | undefined
  const messages = (current?.messages as Array<Record<string, unknown>>) ?? []

  return (
    <div className="page">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="page-title">Sessions</h1>
          <p className="page-sub">Live agent sessions — messages become capability-routed missions.</p>
        </div>
        <button type="button" className="btn btn-primary" onClick={() => create.mutate()}>
          New session
        </button>
      </div>

      <div className="mt-5 grid lg:grid-cols-[16rem_1fr] gap-4">
        <ul className="card divide-y divide-[var(--line)] overflow-hidden">
          {list.map((s) => (
            <li key={String(s.id)}>
              <button
                type="button"
                className={`w-full text-left px-3 py-2 text-sm hover:bg-[var(--bg-2)] ${
                  active === s.id ? 'bg-[var(--accent-dim)]' : ''
                }`}
                onClick={() => setActive(String(s.id))}
              >
                <div className="font-mono text-xs truncate">{String(s.id)}</div>
                <div className="text-[var(--ink-2)] text-xs">{String(s.status)}</div>
              </button>
            </li>
          ))}
          {list.length === 0 && (
            <li className="p-3 text-sm text-[var(--ink-2)]">No sessions. Create one.</li>
          )}
        </ul>

        <div className="card flex flex-col min-h-[20rem]">
          <div className="flex-1 p-4 space-y-2 overflow-auto max-h-[28rem]">
            {messages.map((m, i) => (
              <div
                key={i}
                className={`text-sm rounded-[var(--radius-control)] px-3 py-2 ${
                  m.role === 'user' ? 'bg-[var(--bg-2)]' : 'bg-[var(--accent-dim)]'
                }`}
              >
                <div className="text-[0.65rem] uppercase text-[var(--ink-2)]">{String(m.role)}</div>
                <pre className="whitespace-pre-wrap font-sans m-0">{String(m.content)}</pre>
              </div>
            ))}
            {!active && <p className="text-sm text-[var(--ink-2)]">Select or create a session.</p>}
          </div>
          <form
            className="border-t border-[var(--line)] p-3 flex gap-2"
            onSubmit={(e: FormEvent) => {
              e.preventDefault()
              if (active && text) send.mutate()
            }}
          >
            <input
              className="input flex-1"
              placeholder="Message agent…"
              value={text}
              disabled={!active}
              onChange={(e) => setText(e.target.value)}
            />
            <button type="submit" className="btn btn-primary" disabled={!active || !text || send.isPending}>
              Send
            </button>
          </form>
        </div>
      </div>
    </div>
  )
}
