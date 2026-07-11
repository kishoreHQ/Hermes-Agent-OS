import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AppShell } from '@/components/AppShell'
import { DashboardPage } from '@/pages/DashboardPage'
import { MissionsPage } from '@/pages/MissionsPage'
import { MissionDetailPage } from '@/pages/MissionDetailPage'
import { FleetPage } from '@/pages/FleetPage'
import { MemoryPage } from '@/pages/MemoryPage'
import { EventsPage } from '@/pages/EventsPage'
import { CredentialsPage } from '@/pages/CredentialsPage'

const qc = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 1000,
    },
  },
})

export default function App() {
  return (
    <QueryClientProvider client={qc}>
      <BrowserRouter>
        <Routes>
          <Route element={<AppShell />}>
            <Route index element={<DashboardPage />} />
            <Route path="missions" element={<MissionsPage />} />
            <Route path="missions/:id" element={<MissionDetailPage />} />
            <Route path="fleet" element={<FleetPage />} />
            <Route path="memory" element={<MemoryPage />} />
            <Route path="events" element={<EventsPage />} />
            <Route path="credentials" element={<CredentialsPage />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
