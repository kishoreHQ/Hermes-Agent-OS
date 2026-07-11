import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/api/client'
import { cn } from '@/lib/cn'

type Template = {
  id: string
  name: string
  description?: string
  driver: string
  baseUrl: string
  local: boolean
  costTier: string
  defaultModel?: string
  needsApiKey: boolean
  category?: string
  suggestedModels?: string[]
  docsUrl?: string
}

type ProviderConfig = {
  id: string
  name: string
  driver: string
  templateId?: string
  baseUrl?: string
  local: boolean
  costTier: string
  defaultModel?: string
  models?: string[]
  credentialHandle?: string
  managed: boolean
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: {
      Accept: 'application/json',
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...init?.headers,
    },
  })
  const body = await res.json()
  if (!res.ok || body.error) {
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data as T
}

const CATEGORIES = [
  { id: 'all', label: 'All' },
  { id: 'cloud', label: 'Cloud' },
  { id: 'gateway', label: 'Gateway' },
  { id: 'local', label: 'Local' },
  { id: 'custom', label: 'Custom' },
]

export function ProvidersPage() {
  const qc = useQueryClient()
  const templates = useQuery({
    queryKey: ['provider-templates'],
    queryFn: () => request<Template[]>('/api/v1/provider-templates'),
  })
  const configs = useQuery({
    queryKey: ['provider-configs'],
    queryFn: () => request<ProviderConfig[]>('/api/v1/provider-configs'),
    refetchInterval: 5000,
  })
  const live = useQuery({
    queryKey: ['provider-models'],
    queryFn: api.listProviderModels,
    refetchInterval: 10000,
  })

  const [category, setCategory] = useState('all')
  const [templateId, setTemplateId] = useState('kimchi')
  const [name, setName] = useState('')
  const [baseUrl, setBaseUrl] = useState('')
  const [defaultModel, setDefaultModel] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [local, setLocal] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [okMsg, setOkMsg] = useState<string | null>(null)

  const allTemplates = templates.data ?? []
  const filtered = useMemo(() => {
    if (category === 'all') return allTemplates
    return allTemplates.filter((t) => (t.category || 'other') === category)
  }, [allTemplates, category])

  const selectedTpl = useMemo(
    () => allTemplates.find((t) => t.id === templateId),
    [allTemplates, templateId],
  )

  // Prefill when template list loads or selection changes
  useEffect(() => {
    if (!selectedTpl) return
    setBaseUrl(selectedTpl.baseUrl)
    setDefaultModel(selectedTpl.defaultModel || '')
    setName(selectedTpl.name)
    setLocal(selectedTpl.local)
  }, [selectedTpl?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  const applyTemplate = (t: Template) => {
    setTemplateId(t.id)
    setName(t.name)
    setBaseUrl(t.baseUrl)
    setDefaultModel(t.defaultModel || '')
    setLocal(t.local)
    setError(null)
    setOkMsg(null)
  }

  const create = useMutation({
    mutationFn: () => {
      if (!baseUrl.trim() && selectedTpl?.driver !== 'echo-provider') {
        throw new Error('Base URL is required')
      }
      return request<ProviderConfig>('/api/v1/provider-configs', {
        method: 'POST',
        body: JSON.stringify({
          fromTemplate: templateId,
          name: name.trim() || selectedTpl?.name || 'Custom provider',
          baseUrl: baseUrl.trim() || selectedTpl?.baseUrl,
          defaultModel: defaultModel.trim() || selectedTpl?.defaultModel,
          apiKey: apiKey.trim() || undefined,
          local,
          costTier: selectedTpl?.costTier || 'standard',
          driver: selectedTpl?.driver || 'openai-compat',
        }),
      })
    },
    onSuccess: (cfg) => {
      setOkMsg(`Added ${cfg.name} (${cfg.id})`)
      setError(null)
      setApiKey('')
      void qc.invalidateQueries({ queryKey: ['provider-configs'] })
      void qc.invalidateQueries({ queryKey: ['provider-models'] })
      void qc.invalidateQueries({ queryKey: ['registry'] })
    },
    onError: (e: Error) => {
      setError(e.message)
      setOkMsg(null)
    },
  })

  const del = useMutation({
    mutationFn: (id: string) =>
      request<{ deleted: string }>(`/api/v1/provider-configs/${encodeURIComponent(id)}`, {
        method: 'DELETE',
      }),
    onSuccess: (_d, id) => {
      setOkMsg(`Deleted ${id}`)
      setError(null)
      void qc.invalidateQueries({ queryKey: ['provider-configs'] })
      void qc.invalidateQueries({ queryKey: ['provider-models'] })
      void qc.invalidateQueries({ queryKey: ['registry'] })
    },
    onError: (e: Error) => setError(e.message),
  })

  const list = configs.data ?? []
  const healthById = new Map((live.data ?? []).map((p) => [p.id, p]))

  return (
    <div className="page">
      <h1 className="page-title">Providers</h1>
      <p className="page-sub">
        Add or remove LLM providers from the UI. Pick a popular template (base URL prefilled) or
        Custom. API keys are stored as credential handles — never shown back.
      </p>

      <section className="card p-4 mt-5 space-y-4">
        <div className="section-label">Popular templates</div>
        <div className="flex flex-wrap gap-2">
          {CATEGORIES.map((c) => (
            <button
              key={c.id}
              type="button"
              className={cn(
                'chip cursor-pointer border border-transparent',
                category === c.id && 'chip-live border-[rgba(0,191,255,0.4)]',
              )}
              onClick={() => setCategory(c.id)}
            >
              {c.label}
            </button>
          ))}
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2 max-h-64 overflow-y-auto pr-1">
          {filtered.map((t) => (
            <button
              key={t.id}
              type="button"
              onClick={() => applyTemplate(t)}
              className={cn(
                'text-left rounded-[var(--radius-control)] border px-3 py-2 transition-colors',
                templateId === t.id
                  ? 'border-[rgba(0,191,255,0.55)] bg-[var(--accent-dim)]'
                  : 'border-[var(--line)] bg-[var(--bg-1)] hover:border-[rgba(0,191,255,0.35)]',
              )}
            >
              <div className="font-medium text-sm flex items-center gap-2">
                {t.name}
                {t.local && <span className="chip chip-ok !text-[0.6rem]">local</span>}
              </div>
              <div className="font-mono text-[0.65rem] text-[var(--ink-2)] truncate mt-0.5">
                {t.baseUrl || '(in-process)'}
              </div>
              {t.description && (
                <div className="text-[0.7rem] text-[var(--ink-2)] mt-1 line-clamp-2">{t.description}</div>
              )}
            </button>
          ))}
          {filtered.length === 0 && (
            <p className="text-sm text-[var(--ink-2)] col-span-full">No templates in this category.</p>
          )}
        </div>
      </section>

      <section className="card p-4 mt-4 max-w-3xl space-y-3">
        <div className="section-label">Configure &amp; add</div>
        {selectedTpl && (
          <p className="text-xs text-[var(--ink-2)]">
            Using <span className="font-mono text-[var(--cyan-100)]">{selectedTpl.id}</span>
            {selectedTpl.docsUrl && (
              <>
                {' · '}
                <a
                  className="underline text-[var(--cyan-100)]"
                  href={selectedTpl.docsUrl}
                  target="_blank"
                  rel="noreferrer"
                >
                  docs
                </a>
              </>
            )}
          </p>
        )}
        <label className="block text-sm">
          <span className="text-[var(--ink-2)] text-xs">Display name</span>
          <input
            className="input mt-1"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="My OpenAI"
          />
        </label>
        <label className="block text-sm">
          <span className="text-[var(--ink-2)] text-xs">Base URL</span>
          <input
            className="input mt-1 font-mono text-sm"
            value={baseUrl}
            onChange={(e) => setBaseUrl(e.target.value)}
            placeholder="https://api.openai.com/v1"
          />
        </label>
        <label className="block text-sm">
          <span className="text-[var(--ink-2)] text-xs">Default model (optional)</span>
          <input
            className="input mt-1 font-mono text-sm"
            value={defaultModel}
            onChange={(e) => setDefaultModel(e.target.value)}
            placeholder={selectedTpl?.defaultModel || 'gpt-4o-mini'}
            list="model-suggestions"
          />
          <datalist id="model-suggestions">
            {(selectedTpl?.suggestedModels ?? []).map((m) => (
              <option key={m} value={m} />
            ))}
          </datalist>
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={local} onChange={(e) => setLocal(e.target.checked)} />
          Prefer as local (free / on-prem tier routing)
        </label>
        <label className="block text-sm">
          <span className="text-[var(--ink-2)] text-xs">
            API key {selectedTpl?.needsApiKey ? '(recommended)' : '(optional for local)'} — stored as
            handle only
          </span>
          <input
            className="input mt-1 font-mono text-sm"
            type="password"
            autoComplete="off"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder={selectedTpl?.needsApiKey ? 'sk-…' : '(not required for local)'}
          />
        </label>
        {error && <p className="text-sm text-[var(--fail)]">{error}</p>}
        {okMsg && <p className="text-sm text-[var(--ok)]">{okMsg}</p>}
        <button
          type="button"
          className="btn btn-primary"
          disabled={create.isPending}
          onClick={() => create.mutate()}
        >
          {create.isPending ? 'Adding…' : 'Add provider'}
        </button>
      </section>

      <section className="card mt-6 overflow-hidden">
        <div className="px-4 py-3 border-b border-[var(--line)] section-label !mb-0">
          Configured providers ({list.length})
        </div>
        <ul className="divide-y divide-[var(--line)]">
          {list.map((c) => {
            const liveInfo = healthById.get(c.id)
            return (
              <li
                key={c.id}
                className="px-4 py-3 flex flex-col sm:flex-row sm:items-center gap-2 justify-between"
              >
                <div className="min-w-0">
                  <div className="font-medium flex flex-wrap items-center gap-2">
                    {c.name}
                    {c.managed ? (
                      <span className="chip chip-live">managed</span>
                    ) : (
                      <span className="chip">bootstrap</span>
                    )}
                    {c.local && <span className="chip chip-ok">local</span>}
                    <span className="chip">{c.costTier || '—'}</span>
                  </div>
                  <div className="font-mono text-xs text-[var(--ink-2)] truncate">
                    {c.id}
                    {c.baseUrl ? ` · ${c.baseUrl}` : ''}
                  </div>
                  <div className="text-xs text-[var(--ink-2)] mt-0.5">
                    health: {liveInfo?.health ?? '—'} · models:{' '}
                    {liveInfo?.models?.length ?? c.models?.length ?? 0}
                    {c.credentialHandle ? ` · cred ${c.credentialHandle.slice(0, 12)}…` : ''}
                    {c.templateId ? ` · template ${c.templateId}` : ''}
                  </div>
                </div>
                <div className="flex gap-2 shrink-0">
                  {c.managed ? (
                    <button
                      type="button"
                      className="btn btn-danger"
                      disabled={del.isPending}
                      onClick={() => {
                        if (confirm(`Delete provider ${c.name} (${c.id})?`)) del.mutate(c.id)
                      }}
                    >
                      Delete
                    </button>
                  ) : (
                    <span className="text-xs text-[var(--ink-2)] self-center">
                      seed / disk — not deletable
                    </span>
                  )}
                </div>
              </li>
            )
          })}
          {list.length === 0 && (
            <li className="p-4 text-sm text-[var(--ink-2)]">No providers tracked yet.</li>
          )}
        </ul>
      </section>
    </div>
  )
}
