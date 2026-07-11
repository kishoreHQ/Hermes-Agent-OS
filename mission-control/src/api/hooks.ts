import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './client'

export function useHealth() {
  return useQuery({
    queryKey: ['health'],
    queryFn: api.health,
    refetchInterval: 5000,
  })
}

export function useMissions(state?: string) {
  return useQuery({
    queryKey: ['missions', state ?? 'all'],
    queryFn: () => api.listMissions(state),
    refetchInterval: 3000,
  })
}

export function useMission(id: string | undefined) {
  return useQuery({
    queryKey: ['mission', id],
    queryFn: () => api.getMission(id!),
    enabled: !!id,
    refetchInterval: 2000,
  })
}

export function useCreateMission() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.createMission,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['missions'] })
      void qc.invalidateQueries({ queryKey: ['events'] })
      void qc.invalidateQueries({ queryKey: ['memory'] })
      void qc.invalidateQueries({ queryKey: ['credentials'] })
    },
  })
}

export function useCancelMission() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason?: string }) =>
      api.cancelMission(id, reason),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({ queryKey: ['missions'] })
      void qc.invalidateQueries({ queryKey: ['mission', vars.id] })
    },
  })
}

export function useEvents(since = 0) {
  return useQuery({
    queryKey: ['events', since],
    queryFn: () => api.events(since),
    refetchInterval: 2500,
  })
}

export function useReplay(id: string | undefined) {
  return useQuery({
    queryKey: ['replay', id],
    queryFn: () => api.replay(id!),
    enabled: !!id,
  })
}

export function useRegistry(kind: 'providers' | 'runtimes' | 'tools') {
  return useQuery({
    queryKey: ['registry', kind],
    queryFn: () => api.registry(kind),
    refetchInterval: 10000,
  })
}

export function useMemory(q?: string) {
  return useQuery({
    queryKey: ['memory', q ?? ''],
    queryFn: () => api.memorySearch({ q: q || undefined }),
    refetchInterval: 5000,
  })
}

export function useCredentials() {
  return useQuery({
    queryKey: ['credentials'],
    queryFn: api.credentials,
    refetchInterval: 8000,
  })
}
