import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/api/client'
import { useBoards } from '@/api/hooks'

const cols = ['backlog', 'queued', 'in_progress', 'review', 'done'] as const

export function BoardPage() {
  const boards = useBoards()
  const qc = useQueryClient()
  const [title, setTitle] = useState('')
  const create = useMutation({
    mutationFn: () => api.createTask({ title, column: 'backlog', capabilities: ['coding'] }),
    onSuccess: () => {
      setTitle('')
      void qc.invalidateQueries({ queryKey: ['boards'] })
    },
  })
  const claim = useMutation({
    mutationFn: (id: string) => api.claimTask(id),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['boards'] }),
  })
  const move = useMutation({
    mutationFn: ({ id, column }: { id: string; column: string }) => api.moveTask(id, column),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['boards'] }),
  })

  const board = ((boards.data ?? [])[0] ?? null) as Record<string, unknown> | null
  const tasks = ((board?.tasks as Array<Record<string, unknown>>) ?? []) as Array<
    Record<string, unknown>
  >

  return (
    <div className="page">
      <h1 className="page-title">Board</h1>
      <p className="page-sub">Kanban for deck work — claim and advance columns.</p>

      <form
        className="mt-4 flex gap-2 max-w-xl"
        onSubmit={(e) => {
          e.preventDefault()
          if (title) create.mutate()
        }}
      >
        <input
          className="input"
          placeholder="New task title"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
        <button type="submit" className="btn btn-primary" disabled={!title}>
          Add
        </button>
      </form>

      <div className="mt-5 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3">
        {cols.map((col) => (
          <div key={col} className="card p-2 min-h-[12rem]">
            <div className="section-label !mb-2 px-1">{col.replace('_', ' ')}</div>
            <ul className="space-y-2">
              {tasks
                .filter((t) => t.column === col)
                .map((t) => (
                  <li key={String(t.id)} className="rounded-[var(--radius-control)] border border-[var(--line)] bg-[var(--bg-0)] p-2 text-sm">
                    <div className="font-medium">{String(t.title)}</div>
                    {t.assignee ? (
                      <div className="text-xs text-[var(--ink-2)]">{String(t.assignee)}</div>
                    ) : null}
                    <div className="mt-2 flex flex-wrap gap-1">
                      <button type="button" className="btn" onClick={() => claim.mutate(String(t.id))}>
                        Claim
                      </button>
                      {col !== 'done' && (
                        <button
                          type="button"
                          className="btn"
                          onClick={() =>
                            move.mutate({
                              id: String(t.id),
                              column: cols[Math.min(cols.indexOf(col) + 1, cols.length - 1)],
                            })
                          }
                        >
                          Advance
                        </button>
                      )}
                    </div>
                  </li>
                ))}
            </ul>
          </div>
        ))}
      </div>
    </div>
  )
}
